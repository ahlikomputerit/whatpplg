package antiban

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// AntiBanClient wraps a whatsmeow.Client with anti-ban middleware.
// All outbound messages go through the anti-ban pipeline (rate limit,
// content variation, warmup, etc.) before being sent.
type AntiBanClient struct {
	WAClient *whatsmeow.Client
	AntiBan  *AntiBan
	Config   Config

	mu        sync.Mutex
	sendQueue chan *pendingMessage
	stopCh    chan struct{}
	started   bool
	log       waLog.Logger
}

type pendingMessage struct {
	ctx     context.Context
	to      types.JID
	message *waE2E.Message
	extra   []whatsmeow.SendRequestExtra
	respCh  chan sendResult
}

type sendResult struct {
	resp whatsmeow.SendResponse
	err  error
}

// WrapClient wraps a whatsmeow.Client with anti-ban middleware.
// The preset determines base rate limits and safety parameters.
// An optional Config can override specific preset values.
func WrapClient(wac *whatsmeow.Client, preset Preset, opts ...Config) *AntiBanClient {
	cfg := DefaultConfig(preset)
	if len(opts) > 0 {
		cfg = ResolveConfig(preset, opts[0])
	}

	log := wac.Log
	if log == nil {
		log = waLog.Noop
	}

	ab := New(preset, cfg)
	abc := &AntiBanClient{
		WAClient: wac,
		AntiBan:  ab,
		Config:   cfg,
		log:      log.Sub("AntiBan"),
	}

	wac.AddEventHandler(abc.handleEvent)

	return abc
}

// Start launches the send queue loop and restores persisted state.
// Must be called before Connect or SendMessage.
func (abc *AntiBanClient) Start(ctx context.Context) {
	abc.mu.Lock()
	defer abc.mu.Unlock()

	if abc.started {
		return
	}

	abc.sendQueue = make(chan *pendingMessage, 1000)
	abc.stopCh = make(chan struct{})
	abc.started = true

	go abc.sendLoop(ctx)

	state := abc.AntiBan.StateManager.GetPersistedState()
	if state.KnownChats != nil && len(state.KnownChats) > 0 {
		abc.AntiBan.RateLimiter.RestoreKnownChats(state.KnownChats)
	}

	abc.AntiBan.StateManager.StartAutoSave(30 * time.Second)
}

// Stop shuts down the send queue, saves state, and stops auto-save.
func (abc *AntiBanClient) Stop() {
	abc.mu.Lock()
	defer abc.mu.Unlock()

	if !abc.started {
		return
	}

	close(abc.stopCh)
	abc.started = false
	abc.AntiBan.StateManager.Stop()
	abc.saveState()
}

// SendMessage enqueues a message through the anti-ban pipeline.
// It blocks until the message is sent or the context is cancelled.
// The message passes through rate limiting, content variation, and delays.
func (abc *AntiBanClient) SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	respCh := make(chan sendResult, 1)

	select {
	case abc.sendQueue <- &pendingMessage{
		ctx:     ctx,
		to:      to,
		message: message,
		extra:   extra,
		respCh:  respCh,
	}:
	case <-ctx.Done():
		return whatsmeow.SendResponse{}, ctx.Err()
	}

	select {
	case result := <-respCh:
		return result.resp, result.err
	case <-ctx.Done():
		return whatsmeow.SendResponse{}, ctx.Err()
	}
}

