package antiban

import (
	"math"
	"sync"
	"time"
)

// HealthMonitor tracks account health via a scoring system and auto-pauses on elevated risk.
type HealthMonitor struct {
	mu sync.RWMutex

	cfg *Config

	score float64

	disconnects       int
	forbiddenErrors   int
	loggedOutErrors   int
	failedMessages    int
	timelockErrors    int

	lastDecay      time.Time
	isPaused       bool
	pauseTriggered bool

	onRiskChange func(RiskLevel)
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(cfg *Config) *HealthMonitor {
	return &HealthMonitor{
		cfg:       cfg,
		lastDecay: time.Now(),
	}
}

// OnRiskChange registers a callback for when the risk level changes.
func (hm *HealthMonitor) OnRiskChange(fn func(RiskLevel)) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.onRiskChange = fn
}

func (hm *HealthMonitor) decay() {
	now := time.Now()
	elapsed := now.Sub(hm.lastDecay)
	if elapsed < time.Minute {
		return
	}
	hm.lastDecay = now

	decayPerMin := 2.0
	if hm.score < 40 {
		decayPerMin = 5.0
	}

	decay := decayPerMin * elapsed.Minutes()
	hm.score = math.Max(0, hm.score-decay)
}

func (hm *HealthMonitor) scoreEvent(weight float64) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.decay()
	hm.score = math.Min(100, hm.score+weight)
	hm.checkPause()
}

func (hm *HealthMonitor) checkPause() {
	if !hm.pauseTriggered && hm.score >= float64(hm.cfg.AutoPauseRiskLevel) {
		hm.pauseTriggered = true
		hm.isPaused = true
		if hm.onRiskChange != nil {
			go hm.onRiskChange(hm.getRiskLevel())
		}
	}
	if hm.pauseTriggered && hm.score < float64(hm.cfg.AutoPauseRiskLevel)*0.5 {
		hm.pauseTriggered = false
		hm.isPaused = false
		if hm.onRiskChange != nil {
			go hm.onRiskChange(hm.getRiskLevel())
		}
	}
}

// RecordDisconnect records a disconnect event (+10 score).
func (hm *HealthMonitor) RecordDisconnect() {
	hm.mu.Lock()
	hm.disconnects++
	hm.mu.Unlock()
	hm.scoreEvent(10)
}

// RecordReconnect records a successful reconnect (-5 score).
func (hm *HealthMonitor) RecordReconnect() {
	hm.scoreEvent(-5)
}

// RecordMessageFailed records a message failure event (+5 score).
func (hm *HealthMonitor) RecordMessageFailed() {
	hm.mu.Lock()
	hm.failedMessages++
	hm.mu.Unlock()
	hm.scoreEvent(5)
}

// RecordForbidden records a forbidden/405 error (+20 score).
func (hm *HealthMonitor) RecordForbidden() {
	hm.mu.Lock()
	hm.forbiddenErrors++
	hm.mu.Unlock()
	hm.scoreEvent(20)
}

// RecordLoggedOut records a logged out/401 error (+40 score).
func (hm *HealthMonitor) RecordLoggedOut() {
	hm.mu.Lock()
	hm.loggedOutErrors++
	hm.mu.Unlock()
	hm.scoreEvent(40)
}

// RecordReachoutTimelock records a timelock/463 error (+30 score).
func (hm *HealthMonitor) RecordReachoutTimelock() {
	hm.mu.Lock()
	hm.timelockErrors++
	hm.mu.Unlock()
	hm.scoreEvent(30)
}

func (hm *HealthMonitor) getRiskLevel() RiskLevel {
	switch {
	case hm.score >= 80:
		return RiskCritical
	case hm.score >= 40:
		return RiskHigh
	case hm.score >= 15:
		return RiskMedium
	default:
		return RiskLow
	}
}

// GetRiskLevel returns the current risk level based on the health score.
func (hm *HealthMonitor) GetRiskLevel() RiskLevel {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.getRiskLevel()
}

// IsPaused returns whether the health monitor has triggered a pause.
func (hm *HealthMonitor) IsPaused() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.isPaused
}

// SetPaused manually overrides the paused state.
func (hm *HealthMonitor) SetPaused(paused bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.isPaused = paused
}

func (hm *HealthMonitor) GetStatus() map[string]any {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return map[string]any{
		"score":            hm.score,
		"risk_level":       hm.getRiskLevel(),
		"is_paused":        hm.isPaused,
		"disconnects":      hm.disconnects,
		"forbidden_errors": hm.forbiddenErrors,
		"logged_out":       hm.loggedOutErrors,
		"failed_messages":  hm.failedMessages,
		"timelock_errors":  hm.timelockErrors,
	}
}

func (hm *HealthMonitor) Reset() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.score = 0
	hm.disconnects = 0
	hm.forbiddenErrors = 0
	hm.loggedOutErrors = 0
	hm.failedMessages = 0
	hm.timelockErrors = 0
	hm.isPaused = false
	hm.pauseTriggered = false
	hm.lastDecay = time.Now()
}
