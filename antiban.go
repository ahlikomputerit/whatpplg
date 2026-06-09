// Package antiban provides an anti-ban system for WhatsApp messaging via whatsmeow.
// It orchestrates rate limiting, warm-up, health monitoring, circuit breaking,
// content variation, and ban recovery to reduce the risk of account restrictions.
package antiban

import (
	"sync"
	"sync/atomic"
	"time"
)

// AntiBan is the main orchestrator that coordinates all anti-ban subsystems.
// It provides BeforeSend/AfterSend hooks and manages rate limiting, warm-up,
// health monitoring, circuit breakers, and ban recovery.
type AntiBan struct {
	mu sync.Mutex

	cfg    Config
	preset Preset

	RateLimiter       *RateLimiter
	WarmUp            *WarmUp
	Health            *HealthMonitor
	TimelockGuard     *TimelockGuard
	ReconnectThrottle *PostReconnectThrottle
	RetryTracker      *RetryReasonTracker
	DeliveryTracker   *DeliveryTracker
	ContactGraph      *ContactGraphWarmer
	JidCanonicalizer  *JidCanonicalizer
	LidResolver       *LidResolver
	CircuitBreaker    *JidCircuitBreaker
	GroupGuard        *GroupOperationGuard
	Scheduler         *Scheduler
	ContentVariator   *ContentVariator
	BanRecovery       *BanRecoveryOrchestrator
	StateManager      *StateManager

	isPaused atomic.Bool
	destroyed atomic.Bool
}

// New creates a new AntiBan instance with the given preset and optional config overrides.
func New(preset Preset, overrides ...Config) *AntiBan {
	cfg := DefaultConfig(preset)
	if len(overrides) > 0 {
		o := overrides[0]
		cfg.Preset = preset
		cfg.applyPreset()
		if o.MaxPerMinute > 0 {
			cfg.MaxPerMinute = o.MaxPerMinute
		}
		if o.MaxPerHour > 0 {
			cfg.MaxPerHour = o.MaxPerHour
		}
		if o.MaxPerDay > 0 {
			cfg.MaxPerDay = o.MaxPerDay
		}
		if o.MinDelayMs > 0 {
			cfg.MinDelayMs = o.MinDelayMs
		}
		if o.MaxDelayMs > 0 {
			cfg.MaxDelayMs = o.MaxDelayMs
		}
		if o.NewChatDelayMs > 0 {
			cfg.NewChatDelayMs = o.NewChatDelayMs
		}
		if o.WarmUpDays > 0 {
			cfg.WarmUpDays = o.WarmUpDays
		}
	}

	resolver := NewLidResolver(1000)

	ab := &AntiBan{
		cfg:    cfg,
		preset: preset,

		RateLimiter:       NewRateLimiter(&cfg),
		WarmUp:            NewWarmUp(&cfg),
		Health:            NewHealthMonitor(&cfg),
		TimelockGuard:     NewTimelockGuard(&cfg),
		ReconnectThrottle: NewPostReconnectThrottle(&cfg),
		RetryTracker:      NewRetryReasonTracker(5),
		DeliveryTracker:   NewDeliveryTracker(&cfg),
		ContactGraph:      NewContactGraphWarmer(&cfg),
		JidCanonicalizer:  NewJidCanonicalizer(resolver),
		LidResolver:       resolver,
		CircuitBreaker:    NewJidCircuitBreaker(&cfg),
		GroupGuard:        NewGroupOperationGuard(&cfg),
		Scheduler:         NewScheduler(&cfg),
		ContentVariator:   NewContentVariator(&cfg),
		BanRecovery:       NewBanRecoveryOrchestrator(&cfg),
		StateManager:      NewStateManager(&cfg),
	}

	ab.Health.OnRiskChange(func(level RiskLevel) {
		if level == RiskCritical || level == RiskHigh {
			ab.isPaused.Store(true)
		} else {
			ab.isPaused.Store(false)
		}
	})

	return ab
}

