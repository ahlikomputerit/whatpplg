package antiban

import (
	"math/rand/v2"
	"sync"
	"time"
)

// CBState represents the state of a circuit breaker entry.
type CBState int

const (
	CBClosed   CBState = iota
	CBOpen
	CBHalfOpen
)

type breakerEntry struct {
	state     CBState
	failures  int
	lastFail  time.Time
	lastOpen  time.Time
}

// JidCircuitBreaker prevents sending to JIDs that have experienced repeated failures.
type JidCircuitBreaker struct {
	mu sync.Mutex

	cfg       *Config
	breakers  map[string]*breakerEntry
}

// NewJidCircuitBreaker creates a new JID circuit breaker.
func NewJidCircuitBreaker(cfg *Config) *JidCircuitBreaker {
	return &JidCircuitBreaker{
		cfg:      cfg,
		breakers: make(map[string]*breakerEntry),
	}
}

// CanSend checks if the circuit breaker allows sending to the given JID.
func (jcb *JidCircuitBreaker) CanSend(jid string) bool {
	jcb.mu.Lock()
	defer jcb.mu.Unlock()

	entry, ok := jcb.breakers[jid]
	if !ok {
		return true
	}

	jcb.evictStale()

	switch entry.state {
	case CBClosed:
		return true
	case CBOpen:
		if time.Since(entry.lastOpen) >= jcb.cfg.CircuitBreakerCooldown {
			entry.state = CBHalfOpen
			return true
		}
		return false
	case CBHalfOpen:
		return true
	default:
		return true
	}
}

// RecordSuccess resets the circuit breaker for the given JID after a successful send.
func (jcb *JidCircuitBreaker) RecordSuccess(jid string) {
	jcb.mu.Lock()
	defer jcb.mu.Unlock()

	entry, ok := jcb.breakers[jid]
	if !ok {
		return
	}

	entry.state = CBClosed
	entry.failures = 0
}

// RecordFailure increments the failure count and opens the circuit if threshold is exceeded.
func (jcb *JidCircuitBreaker) RecordFailure(jid string) {
	jcb.mu.Lock()
	defer jcb.mu.Unlock()

	entry, ok := jcb.breakers[jid]
	if !ok {
		entry = &breakerEntry{state: CBClosed}
		jcb.breakers[jid] = entry
	}

	entry.failures++
	entry.lastFail = time.Now()

	if entry.failures >= jcb.cfg.CircuitBreakerThreshold {
		entry.state = CBOpen
		entry.lastOpen = time.Now()
	}
}

func (jcb *JidCircuitBreaker) evictStale() {
	for jid, entry := range jcb.breakers {
		if entry.state == CBClosed && time.Since(entry.lastFail) > 10*time.Minute {
			delete(jcb.breakers, jid)
		}
	}
}

// GetJitter returns a jittered cooldown duration for a JID whose circuit is open.
func (jcb *JidCircuitBreaker) GetJitter(jid string) time.Duration {
	if !jcb.CanSend(jid) {
		base := jcb.cfg.CircuitBreakerCooldown
		jitter := time.Duration(rand.Int64N(int64(base) / 4))
		return base + jitter
	}
	return 0
}

// GetStats returns circuit breaker statistics for monitoring.
func (jcb *JidCircuitBreaker) GetStats() map[string]any {
	jcb.mu.Lock()
	defer jcb.mu.Unlock()

	open := 0
	closed := 0
	halfOpen := 0
	for _, entry := range jcb.breakers {
		switch entry.state {
		case CBClosed:
			closed++
		case CBOpen:
			open++
		case CBHalfOpen:
			halfOpen++
		}
	}

	return map[string]any{
		"total":    len(jcb.breakers),
		"closed":   closed,
		"open":     open,
		"halfopen": halfOpen,
	}
}
