package antiban

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiter controls message throughput with per-minute/hour/day limits,
// burst allowance, and Gaussian-jittered delays. It also detects identical
// message content and applies extra delay when the limit is exceeded.
type RateLimiter struct {
	mu sync.Mutex

	cfg *Config

	perMinute   []time.Time
	perHour     []time.Time
	perDay      []time.Time
	burstCredit int

	knownChats     map[string]bool
	identicalCount map[string]int
	identicalTime  map[string]time.Time

	sent int64
}

// NewRateLimiter creates a new rate limiter from the given config.
func NewRateLimiter(cfg *Config) *RateLimiter {
	return &RateLimiter{
		cfg:            cfg,
		knownChats:     make(map[string]bool),
		identicalCount: make(map[string]int),
		identicalTime:  make(map[string]time.Time),
	}
}

func gaussianJitter(mean, stdDev float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + z*stdDev
}

func (rl *RateLimiter) contentHash(msg []byte) string {
	h := sha256.Sum256(msg)
	return fmt.Sprintf("%x", h[:8])
}

// GetDelay returns the waiting time before sending a message.
// Unknown chats get a higher base delay. Identical content beyond
// MaxIdenticalMessages doubles the delay.
func (rl *RateLimiter) GetDelay(chatID string, content []byte) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	_, known := rl.knownChats[chatID]
	baseDelayMs := float64(rl.cfg.MinDelayMs)
	if !known {
		baseDelayMs = float64(rl.cfg.NewChatDelayMs)
	}

	delayMs := gaussianJitter(baseDelayMs, baseDelayMs*0.3)
	if delayMs < float64(rl.cfg.MinDelayMs) {
		delayMs = float64(rl.cfg.MinDelayMs)
	}
	if delayMs > float64(rl.cfg.MaxDelayMs) {
		delayMs = float64(rl.cfg.MaxDelayMs)
	}

	if content != nil {
		hash := rl.contentHash(content)
		rl.identicalCount[hash]++
		rl.identicalTime[hash] = time.Now()
		if rl.identicalCount[hash] > rl.cfg.MaxIdenticalMessages {
			delayMs *= 2
		}
	}

	return time.Duration(delayMs) * time.Millisecond
}

// CanSend checks whether sending is allowed under current rate limits.
// It considers per-minute, per-hour, and per-day caps, plus burst credit.
func (rl *RateLimiter) CanSend() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.canSend()
}

func (rl *RateLimiter) canSend() bool {
	now := time.Now()
	cutoffMin := now.Add(-1 * time.Minute)
	cutoffHour := now.Add(-1 * time.Hour)
	cutoffDay := now.Add(-24 * time.Hour)

	rl.prune(&rl.perMinute, cutoffMin)
	rl.prune(&rl.perHour, cutoffHour)
	rl.prune(&rl.perDay, cutoffDay)

	if len(rl.perDay) >= rl.cfg.MaxPerDay {
		return false
	}
	if len(rl.perHour) >= rl.cfg.MaxPerHour {
		return false
	}
	if len(rl.perMinute) >= rl.cfg.MaxPerMinute {
		if rl.burstCredit < rl.cfg.BurstAllowance {
			rl.burstCredit++
			return true
		}
		return false
	}
	return true
}

func (rl *RateLimiter) prune(slice *[]time.Time, cutoff time.Time) {
	i := 0
	for _, t := range *slice {
		if t.After(cutoff) {
			(*slice)[i] = t
			i++
		}
	}
	*slice = (*slice)[:i]
}

// Record logs a sent message for rate limit tracking.
// Call this after a successful send to update counters.
func (rl *RateLimiter) Record(chatID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.perMinute = append(rl.perMinute, now)
	rl.perHour = append(rl.perHour, now)
	rl.perDay = append(rl.perDay, now)
	rl.knownChats[chatID] = true
	atomic.AddInt64(&rl.sent, 1)
}

// AdaptLimits scales all rate limits by the given factor (0.1-1.0).
// Used during ban recovery to reduce throughput gradually.
func (rl *RateLimiter) AdaptLimits(factor float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if factor < 0.1 {
		factor = 0.1
	}
	if factor > 1.0 {
		factor = 1.0
	}

	rl.cfg.MaxPerMinute = int(float64(rl.cfg.MaxPerMinute) * factor)
	rl.cfg.MaxPerHour = int(float64(rl.cfg.MaxPerHour) * factor)
	rl.cfg.MaxPerDay = int(float64(rl.cfg.MaxPerDay) * factor)
	rl.cfg.MinDelayMs = int(float64(rl.cfg.MinDelayMs) / factor)
	rl.cfg.MaxDelayMs = int(float64(rl.cfg.MaxDelayMs) / factor)
}

// GetStats returns current rate limiter counters for monitoring.
func (rl *RateLimiter) GetStats() map[string]any {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return map[string]any{
		"sent":         atomic.LoadInt64(&rl.sent),
		"known_chats":  len(rl.knownChats),
		"per_minute":   len(rl.perMinute),
		"per_hour":     len(rl.perHour),
		"per_day":      len(rl.perDay),
		"burst_credit": rl.burstCredit,
	}
}

// GetKnownChats returns a copy of the known chats map for persistence.
func (rl *RateLimiter) GetKnownChats() map[string]bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	chats := make(map[string]bool, len(rl.knownChats))
	for k, v := range rl.knownChats {
		chats[k] = v
	}
	return chats
}

// RestoreKnownChats loads previously persisted known chats into the limiter.
func (rl *RateLimiter) RestoreKnownChats(chats map[string]bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, v := range chats {
		rl.knownChats[k] = v
	}
}