func (abc *AntiBanClient) sendLoop(ctx context.Context) {
	for {
		select {
		case pm := <-abc.sendQueue:
			abc.processSend(pm)
		case <-abc.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (abc *AntiBanClient) processSend(pm *pendingMessage) {
	chatID := pm.to.String()
	var contentBytes []byte
	if pm.message != nil {
		if txt := pm.message.GetConversation(); txt != "" {
			contentBytes = []byte(txt)
		} else if txt := pm.message.GetExtendedTextMessage().GetText(); txt != "" {
			contentBytes = []byte(txt)
		}
	}

	if !pm.to.IsBot() && !IsGroup(chatID) {
		canonJID := abc.AntiBan.JidCanonicalizer.CanonicalizeTarget(chatID)
		if canonJID != chatID {
			parsed, err := types.ParseJID(canonJID)
			if err == nil {
				pm.to = parsed
				chatID = canonJID
			}
		}
	}

	delay, allowed := abc.AntiBan.BeforeSend(chatID, contentBytes)
	if !allowed {
		if delay > 0 {
			abc.log.Debugf("AntiBan blocked send to %s, retrying after %v", chatID, delay)
			time.Sleep(delay)
			delay, allowed = abc.AntiBan.BeforeSend(chatID, contentBytes)
			if !allowed {
				pm.respCh <- sendResult{err: fmt.Errorf("antiban: send blocked after retry")}
				return
			}
		} else {
			pm.respCh <- sendResult{err: fmt.Errorf("antiban: send blocked")}
			return
		}
	}

	if delay > 0 {
		abc.log.Debugf("AntiBan delaying send to %s by %v", chatID, delay)
		select {
		case <-time.After(delay):
		case <-pm.ctx.Done():
			pm.respCh <- sendResult{err: pm.ctx.Err()}
			return
		}
	}

	shouldVary := abc.Config.EnableTypoInjection || abc.Config.EnableZeroWidth || abc.Config.EnableEmojiPadding || abc.Config.EnablePunctuationVary
	if shouldVary && pm.message != nil && pm.message.GetConversation() != "" {
		original := pm.message.GetConversation()
		varied := abc.AntiBan.ContentVariator.Vary(original)
		if varied != original {
			pm.message.Conversation = proto.String(varied)
		}
	}

	resp, err := abc.WAClient.SendMessage(pm.ctx, pm.to, pm.message, pm.extra...)

	if err != nil {
		abc.AntiBan.AfterSendFailed(chatID, err)
		pm.respCh <- sendResult{err: err}
		return
	}

	abc.AntiBan.AfterSend(chatID, true)
	pm.respCh <- sendResult{resp: resp}
}

func (abc *AntiBanClient) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Disconnected:
		abc.AntiBan.OnDisconnect()
		abc.saveState()

	case *events.Connected:
		abc.AntiBan.OnReconnect()

	case *events.StreamError:
		err := fmt.Errorf("stream error: code=%s", v.Code)
		abc.AntiBan.OnStreamError(err)
		if v.Code == "515" {
			abc.AntiBan.OnDisconnect()
		}

	case *events.LoggedOut:
		abc.AntiBan.OnStreamError(fmt.Errorf("logged out: %v", v.Reason))
		abc.AntiBan.Health.RecordLoggedOut()

	case *events.TemporaryBan:
		err := fmt.Errorf("temporary ban: %v (expires in %v)", v.Code, v.Expire)
		abc.AntiBan.OnStreamError(err)
		abc.AntiBan.Pause()
		time.AfterFunc(v.Expire, func() {
			abc.AntiBan.Resume()
		})

	case *events.Message:
		if !v.Info.IsFromMe {
			sender := v.Info.Sender.String()
			if sender == "" {
				sender = v.Info.Chat.String()
			}
			abc.AntiBan.OnIncomingMessage(sender)
			if v.Info.Chat.Server == types.DefaultUserServer || v.Info.Chat.Server == types.HiddenUserServer {
				if v.Info.Sender.String() != "" {
					senderStr := v.Info.Sender.String()
					if !strings.Contains(senderStr, "@lid") {
						chatStr := v.Info.Chat.String()
						if strings.Contains(chatStr, "@lid") || strings.Contains(chatStr, "@s.whatsapp.net") {
							abc.AntiBan.ContactGraph.RegisterKnownContact(senderStr)
						}
					}
				}
			}
			abc.AntiBan.ContactGraph.RegisterKnownContact(v.Info.Sender.ToNonAD().String())
		}

	case *events.Receipt:
		if v.Type == types.ReceiptTypeDelivered || v.Type == types.ReceiptTypeRead {
			abc.AntiBan.OnDeliveryReceipt()
		}

	case *events.IdentityChange:
		abc.log.Debugf("Identity change detected for %s", v.JID)

	case *events.ConnectFailure:
		err := fmt.Errorf("connect failure: %s", v.Reason)
		abc.AntiBan.OnStreamError(err)

	case *events.OfflineSyncCompleted:
		abc.log.Debugf("Offline sync completed: %d messages", v.Count)
	}
}

func (abc *AntiBanClient) saveState() {
	warmupState := abc.AntiBan.WarmUp.ExportState()
	day, _ := warmupState["day"].(int)
	dc, _ := warmupState["daily_count"].(int)
	gf, _ := warmupState["growth_factor"].(float64)
	sa, _ := warmupState["started_at"].(time.Time)

	abc.AntiBan.StateManager.SaveWarmupState(day, dc, gf, sa)
	abc.AntiBan.StateManager.SaveKnownChats(abc.AntiBan.RateLimiter.GetKnownChats())
}

// GetStats returns a snapshot of all anti-ban module statistics.
func (abc *AntiBanClient) GetStats() map[string]any {
	return abc.AntiBan.GetStats()
}

// GetWAClient returns the underlying whatsmeow.Client instance.
func (abc *AntiBanClient) GetWAClient() *whatsmeow.Client {
	return abc.WAClient
}
