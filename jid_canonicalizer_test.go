package antiban

import (
	"testing"
)

func TestJidCanonicalizer_New(t *testing.T) {
	resolver := NewLidResolver(100)
	jc := NewJidCanonicalizer(resolver)
	if jc == nil {
		t.Fatal("expected non-nil JidCanonicalizer")
	}
}

func TestJidCanonicalizer_NilResolver(t *testing.T) {
	jc := NewJidCanonicalizer(nil)
	result := jc.CanonicalizeTarget("test@s.whatsapp.net")
	if result != "test@s.whatsapp.net" {
		t.Fatalf("expected unchanged JID, got %s", result)
	}
}

func TestJidCanonicalizer_NilReceiver(t *testing.T) {
	var jc *JidCanonicalizer
	result := jc.CanonicalizeTarget("test@s.whatsapp.net")
	if result != "test@s.whatsapp.net" {
		t.Fatalf("expected unchanged JID on nil receiver, got %s", result)
	}
}

func TestJidCanonicalizer_CanonicalizeTarget(t *testing.T) {
	resolver := NewLidResolver(100)
	resolver.Learn("lid:123", "pn:456")
	jc := NewJidCanonicalizer(resolver)

	result := jc.CanonicalizeTarget("lid:123")
	if result != "pn:456" {
		t.Fatalf("expected pn:456, got %s", result)
	}
}

func TestJidCanonicalizer_CanonicalKey(t *testing.T) {
	resolver := NewLidResolver(100)
	jc := NewJidCanonicalizer(resolver)

	key := jc.CanonicalKey("test@s.whatsapp.net")
	if key != "test" {
		t.Fatalf("expected 'test', got %s", key)
	}
}

func TestJidCanonicalizer_OnIncomingEvent(t *testing.T) {
	resolver := NewLidResolver(100)
	jc := NewJidCanonicalizer(resolver)

	jc.OnIncomingEvent("lid:123@lid", "pn:456@s.whatsapp.net")
	if !resolver.HasMapping("lid:123") {
		t.Fatal("expected mapping to be learned from incoming event")
	}
}

func TestJidCanonicalizer_OnIncomingEvent_Reverse(t *testing.T) {
	resolver := NewLidResolver(100)
	jc := NewJidCanonicalizer(resolver)

	jc.OnIncomingEvent("pn:456@s.whatsapp.net", "lid:123@lid")
	if !resolver.HasMapping("lid:123") {
		t.Fatal("expected mapping to be learned from reverse event")
	}
}

func TestJidCanonicalizer_OnIncomingEvent_Nil(t *testing.T) {
	var jc *JidCanonicalizer
	jc.OnIncomingEvent("test", "participant")
}

func TestExtractJIDUser(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test@s.whatsapp.net", "test"},
		{"lid:123@lid", "lid:123"},
		{"noat", "noat"},
		{"", ""},
	}

	for _, tc := range tests {
		got := extractJIDUser(tc.input)
		if got != tc.expected {
			t.Errorf("extractJIDUser(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