// BeforeSend checks all anti-ban policies and returns whether the message is allowed
// and the recommended delay before sending. Called before each outbound message.
func (ab *AntiBan) BeforeSend(chatID string, content []byte) (time.Duration, bool) {
	if ab.destroyed.Load() {
		return 0, false
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.isPaused.Load() {
		return 0, false
	}

	if !ab.Scheduler.IsActiveTime() {
		return ab.Scheduler.MsUntilActive(), false
	}

	if !ab.WarmUp.CanSend() {
		return time.Minute, false
	}

	if !ab.RateLimiter.CanSend() {
		return time.Duration(ab.cfg.MinDelayMs) * time.Millisecond, false
	}

	if !ab.CircuitBreaker.CanSend(chatID) {
		return ab.CircuitBreaker.GetJitter(chatID), false
	}

	if !ab.TimelockGuard.CanSend(chatID) {
		return 30 * time.Second, false
	}

	if !ab.ContactGraph.CanMessage(chatID, IsGroup(chatID)) {
		return time.Hour, false
	}

	delay := ab.ReconnectThrottle.BeforeSend()

	rateDelay := ab.RateLimiter.GetDelay(chatID, content)
	if rateDelay > delay {
		delay = rateDelay
	}

	if !IsGroup(chatID) {
		canonJID := ab.JidCanonicalizer.CanonicalizeTarget(chatID)
		if canonJID != chatID {
			chatID = canonJID
		}
	}

	return ab.Scheduler.AdjustDelay(delay), true
}

// AfterSend records metrics after a successful send attempt.
func (ab *AntiBan) AfterSend(chatID string, success bool) {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.RateLimiter.Record(chatID)
	ab.WarmUp.Record()
	ab.ContactGraph.RecordMessage(chatID)
	ab.CircuitBreaker.RecordSuccess(chatID)

	if success {
		ab.DeliveryTracker.OnMessageSent()
	}
}

// AfterSendFailed records a failed send, updates health and circuit breaker,
// and classifies the error for ban recovery.
func (ab *AntiBan) AfterSendFailed(chatID string, err error) {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.Health.RecordMessageFailed()
	ab.CircuitBreaker.RecordFailure(chatID)

	banType := ClassifyError(err)
	if banType != "" {
		ab.BanRecovery.RecordBanEvent(BanEvent{
			Type:      banType,
			Timestamp: time.Now(),
		})
		mult := ab.BanRecovery.GetRateMultiplier()
		if mult < 1.0 {
			ab.RateLimiter.AdaptLimits(mult)
		}
	}
}

// OnDisconnect notifies the anti-ban system of a disconnection event.
func (ab *AntiBan) OnDisconnect() {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.Health.RecordDisconnect()
	ab.ReconnectThrottle.OnDisconnect()
}

// OnReconnect notifies the anti-ban system of a successful reconnection.
func (ab *AntiBan) OnReconnect() {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.Health.RecordReconnect()
	ab.ReconnectThrottle.OnReconnect()
	ab.RateLimiter.AdaptLimits(ab.BanRecovery.GetRateMultiplier())
}

// OnIncomingMessage registers an incoming message to update contact graph and JID mappings.
func (ab *AntiBan) OnIncomingMessage(senderID string) {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.ContactGraph.OnIncomingMessage(senderID)
	ab.JidCanonicalizer.OnIncomingEvent(senderID, "")
}

// OnDeliveryReceipt records a delivery receipt for tracking delivery rates.
func (ab *AntiBan) OnDeliveryReceipt() {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.DeliveryTracker.OnDeliveryReceipt()
}

// OnStreamError handles stream errors by classifying them and triggering ban recovery.
func (ab *AntiBan) OnStreamError(err error) {
	if ab.destroyed.Load() {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	banType := ClassifyError(err)
	switch banType {
	case BanHard:
		ab.Health.RecordLoggedOut()
	case BanSoft:
		ab.Health.RecordForbidden()
	case BanTimelock:
		ab.Health.RecordReachoutTimelock()
		ab.TimelockGuard.Record463Error()
	}

	if banType != "" {
		ab.BanRecovery.RecordBanEvent(BanEvent{
			Type:      banType,
			Timestamp: time.Now(),
		})
	}
}

// Pause stops all outbound messaging until Resume is called.
func (ab *AntiBan) Pause() {
	ab.isPaused.Store(true)
}

// Resume re-enables outbound messaging after a pause.
func (ab *AntiBan) Resume() {
	ab.isPaused.Store(false)
}

// IsPaused returns whether the anti-ban system has paused outbound messaging.
func (ab *AntiBan) IsPaused() bool {
	return ab.isPaused.Load()
}

// Destroy cleans up all anti-ban resources and goroutines.
func (ab *AntiBan) Destroy() {
	ab.destroyed.Store(true)
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.ReconnectThrottle.Destroy()
	ab.RetryTracker.Destroy()
}

// GetStats returns a snapshot of all subsystem stats for monitoring.
func (ab *AntiBan) GetStats() map[string]any {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return map[string]any{
		"preset":            ab.preset,
		"paused":            ab.isPaused.Load(),
		"rate_limiter":      ab.RateLimiter.GetStats(),
		"warmup":            ab.WarmUp.GetStatus(),
		"health":            ab.Health.GetStatus(),
		"reconnect":         ab.ReconnectThrottle.GetStats(),
		"retry":             ab.RetryTracker.GetStats(),
		"delivery":          ab.DeliveryTracker.GetStats(),
		"circuit_breaker":   ab.CircuitBreaker.GetStats(),
		"timelock":          ab.TimelockGuard.GetState(),
		"recovery":          ab.BanRecovery.GetStatus(),
		"scheduler":         ab.Scheduler.GetStatus(),
	}
}
