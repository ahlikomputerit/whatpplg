package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	qrcode "github.com/skip2/go-qrcode"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	antiban "github.com/ahlikomputerit/whatpplg"
)

func main() {
	dbPath := flag.String("db", "whatsmeow.db", "SQLite database path")
	preset := flag.String("preset", "moderate", "preset: conservative, moderate, aggressive, high-volume")
	flag.Parse()

	log := waLog.Stdout("Main", "DEBUG", true)

	container, err := sqlstore.New(context.Background(), "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", *dbPath), log.Sub("DB"))
	if err != nil {
		log.Errorf("Failed to open database: %v", err)
		os.Exit(1)
	}

	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		os.Exit(1)
	}

	client := whatsmeow.NewClient(device, log.Sub("WA"))
	client.EnableAutoReconnect = true

	var presetEnum antiban.Preset
	switch *preset {
	case "conservative":
		presetEnum = antiban.PresetConservative
	case "aggressive":
		presetEnum = antiban.PresetAggressive
	case "high-volume":
		presetEnum = antiban.PresetHighVolume
	default:
		presetEnum = antiban.PresetModerate
	}

	abc := antiban.WrapClient(client, presetEnum, antiban.Config{
		EnableTypoInjection:   true,
		EnableZeroWidth:       true,
		EnablePunctuationVary: true,
		GroupLurkPeriod:       5 * time.Minute,
		MaxStrangerPerDay:     10,
	})

	abc.WAClient.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.QR:
			code := v.Codes[0]
			fmt.Println()
			fmt.Println("========================================")
			fmt.Println("      SCAN QR CODE WhatsApp")
			fmt.Println("========================================")
			fmt.Println()
			fmt.Println("1. Buka WhatsApp di HP")
			fmt.Println("2. Settings > Linked Devices > Link a Device")
			fmt.Println("3. Scan QR CODE di bawah ini:")
			fmt.Println()

			qr, err := qrcode.New(code, qrcode.Medium)
			if err == nil {
				art := qr.ToString(true)
				for _, line := range strings.Split(art, "\n") {
					if strings.TrimSpace(line) != "" {
						fmt.Println(line)
					}
				}
				fmt.Println()
			}

			fmt.Println("   Atau buka link berikut di browser HP:")
			fmt.Println(code)
			fmt.Println()
			fmt.Println("========================================")
			fmt.Println()

		case *events.Message:
			if !v.Info.IsFromMe {
				chat := v.Info.Chat
				text := ""
				if v.Message.GetConversation() != "" {
					text = v.Message.GetConversation()
				} else if v.Message.GetExtendedTextMessage().GetText() != "" {
					text = v.Message.GetExtendedTextMessage().GetText()
				}
				log.Infof("Pesan dari %s: %s", chat, text)

				if text == "!ping" {
					go func() {
						resp, err := abc.SendMessage(context.Background(), chat, &waE2E.Message{
							Conversation: proto.String("Pong!"),
						})
						if err != nil {
							log.Errorf("Gagal: %v", err)
						} else {
							log.Infof("Terikirim (ID: %s)", resp.ID)
						}
					}()
				}

				if text == "!stats" {
					stats := abc.GetStats()
					log.Infof("AntiBan stats: %+v", stats)
				}
			}

		case *events.Connected:
			log.Infof("Terhubung! Login sebagai %s", abc.WAClient.Store.GetJID())
			fmt.Println()
			fmt.Println("✅ Terhubung! Kirim '!ping' untuk test atau '!stats' untuk statistik.")
			fmt.Println()

		case *events.Disconnected:
			log.Infof("Tatputus")

		case *events.LoggedOut:
			log.Infof("Logout")
		}
	})

	abc.Start(context.Background())

	hasSession := client.Store.ID != nil
	err = client.Connect()
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		os.Exit(1)
	}

	if !hasSession {
		log.Infof("Scan QR code to log in")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Infof("Shutting down...")
	abc.Stop()
	client.Disconnect()
}
