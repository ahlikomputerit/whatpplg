package antiban

import (
	"errors"
	"testing"
	"time"
)

func TestBanRecoveryOrchestrator_New(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	if bro == nil {
		t.Fatal("expected non-nil BanRecoveryOrchestrator")
	}
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhaseGraduated {
		t.Fatalf("expected graduated phase, got %v", st["phase"])
	}
}

func TestBanRecoveryOrchestrator_GetRateMultiplier_Graduated(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	if bro.GetRateMultiplier() != 1.0 {
		t.Fatalf("expected multiplier 1.0 in graduated phase, got %f", bro.GetRateMultiplier())
	}
}

func TestBanRecoveryOrchestrator_RecordBanEvent_Timelock(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanTimelock, Timestamp: time.Now()})
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhasePaused {
		t.Fatalf("expected paused phase after timelock, got %v", st["phase"])
	}
	if bro.GetRateMultiplier() != 0 {
		t.Fatalf("expected multiplier 0 in paused phase, got %f", bro.GetRateMultiplier())
	}
}

func TestBanRecoveryOrchestrator_RecordBanEvent_RateOverlimit(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanRateOverlimit, Timestamp: time.Now()})
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhasePaused {
		t.Fatalf("expected paused phase after rate overlimit, got %v", st["phase"])
	}
}

func TestBanRecoveryOrchestrator_RecordBanEvent_Soft(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanSoft, Timestamp: time.Now()})
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhaseRecovering {
		t.Fatalf("expected recovering phase after soft ban, got %v", st["phase"])
	}
}

func TestBanRecoveryOrchestrator_RecordBanEvent_Hard(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanHard, Timestamp: time.Now()})
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhasePaused {
		t.Fatalf("expected paused phase after hard ban, got %v", st["phase"])
	}
}

func TestBanRecoveryOrchestrator_RecordBanEvent_MultipleBans(t *testing.T) {
	cfg := newTestCfg()
	cfg.MaxBansBeforeHard = 2
	bro := NewBanRecoveryOrchestrator(cfg)

	bro.RecordBanEvent(BanEvent{Type: BanHard, Timestamp: time.Now()})
	bro.RecordBanEvent(BanEvent{Type: BanHard, Timestamp: time.Now()})
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhaseDead {
		t.Fatalf("expected dead phase after multiple hard bans, got %v", st["phase"])
	}
	if bro.GetRateMultiplier() != 0 {
		t.Fatalf("expected multiplier 0 in dead phase, got %f", bro.GetRateMultiplier())
	}
}

func TestBanRecoveryOrchestrator_Tick(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())

	bro.RecordBanEvent(BanEvent{Type: BanTimelock, Timestamp: time.Now()})
	if bro.GetRateMultiplier() != 0 {
		t.Fatal("expected 0 multiplier after timelock")
	}

	bro.Tick()
	if bro.GetRateMultiplier() <= 0 {
		t.Fatal("expected positive multiplier after tick to recovering")
	}

	bro.Tick()
	if bro.GetRateMultiplier() >= 1.0 {
		t.Fatal("expected multiplier < 1.0 in ramping phase")
	}
}

func TestBanRecoveryOrchestrator_TickFullCycle(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanTimelock, Timestamp: time.Now()})

	for i := 0; i < 10; i++ {
		bro.Tick()
	}

	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhaseGraduated {
		t.Fatalf("expected graduated after full tick cycle, got %v", st["phase"])
	}
	if bro.GetRateMultiplier() != 1.0 {
		t.Fatalf("expected multiplier 1.0 after full cycle, got %f", bro.GetRateMultiplier())
	}
}

func TestBanRecoveryOrchestrator_Reset(t *testing.T) {
	bro := NewBanRecoveryOrchestrator(newTestCfg())
	bro.RecordBanEvent(BanEvent{Type: BanHard, Timestamp: time.Now()})
	bro.Reset()
	st := bro.GetStatus()
	if st["phase"].(RecoveryPhase) != PhaseGraduated {
		t.Fatalf("expected graduated after reset, got %v", st["phase"])
	}
	if bro.GetRateMultiplier() != 1.0 {
		t.Fatalf("expected multiplier 1.0 after reset, got %f", bro.GetRateMultiplier())
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err      error
		expected BanType
	}{
		{nil, BanType("")},
		{errors.New("463 reachout error"), BanTimelock},
		{errors.New("429 rate-overlimit"), BanRateOverlimit},
		{errors.New("405 client_outdated"), BanSoft},
		{errors.New("401 logged_out"), BanHard},
		{errors.New("402 temp_ban"), BanHard},
		{errors.New("device_removed"), BanHard},
		{errors.New("unknown error"), BanSoft},
	}

	for _, tc := range tests {
		got := ClassifyError(tc.err)
		if got != tc.expected {
			t.Errorf("ClassifyError(%v) = %q, want %q", tc.err, got, tc.expected)
		}
	}
}

func TestBanRecoveryOrchestrator_GetRateMultiplier_Dead(t *testing.T) {
	cfg := newTestCfg()
	cfg.MaxBansBeforeHard = 1
	bro := NewBanRecoveryOrchestrator(cfg)
	bro.RecordBanEvent(BanEvent{Type: BanHard, Timestamp: time.Now()})
	if bro.GetRateMultiplier() != 0 {
		t.Fatalf("expected 0 in dead phase, got %f", bro.GetRateMultiplier())
	}
}
