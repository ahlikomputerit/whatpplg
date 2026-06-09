package antiban

import (
	"testing"
	"time"
)

func TestTimelockGuard_InitialState(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	st := tg.GetState()
	if st["blocked"].(bool) {
		t.Fatal("expected not blocked initially")
	}
	if !tg.CanSend("test@s.whatsapp.net") {
		t.Fatal("expected CanSend initially")
	}
}

func TestTimelockGuard_Record463Error(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.Record463Error()
	st := tg.GetState()
	if !st["blocked"].(bool) {
		t.Fatal("expected blocked after 463 error")
	}
}

func TestTimelockGuard_CanSend_BlocksNewContact(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.Record463Error()
	if tg.CanSend("stranger@s.whatsapp.net") {
		t.Fatal("expected block for unknown contact during timelock")
	}
}

func TestTimelockGuard_CanSend_AllowsKnownChat(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.RegisterKnownChat("friend@s.whatsapp.net")
	tg.Record463Error()
	if !tg.CanSend("friend@s.whatsapp.net") {
		t.Fatal("expected allow for known chat during timelock")
	}
}

func TestTimelockGuard_RegisterKnownChat(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.RegisterKnownChat("user@s.whatsapp.net")
	st := tg.GetState()
	if st["known_chats"].(int) != 1 {
		t.Fatalf("expected 1 known chat, got %d", st["known_chats"])
	}
}

func TestTimelockGuard_Lift(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.Record463Error()
	tg.Lift()
	st := tg.GetState()
	if st["blocked"].(bool) {
		t.Fatal("expected not blocked after lift")
	}
}

func TestTimelockGuard_Reset(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	tg.Record463Error()
	tg.RegisterKnownChat("user@s.whatsapp.net")
	tg.Reset()
	st := tg.GetState()
	if st["blocked"].(bool) {
		t.Fatal("expected not blocked after reset")
	}
	if st["known_chats"].(int) != 0 {
		t.Fatalf("expected 0 known chats after reset, got %d", st["known_chats"])
	}
}

func TestTimelockGuard_OnTimelockDetected(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	fired := make(chan time.Duration, 1)
	tg.OnTimelockDetected(func(d time.Duration) {
		fired <- d
	})
	tg.Record463Error()
	select {
	case d := <-fired:
		if d <= 0 {
			t.Fatal("expected positive duration")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OnTimelockDetected callback did not fire")
	}
}

func TestTimelockGuard_OnTimelockLifted(t *testing.T) {
	cfg := newTestCfg()
	cfg.TimelockBlockDuration = 10 * time.Millisecond
	tg := NewTimelockGuard(cfg)
	fired := make(chan struct{}, 1)
	tg.OnTimelockLifted(func() {
		fired <- struct{}{}
	})
	tg.Record463Error()
	select {
	case <-fired:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("OnTimelockLifted callback did not fire from autoLift")
	}
}

func TestTimelockGuard_AutoLift(t *testing.T) {
	cfg := newTestCfg()
	cfg.TimelockBlockDuration = 50 * time.Millisecond
	tg := NewTimelockGuard(cfg)
	tg.Record463Error()
	time.Sleep(100 * time.Millisecond)
	st := tg.GetState()
	if st["blocked"].(bool) {
		t.Fatal("expected auto-lift after block duration")
	}
}

func TestTimelockGuard_GetState(t *testing.T) {
	tg := NewTimelockGuard(newTestCfg())
	st := tg.GetState()
	if _, ok := st["blocked"]; !ok {
		t.Fatal("expected blocked field in state")
	}
	if _, ok := st["blocked_until"]; !ok {
		t.Fatal("expected blocked_until field in state")
	}
	if _, ok := st["remaining_sec"]; !ok {
		t.Fatal("expected remaining_sec field in state")
	}
	if _, ok := st["known_chats"]; !ok {
		t.Fatal("expected known_chats field in state")
	}
}
