package antiban

import (
	"testing"
)

func TestIsGroup(t *testing.T) {
	tests := []struct {
		jid      string
		expected bool
	}{
		{"123@g.us", true},
		{"test@s.whatsapp.net", false},
		{"newsletter@newsletter", false},
		{"broadcast", false},
		{"", false},
	}

	for _, tc := range tests {
		got := IsGroup(tc.jid)
		if got != tc.expected {
			t.Errorf("IsGroup(%q) = %v, want %v", tc.jid, got, tc.expected)
		}
	}
}

func TestIsNewsletter(t *testing.T) {
	tests := []struct {
		jid      string
		expected bool
	}{
		{"123@newsletter", true},
		{"test@s.whatsapp.net", false},
		{"123@g.us", false},
		{"", false},
	}

	for _, tc := range tests {
		got := IsNewsletter(tc.jid)
		if got != tc.expected {
			t.Errorf("IsNewsletter(%q) = %v, want %v", tc.jid, got, tc.expected)
		}
	}
}

func TestIsBroadcast(t *testing.T) {
	tests := []struct {
		jid      string
		expected bool
	}{
		{"status@broadcast", true},
		{"test@s.whatsapp.net", false},
		{"123@g.us", false},
		{"broadcast", false},
	}

	for _, tc := range tests {
		got := IsBroadcast(tc.jid)
		if got != tc.expected {
			t.Errorf("IsBroadcast(%q) = %v, want %v", tc.jid, got, tc.expected)
		}
	}
}

func TestShouldUseGroupProfile(t *testing.T) {
	tests := []struct {
		jid      string
		expected bool
	}{
		{"123@g.us", true},
		{"test@s.whatsapp.net", false},
		{"123@newsletter", true},
		{"broadcast", false},
	}

	for _, tc := range tests {
		got := ShouldUseGroupProfile(tc.jid)
		if got != tc.expected {
			t.Errorf("ShouldUseGroupProfile(%q) = %v, want %v", tc.jid, got, tc.expected)
		}
	}
}

func TestApplyGroupMultiplier(t *testing.T) {
	got := ApplyGroupMultiplier(100, 0.5)
	if got != 50 {
		t.Fatalf("expected 50, got %d", got)
	}
}

func TestApplyGroupMultiplier_Float(t *testing.T) {
	got := ApplyGroupMultiplier(100.0, 0.5)
	if got != 50.0 {
		t.Fatalf("expected 50.0, got %f", got)
	}
}

func TestApplyGroupMultiplier_NoMultiplier(t *testing.T) {
	got := ApplyGroupMultiplier(100, 1.0)
	if got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}
}

func TestApplyGroupMultiplier_Rounding(t *testing.T) {
	got := ApplyGroupMultiplier(99, 0.5)
	if got != 49 {
		t.Fatalf("expected 49, got %d", got)
	}
}

func TestIsBroadcast_False(t *testing.T) {
	if IsBroadcast("test@s.whatsapp.net") {
		t.Fatal("expected false for non-broadcast")
	}
}

func TestIsNewsletter_False(t *testing.T) {
	if IsNewsletter("test@s.whatsapp.net") {
		t.Fatal("expected false for non-newsletter")
	}
}
