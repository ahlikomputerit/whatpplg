package antiban

import (
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestWrapClient(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	if abc == nil {
		t.Fatal("expected non-nil AntiBanClient")
	}
	if abc.WAClient != wac {
		t.Fatal("expected wrapped client to match")
	}
	if abc.AntiBan == nil {
		t.Fatal("expected non-nil AntiBan instance")
	}
}

func TestWrapClient_WithConfig(t *testing.T) {
	wac := newTestClient(t)
	cfg := DefaultConfig(PresetModerate)
	cfg.MaxPerMinute = 10
	abc := WrapClient(wac, PresetModerate, cfg)
	if abc.Config.MaxPerMinute != 10 {
		t.Fatalf("expected MaxPerMinute 10, got %d", abc.Config.MaxPerMinute)
	}
}

func TestAntiBanClient_GetStats(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	stats := abc.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestAntiBanClient_GetWAClient(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	if abc.GetWAClient() != wac {
		t.Fatal("expected GetWAClient to return original client")
	}
}

func TestAntiBanClient_StartStop(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	ctx := t.Context()
	abc.Start(ctx)
	defer abc.Stop()
}

func TestAntiBanClient_StartTwice(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	ctx := t.Context()
	abc.Start(ctx)
	abc.Start(ctx)
	abc.Stop()
}

func TestAntiBanClient_StopWithoutStart(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	abc.Stop()
}

func TestWrapClient_AllPresets(t *testing.T) {
	wac := newTestClient(t)
	presets := []Preset{PresetConservative, PresetModerate, PresetAggressive, PresetHighVolume}
	for _, p := range presets {
		abc := WrapClient(wac, p)
		if abc.AntiBan == nil {
			t.Fatalf("expected non-nil AntiBan for preset %s", p)
		}
	}
}

func TestAntiBanClient_HandleEvent(t *testing.T) {
	wac := newTestClient(t)
	abc := WrapClient(wac, PresetConservative)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in event handling: %v", r)
			}
		}()
		abc.handleEvent(struct{}{})
	}()
}

func newTestClient(t *testing.T) *whatsmeow.Client {
	t.Helper()
	jid := types.NewJID("test", types.DefaultUserServer)
	device := &store.Device{
		ID: &jid,
	}
	log := waLog.Noop
	client := whatsmeow.NewClient(device, log)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	return client
}
