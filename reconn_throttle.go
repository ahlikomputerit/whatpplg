package antiban

import (
	"math"
	"sync"
	"time"
)

type PostReconnectThrottle struct {
	mu sync.Mutex

	cfg            *Config
	reconnectAt    time.Time
	disconnectAt   time.Time
	sentAfter      int
	steps          int
}

func NewPostReconnectThrottle(cfg *Config) *PostReconnectThrottle {
	return &PostReconnectThrottle{cfg: cfg}
}

func (prt *PostReconnectThrottle) OnReconnect() {
	prt.mu.Lock()
	defer prt.mu.Unlock()
	prt.reconnectAt = time.Now()
	prt.sentAfter = 0
}

func (prt *PostReconnectThrottle) OnDisconnect() {
	prt.mu.Lock()
	defer prt.mu.Unlock()
	prt.disconnectAt = time.Now()
}

func (prt *PostReconnectThrottle) BeforeSend() time.Duration {
	prt.mu.Lock()
	defer prt.mu.Unlock()

	if prt.reconnectAt.IsZero() {
		return 0
	}

	elapsed := time.Since(prt.reconnectAt)
	if elapsed >= prt.cfg.ReconnectRampDuration {
		prt.reconnectAt = time.Time{}
		prt.sentAfter = 0
		return 0
	}

	progress := elapsed.Seconds() / prt.cfg.ReconnectRampDuration.Seconds()
	multiplier := prt.cfg.ReconnectInitialRate + (1-prt.cfg.ReconnectInitialRate)*progress

	idealDelay := time.Duration(float64(prt.cfg.MinDelayMs)/multiplier) * time.Millisecond

	prt.sentAfter++
	if prt.sentAfter > prt.steps {
		prt.steps = prt.sentAfter
	}

	return idealDelay
}

func (prt *PostReconnectThrottle) GetCurrentMultiplier() float64 {
	prt.mu.Lock()
	defer prt.mu.Unlock()
	return prt.getCurrentMultiplier()
}

func (prt *PostReconnectThrottle) getCurrentMultiplier() float64 {
	if prt.reconnectAt.IsZero() {
		return 1.0
	}
	elapsed := time.Since(prt.reconnectAt)
	if elapsed >= prt.cfg.ReconnectRampDuration {
		return 1.0
	}
	progress := elapsed.Seconds() / prt.cfg.ReconnectRampDuration.Seconds()
	return prt.cfg.ReconnectInitialRate + (1-prt.cfg.ReconnectInitialRate)*progress
}

func (prt *PostReconnectThrottle) GetStats() map[string]any {
	prt.mu.Lock()
	defer prt.mu.Unlock()
	mult := prt.getCurrentMultiplier()
	return map[string]any{
		"multiplier":     math.Round(mult*100) / 100,
		"sent_after":     prt.sentAfter,
		"reconnected_s":  time.Since(prt.reconnectAt).Seconds(),
		"ramp_duration_s": prt.cfg.ReconnectRampDuration.Seconds(),
	}
}

func (prt *PostReconnectThrottle) Destroy() {
	prt.mu.Lock()
	defer prt.mu.Unlock()
	prt.reconnectAt = time.Time{}
	prt.disconnectAt = time.Time{}
	prt.sentAfter = 0
	prt.steps = 0
}
