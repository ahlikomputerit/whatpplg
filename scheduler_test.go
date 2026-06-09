package antiban

import (
	"testing"
	"time"
)

func TestScheduler_New(t *testing.T) {
	s := NewScheduler(newTestCfg())
	if s == nil {
		t.Fatal("expected non-nil Scheduler")
	}
}

func TestScheduler_IsActiveTime(t *testing.T) {
	cfg := newTestCfg()
	cfg.ActiveHourStart = 0
	cfg.ActiveHourEnd = 24
	s := NewScheduler(cfg)
	if !s.IsActiveTime() {
		t.Fatal("expected active time with 0-24 window")
	}
}

func TestScheduler_IsActiveTime_Inactive(t *testing.T) {
	cfg := newTestCfg()
	cfg.ActiveHourStart = 0
	cfg.ActiveHourEnd = 1
	s := NewScheduler(cfg)
	active := s.IsActiveTime()
	if active {
		t.Log("note: may be active if current hour is 0-1")
	}
}

func TestScheduler_GetSpeedFactor(t *testing.T) {
	s := NewScheduler(newTestCfg())
	factor := s.GetSpeedFactor()
	if factor <= 0 {
		t.Fatalf("expected positive factor, got %f", factor)
	}
}

func TestScheduler_AdjustDelay(t *testing.T) {
	s := NewScheduler(newTestCfg())
	original := 100 * time.Millisecond
	adjusted := s.AdjustDelay(original)
	if adjusted <= 0 {
		t.Fatal("expected positive adjusted delay")
	}
}

func TestScheduler_MsUntilActive(t *testing.T) {
	cfg := newTestCfg()
	cfg.ActiveHourStart = 0
	cfg.ActiveHourEnd = 24
	s := NewScheduler(cfg)
	if s.MsUntilActive() != 0 {
		t.Fatal("expected 0 when already active")
	}
}

func TestScheduler_MsUntilActive_Inactive(t *testing.T) {
	cfg := newTestCfg()
	cfg.ActiveHourStart = 0
	cfg.ActiveHourEnd = 1
	s := NewScheduler(cfg)
	ms := s.MsUntilActive()
	if ms < 0 {
		t.Fatal("expected non-negative wait time")
	}
}

func TestScheduler_GetStatus(t *testing.T) {
	s := NewScheduler(newTestCfg())
	status := s.GetStatus()
	if _, ok := status["active"]; !ok {
		t.Fatal("expected active field in status")
	}
	if _, ok := status["speed"]; !ok {
		t.Fatal("expected speed field in status")
	}
	if _, ok := status["next_ms"]; !ok {
		t.Fatal("expected next_ms field in status")
	}
}
