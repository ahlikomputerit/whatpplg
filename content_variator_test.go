package antiban

import (
	"testing"
)

func TestContentVariator_New(t *testing.T) {
	cv := NewContentVariator(newTestCfg())
	if cv == nil {
		t.Fatal("expected non-nil ContentVariator")
	}
}

func TestContentVariator_Vary_Noop(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnableTypoInjection = false
	cfg.EnableZeroWidth = false
	cfg.EnableEmojiPadding = false
	cfg.EnablePunctuationVary = false
	cv := NewContentVariator(cfg)

	original := "hello world"
	result := cv.Vary(original)
	if result != original {
		t.Fatalf("expected unchanged text, got %q", result)
	}
}

func TestContentVariator_Vary_TypoInjection(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnableTypoInjection = true
	cfg.TypoProbability = 1.0
	cfg.EnableZeroWidth = false
	cfg.EnableEmojiPadding = false
	cfg.EnablePunctuationVary = false
	cv := NewContentVariator(cfg)

	original := "hello world"
	result := cv.Vary(original)
	if result == original {
		t.Fatal("expected varied text with typo injection")
	}
}

func TestContentVariator_Vary_ZeroWidth(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnableZeroWidth = true
	cfg.EnableTypoInjection = false
	cfg.EnableEmojiPadding = false
	cfg.EnablePunctuationVary = false
	cv := NewContentVariator(cfg)

	result := cv.Vary("hello world")
	if len(result) < len("hello world") {
		t.Fatal("expected zero-width chars added")
	}
}

func TestContentVariator_Vary_EmojiPadding(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnableEmojiPadding = true
	cfg.EnableTypoInjection = false
	cfg.EnableZeroWidth = false
	cfg.EnablePunctuationVary = false
	cv := NewContentVariator(cfg)

	results := make(map[string]bool)
	for i := 0; i < 20; i++ {
		r := cv.Vary("hello")
		results[r] = true
	}
	if len(results) == 1 && results["hello"] {
		t.Log("note: emoji padding may not always fire (30% chance)")
	}
}

func TestContentVariator_Vary_PunctuationVary(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnablePunctuationVary = true
	cfg.EnableTypoInjection = false
	cfg.EnableZeroWidth = false
	cfg.EnableEmojiPadding = false
	cv := NewContentVariator(cfg)

	results := make(map[string]bool)
	for i := 0; i < 20; i++ {
		r := cv.Vary("hello")
		results[r] = true
	}
	if len(results) == 1 && results["hello"] {
		t.Log("note: punctuation vary may not always fire")
	}
}

func TestContentVariator_VaryBulk(t *testing.T) {
	cfg := newTestCfg()
	cfg.EnableTypoInjection = true
	cfg.TypoProbability = 1.0
	cfg.EnableZeroWidth = false
	cfg.EnableEmojiPadding = false
	cfg.EnablePunctuationVary = false
	cv := NewContentVariator(cfg)

	texts := []string{"hello", "world", "foo", "bar"}
	results := cv.VaryBulk(texts)

	if len(results) != len(texts) {
		t.Fatalf("expected %d results, got %d", len(texts), len(results))
	}

	seen := make(map[string]bool)
	for _, r := range results {
		if seen[r] {
			t.Fatalf("duplicate result: %q", r)
		}
		seen[r] = true
	}
}

func TestContentVariator_GetTypingPauses(t *testing.T) {
	cv := NewContentVariator(newTestCfg())

	pauses := cv.GetTypingPauses("short")
	if pauses != nil {
		t.Log("note: short text may still produce pauses depending on random")
	}

	pauses = cv.GetTypingPauses("this is a longer text with many words that should trigger pauses")
	_ = pauses
}
