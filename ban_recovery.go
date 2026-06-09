package antiban

import (
	"math"
	"sync"
	"time"
)

// BanEvent records a single ban-related event for recovery tracking.
type BanEvent struct {
	Type      BanType
	Timestamp time.Time
}

// BanRecoveryOrchestrator manages the phased recovery from ban events,
// gradually ramping up messaging activity after a ban.
type BanRecoveryOrchestrator struct {
	mu sync.Mutex

	cfg       *Config
	phase     RecoveryPhase
	banEvents []BanEvent
	rampDay   int
}

// NewBanRecoveryOrchestrator creates a new ban recovery orchestrator.
func NewBanRecoveryOrchestrator(cfg *Config) *BanRecoveryOrchestrator {
	return &BanRecoveryOrchestrator{
		cfg:   cfg,
		phase: PhaseGraduated,
	}
}

// ClassifyError maps an error to a BanType based on its error string contents.
func ClassifyError(err error) BanType {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	switch {
	case contains(errStr, "463") || contains(errStr, "reachout"):
		return BanTimelock
	case contains(errStr, "429") || contains(errStr, "rate-overlimit"):
		return BanRateOverlimit
	case contains(errStr, "405") || contains(errStr, "client_outdated"):
		return BanSoft
	case contains(errStr, "401") || contains(errStr, "logged_out") || contains(errStr, "device_removed"):
		return BanHard
	case contains(errStr, "402") || contains(errStr, "temp_ban") || contains(errStr, "temporary_ban"):
		return BanHard
	default:
		return BanSoft
	}
}

// RecordBanEvent records a ban event and transitions the recovery phase accordingly.
func (bro *BanRecoveryOrchestrator) RecordBanEvent(event BanEvent) {
	bro.mu.Lock()
	defer bro.mu.Unlock()

	bro.banEvents = append(bro.banEvents, event)

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	recentBans := 0
	for _, e := range bro.banEvents {
		if e.Timestamp.After(cutoff) {
			recentBans++
		}
	}

	bro.pruneOldEvents()

	switch event.Type {
	case BanTimelock:
		bro.phase = PhasePaused
	case BanRateOverlimit:
		bro.phase = PhasePaused
	case BanSoft:
		if recentBans >= bro.cfg.MaxBansBeforeHard {
			bro.phase = PhaseDead
		} else {
			bro.phase = PhaseRecovering
		}
	case BanHard:
		if recentBans >= bro.cfg.MaxBansBeforeHard {
			bro.phase = PhaseDead
		} else {
			bro.phase = PhasePaused
		}
	}
	bro.rampDay = 1
}

// GetRateMultiplier returns the current rate multiplier based on recovery phase.
func (bro *BanRecoveryOrchestrator) GetRateMultiplier() float64 {
	bro.mu.Lock()
	defer bro.mu.Unlock()
	return bro.getRateMultiplier()
}

// Tick advances the recovery phase (paused -> recovering -> ramping -> graduated).
func (bro *BanRecoveryOrchestrator) Tick() {
	bro.mu.Lock()
	defer bro.mu.Unlock()

	switch bro.phase {
	case PhasePaused:
		bro.phase = PhaseRecovering
	case PhaseRecovering:
		bro.phase = PhaseRamping
		bro.rampDay = 1
	case PhaseRamping:
		bro.rampDay++
		if bro.rampDay > 7 {
			bro.phase = PhaseGraduated
		}
	}
}

// GetStatus returns the current recovery status for monitoring.
func (bro *BanRecoveryOrchestrator) GetStatus() map[string]any {
	bro.mu.Lock()
	defer bro.mu.Unlock()
	bro.pruneOldEvents()

	mult := bro.getRateMultiplier()

	return map[string]any{
		"phase":           bro.phase,
		"ramp_day":        bro.rampDay,
		"rate_mult":       math.Round(mult*100) / 100,
		"recent_bans_30d": len(bro.banEvents),
	}
}

func (bro *BanRecoveryOrchestrator) getRateMultiplier() float64 {
	switch bro.phase {
	case PhasePaused:
		return 0
	case PhaseRecovering:
		return bro.cfg.RecoveryRampPct
	case PhaseRamping:
		progress := float64(bro.rampDay) / 7.0
		if progress > 1.0 {
			progress = 1.0
		}
		return bro.cfg.RecoveryRampPct + (1-bro.cfg.RecoveryRampPct)*progress
	case PhaseGraduated:
		return 1.0
	case PhaseDead:
		return 0
	default:
		return 0
	}
}

func (bro *BanRecoveryOrchestrator) pruneOldEvents() {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	i := 0
	for _, e := range bro.banEvents {
		if e.Timestamp.After(cutoff) {
			bro.banEvents[i] = e
			i++
		}
	}
	bro.banEvents = bro.banEvents[:i]
}

// Reset clears all ban events and returns to the graduated phase.
func (bro *BanRecoveryOrchestrator) Reset() {
	bro.mu.Lock()
	defer bro.mu.Unlock()
	bro.phase = PhaseGraduated
	bro.banEvents = nil
	bro.rampDay = 0
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
