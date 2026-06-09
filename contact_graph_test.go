package antiban

import (
	"testing"
)

func TestContactGraphWarmer_New(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	if cgw == nil {
		t.Fatal("expected non-nil ContactGraphWarmer")
	}
}

func TestContactGraphWarmer_CanMessage_UnknownContact(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	if !cgw.CanMessage("stranger@s.whatsapp.net", false) {
		t.Fatal("expected allow for unknown contact")
	}
}

func TestContactGraphWarmer_CanMessage_Group(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	if !cgw.CanMessage("group@g.us", true) {
		t.Fatal("expected allow for group")
	}
}

func TestContactGraphWarmer_GroupLurkPeriod(t *testing.T) {
	cfg := newTestCfg()
	cfg.GroupLurkPeriod = 0
	cgw := NewContactGraphWarmer(cfg)
	cgw.RegisterGroupJoin("group@g.us")
	if !cgw.CanMessage("group@g.us", true) {
		t.Fatal("expected allow for group with no lurk period")
	}
}

func TestContactGraphWarmer_RegisterKnownContact(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	cgw.RegisterKnownContact("friend@s.whatsapp.net")
	state := cgw.GetContactState("friend@s.whatsapp.net")
	if state != ContactKnown {
		t.Fatalf("expected ContactKnown, got %v", state)
	}
}

func TestContactGraphWarmer_MarkHandshakeSent(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	cgw.MarkHandshakeSent("new@s.whatsapp.net")
	state := cgw.GetContactState("new@s.whatsapp.net")
	if state != ContactHandshakeSent {
		t.Fatalf("expected ContactHandshakeSent, got %v", state)
	}
}

func TestContactGraphWarmer_MarkHandshakeComplete(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	cgw.MarkHandshakeComplete("contact@s.whatsapp.net")
	state := cgw.GetContactState("contact@s.whatsapp.net")
	if state != ContactHandshakeComplete {
		t.Fatalf("expected ContactHandshakeComplete, got %v", state)
	}
}

func TestContactGraphWarmer_OnIncomingMessage(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	cgw.OnIncomingMessage("sender@s.whatsapp.net")
	state := cgw.GetContactState("sender@s.whatsapp.net")
	if state != ContactKnown {
		t.Fatalf("expected ContactKnown after incoming message, got %v", state)
	}
}

func TestContactGraphWarmer_GetContactState_Unknown(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	state := cgw.GetContactState("unknown@s.whatsapp.net")
	if state != ContactStranger {
		t.Fatalf("expected ContactStranger for unknown, got %v", state)
	}
}

func TestContactGraphWarmer_RecordMessage(t *testing.T) {
	cgw := NewContactGraphWarmer(newTestCfg())
	cgw.RegisterKnownContact("friend@s.whatsapp.net")
	cgw.RecordMessage("friend@s.whatsapp.net")
	// Should not panic
}

func TestContactGraphWarmer_CanMessage_StrangerLimit(t *testing.T) {
	cfg := newTestCfg()
	cfg.MaxStrangerPerDay = 2
	cgw := NewContactGraphWarmer(cfg)

	cgw.RegisterKnownContact("stranger@s.whatsapp.net")
	for i := 0; i < 2; i++ {
		if !cgw.CanMessage("stranger@s.whatsapp.net", false) {
			t.Fatalf("expected allow at attempt %d", i+1)
		}
		cgw.RecordMessage("stranger@s.whatsapp.net")
	}
}

func TestContactGraphWarmer_RegisterGroupJoin(t *testing.T) {
	cfg := newTestCfg()
	cfg.GroupLurkPeriod = 0
	cgw := NewContactGraphWarmer(cfg)
	cgw.RegisterGroupJoin("group@g.us")
	if !cgw.CanMessage("group@g.us", true) {
		t.Fatal("expected allow after group join with no lurk")
	}
}
