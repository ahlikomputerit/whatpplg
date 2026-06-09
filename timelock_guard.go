package antiban

import (
	"sync"
	"sync/atomic"
	"time"
)

type TimelockGuard struct {
	mu sync.Mutex

	cfg            *Config
	blockedUntil   time.Time
	knownChats     map[string]bool
	generation     atomic.Int64

	onTimelockDetected func(duration time.Duration)
	onTimelockLifted   func()
}

func NewTimelockGuard(cfg *Config) *TimelockGuard {
	return &TimelockGuard{
		cfg:        cfg,
		knownChats: make(map[string]bool),
	}
}

func (tg *TimelockGuard) OnTimelockDetected(fn func(time.Duration)) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.onTimelockDetected = fn
}

func (tg *TimelockGuard) OnTimelockLifted(fn func()) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.onTimelockLifted = fn
}

func (tg *TimelockGuard) Record463Error() {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	tg.generation.Add(1)
	blockDuration := tg.cfg.TimelockBlockDuration
	tg.blockedUntil = time.Now().Add(blockDuration)

	if tg.onTimelockDetected != nil {
		go tg.onTimelockDetected(blockDuration)
	}

	go tg.autoLift(blockDuration)
}

func (tg *TimelockGuard) autoLift(duration time.Duration) {
	gen := tg.generation.Load()
	time.Sleep(duration)

	if tg.generation.Load() != gen {
		return
	}

	tg.mu.Lock()
	if time.Now().After(tg.blockedUntil) {
		if tg.onTimelockLifted != nil {
			go tg.onTimelockLifted()
		}
	}
	tg.mu.Unlock()
}

func (tg *TimelockGuard) CanSend(chatID string) bool {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	if time.Now().Before(tg.blockedUntil) {
		_, known := tg.knownChats[chatID]
		if !known {
			return false
		}
	}
	return true
}

func (tg *TimelockGuard) RegisterKnownChat(chatID string) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.knownChats[chatID] = true
}

func (tg *TimelockGuard) GetState() map[string]any {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	return map[string]any{
		"blocked":         time.Now().Before(tg.blockedUntil),
		"blocked_until":   tg.blockedUntil,
		"remaining_sec":   time.Until(tg.blockedUntil).Seconds(),
		"known_chats":     len(tg.knownChats),
	}
}

func (tg *TimelockGuard) Lift() {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.generation.Add(1)
	tg.blockedUntil = time.Time{}
}

func (tg *TimelockGuard) Reset() {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.generation.Add(1)
	tg.blockedUntil = time.Time{}
	tg.knownChats = make(map[string]bool)
}
