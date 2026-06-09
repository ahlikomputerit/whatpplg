package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	antiban "github.com/ahlikomputerit/whatpplg"
)

type GatewayConfig struct {
	Server struct {
		Port   int    `yaml:"port"`
		APIKey string `yaml:"api_key"`
	} `yaml:"server"`
	Whatsapp struct {
		DBPath string `yaml:"db_path"`
		Preset string `yaml:"preset"`
		Config struct {
			EnableTypoInjection   bool `yaml:"enable_typo_injection"`
			EnableZeroWidth       bool `yaml:"enable_zero_width"`
			EnableEmojiPadding    bool `yaml:"enable_emoji_padding"`
			EnablePunctuationVary bool `yaml:"enable_punctuation_vary"`
		} `yaml:"config"`
	} `yaml:"whatsapp"`
	Sources   []SourceConfig   `yaml:"sources"`
	Templates []TemplateConfig `yaml:"templates"`
	Queue     struct {
		Type    string `yaml:"type"`
		MaxSize int    `yaml:"max_size"`
	} `yaml:"queue"`
}

type SourceConfig struct {
	Name             string   `yaml:"name"`
	Mode             string   `yaml:"mode"`
	APIKey           string   `yaml:"api_key"`
	AllowedTemplates []string `yaml:"allowed_templates,omitempty"`
}

type TemplateConfig struct {
	Name string `yaml:"name"`
	Body string `yaml:"body"`
}

type Gateway struct {
	cfg       *GatewayConfig
	abc       *antiban.AntiBanClient
	client    *whatsmeow.Client
	log       waLog.Logger
	templates map[string]string
	qrChan    chan string
}

func readConfig(path string) *GatewayConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config %s: %v", path, err)
	}
	var cfg GatewayConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	return &cfg
}

func resolvePreset(name string) antiban.Preset {
	switch strings.ToLower(name) {
	case "conservative":
		return antiban.PresetConservative
	case "aggressive":
		return antiban.PresetAggressive
	case "high-volume":
		return antiban.PresetHighVolume
	default:
		return antiban.PresetModerate
	}
}

func buildAntibanConfig(gcfg *GatewayConfig) antiban.Config {
	preset := resolvePreset(gcfg.Whatsapp.Preset)
	ac := antiban.DefaultConfig(preset)
	ac.EnableTypoInjection = gcfg.Whatsapp.Config.EnableTypoInjection
	ac.EnableZeroWidth = gcfg.Whatsapp.Config.EnableZeroWidth
	ac.EnableEmojiPadding = gcfg.Whatsapp.Config.EnableEmojiPadding
	ac.EnablePunctuationVary = gcfg.Whatsapp.Config.EnablePunctuationVary
	return ac
}

func validateAPIKey(expected, provided string) bool {
	return expected == "" || expected == provided
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	gcfg := readConfig(*configPath)

	appLog := waLog.Stdout("Gateway", "DEBUG", true)
	appLog.Infof("Starting WA Gateway on port %d", gcfg.Server.Port)

	container, err := sqlstore.New(context.Background(), "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", gcfg.Whatsapp.DBPath), appLog.Sub("DB"))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	client := whatsmeow.NewClient(device, appLog.Sub("WA"))
	client.EnableAutoReconnect = true

	ac := buildAntibanConfig(gcfg)
	abc := antiban.WrapClient(client, resolvePreset(gcfg.Whatsapp.Preset), ac)

	templates := make(map[string]string)
	for _, t := range gcfg.Templates {
		templates[t.Name] = t.Body
	}

	gw := &Gateway{
		cfg:       gcfg,
		abc:       abc,
		client:    client,
		log:       appLog,
		templates: templates,
		qrChan:    make(chan string, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	abc.Start(ctx)

	client.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.QR:
			code := v.Codes[0]
			gw.qrChan <- code
			fmt.Println()
			fmt.Println("========================================")
			fmt.Println("      SCAN QR CODE WhatsApp")
			fmt.Println("========================================")
			fmt.Println(code)
			fmt.Println("========================================")
			fmt.Println()

		case *events.Connected:
			appLog.Infof("Connected as %s", client.Store.GetJID())

		case *events.Disconnected:
			appLog.Infof("Disconnected")

		case *events.LoggedOut:
			appLog.Infof("Logged out")
		}
	})

	hasSession := client.Store.ID != nil
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	if !hasSession {
		appLog.Infof("Waiting for QR scan...")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", gw.handleHealth)
	mux.HandleFunc("/api/v1/send", gw.handleSend)
	mux.HandleFunc("/api/v1/send-template", gw.handleSendTemplate)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", gcfg.Server.Port),
		Handler: mux,
	}

	go func() {
		appLog.Infof("HTTP server listening on :%d", gcfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	appLog.Infof("Shutting down...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	srv.Shutdown(ctxShutdown)
	abc.Stop()
	client.Disconnect()
}

func (gw *Gateway) authMiddleware(r *http.Request) bool {
	key := r.Header.Get("Authorization")
	if key == "" {
		key = r.URL.Query().Get("api_key")
	}
	if strings.HasPrefix(key, "Bearer ") {
		key = strings.TrimPrefix(key, "Bearer ")
	}
	return validateAPIKey(gw.cfg.Server.APIKey, key)
}

func (gw *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !gw.authMiddleware(r) {
		writeError(w, http.StatusUnauthorized, "invalid API key")
		return
	}
	stats := gw.abc.GetStats()
	connected := gw.client.IsConnected()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"connected": connected,
		"stats":     stats,
	})
}

type SendRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type SendTemplateRequest struct {
	To       string         `json:"to"`
	Template string         `json:"template"`
	Data     map[string]string `json:"data"`
}

func (gw *Gateway) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !gw.authMiddleware(r) {
		writeError(w, http.StatusUnauthorized, "invalid API key")
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.To == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "to and message are required")
		return
	}

	jid, err := types.ParseJID(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JID: %v", err))
		return
	}

	resp, err := gw.abc.SendMessage(r.Context(), jid, &waE2E.Message{
		Conversation: proto.String(req.Message),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "sent",
		"id":     resp.ID,
	})
}

func (gw *Gateway) handleSendTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !gw.authMiddleware(r) {
		writeError(w, http.StatusUnauthorized, "invalid API key")
		return
	}

	var req SendTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.To == "" || req.Template == "" {
		writeError(w, http.StatusBadRequest, "to and template are required")
		return
	}

	templateBody, ok := gw.templates[req.Template]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("template %q not found", req.Template))
		return
	}

	message := templateBody
	for k, v := range req.Data {
		message = strings.ReplaceAll(message, "{"+k+"}", v)
	}

	jid, err := types.ParseJID(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JID: %v", err))
		return
	}

	resp, err := gw.abc.SendMessage(r.Context(), jid, &waE2E.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "sent",
		"id":     resp.ID,
	})
}
