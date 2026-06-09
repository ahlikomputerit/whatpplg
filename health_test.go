package antiban

import (
	"testing"
	"time"
)

func TestHealthMonitor_InitialState(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	st := h.GetStatus()
	if st["score"].(float64) != 0 {
		t.Fatalf("expected score 0, got %v", st["score"])
	}
	if st["risk_level"].(RiskLevel) != RiskLow {
		t.Fatalf("expected low risk, got %v", st["risk_level"])
	}
	if h.IsPaused() {
		t.Fatal("expected not paused initially")
	}
}

func TestHealthMonitor_RecordDisconnect(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordDisconnect()
	st := h.GetStatus()
	if st["disconnects"].(int) != 1 {
		t.Fatalf("expected 1 disconnect, got %d", st["disconnects"])
	}
}

func TestHealthMonitor_RecordMessageFailed(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordMessageFailed()
	st := h.GetStatus()
	if st["failed_messages"].(int) != 1 {
		t.Fatalf("expected 1 failed message, got %d", st["failed_messages"])
	}
}

func TestHealthMonitor_RiskEscalation(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())

	h.RecordForbidden()
	h.RecordForbidden()
	st := h.GetStatus()
	if st["risk_level"].(RiskLevel) != RiskHigh {
		t.Fatalf("expected high risk after 2 forbidden, got %v", st["risk_level"])
	}
}

func TestHealthMonitor_LoggedOut(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordLoggedOut()
	h.RecordLoggedOut()
	st := h.GetStatus()
	if st["logged_out"].(int) != 2 {
		t.Fatalf("expected 2 logged out, got %d", st["logged_out"])
	}
	if st["risk_level"].(RiskLevel) != RiskCritical {
		t.Fatalf("expected critical risk after 2 logged out (80 pts), got %v", st["risk_level"])
	}
}

func TestHealthMonitor_Timelock(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordReachoutTimelock()
	st := h.GetStatus()
	if st["timelock_errors"].(int) != 1 {
		t.Fatalf("expected 1 timelock error, got %d", st["timelock_errors"])
	}
}

func TestHealthMonitor_Reconnect(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordForbidden()
	before := h.GetStatus()["score"].(float64)
	h.RecordReconnect()
	after := h.GetStatus()["score"].(float64)
	if after >= before {
		t.Fatalf("expected score to decrease on reconnect, was %v now %v", before, after)
	}
}

func TestHealthMonitor_RiskChangeCallback(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	fired := make(chan RiskLevel, 1)
	h.OnRiskChange(func(level RiskLevel) {
		fired <- level
	})
	h.RecordLoggedOut()
	select {
	case <-fired:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected OnRiskChange callback to fire")
	}
}

func TestHealthMonitor_AutoPause(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordLoggedOut()
	if !h.IsPaused() {
		t.Fatal("expected auto-pause on critical risk")
	}
}

func TestHealthMonitor_SetPaused(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.SetPaused(true)
	if !h.IsPaused() {
		t.Fatal("expected paused after SetPaused(true)")
	}
	h.SetPaused(false)
	if h.IsPaused() {
		t.Fatal("expected not paused after SetPaused(false)")
	}
}

func TestHealthMonitor_Reset(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	h.RecordLoggedOut()
	h.Reset()
	st := h.GetStatus()
	if st["score"].(float64) != 0 {
		t.Fatalf("expected score 0 after reset, got %v", st["score"])
	}
	if st["disconnects"].(int) != 0 {
		t.Fatalf("expected 0 disconnects after reset, got %d", st["disconnects"])
	}
	if h.IsPaused() {
		t.Fatal("expected not paused after reset")
	}
}

func TestHealthMonitor_GetRiskLevel(t *testing.T) {
	h := NewHealthMonitor(newTestCfg())
	if h.GetRiskLevel() != RiskLow {
		t.Fatal("expected low risk initially")
	}
	h.RecordLoggedOut()
	if h.GetRiskLevel() != RiskHigh {
		t.Fatal("expected high risk after logged out (40 pts)")
	}
	h.RecordLoggedOut()
	if h.GetRiskLevel() != RiskCritical {
		t.Fatal("expected critical risk after 2 logged out (80 pts)")
	}
}
