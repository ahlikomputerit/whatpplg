package antiban

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// TypingPause represents a simulated pause during typing, used to mimic human behavior.
type TypingPause struct {
	Duration time.Duration
	AtWord   int
}

var synonyms = map[string][]string{
	"hello":   {"hi", "hey", "hey there", "yo", "greetings"},
	"goodbye": {"bye", "see you", "laters", "take care"},
	"thanks":  {"thank you", "thx", "appreciate it", "cheers"},
	"yes":     {"yeah", "yep", "sure", "okay", "alright"},
	"no":      {"nope", "nah", "not really"},
	"please":  {"pls", "pretty please", "if you don't mind"},
	"sorry":   {"my bad", "apologies", "pardon"},
}

// ContentVariator applies variations to message content (typos, zero-width chars,
// emoji padding, punctuation changes) to avoid detection of identical messages.
type ContentVariator struct {
	cfg *Config
}

// NewContentVariator creates a new content variator.
func NewContentVariator(cfg *Config) *ContentVariator {
	return &ContentVariator{cfg: cfg}
}

var zeroWidthChars = []string{
	"\u200B", "\u200C", "\u200D", "\uFEFF", "\u2060",
}

// Vary applies configured variations to the given text.
func (cv *ContentVariator) Vary(text string) string {
	if cv.cfg.EnableZeroWidth {
		text = cv.injectZeroWidth(text)
	}
	if cv.cfg.EnablePunctuationVary {
		text = cv.varyPunctuation(text)
	}
	if cv.cfg.EnableTypoInjection && rand.Float64() < cv.cfg.TypoProbability {
		text = cv.injectTypo(text)
	}
	if cv.cfg.EnableEmojiPadding {
		text = cv.addEmojiPadding(text)
	}
	return text
}

// VaryBulk applies variations to multiple texts, ensuring no two results are identical.
func (cv *ContentVariator) VaryBulk(texts []string) []string {
	result := make([]string, len(texts))
	used := make(map[string]bool)
	for i, t := range texts {
		v := cv.Vary(t)
		for used[v] {
			v = cv.Vary(t)
		}
		used[v] = true
		result[i] = v
	}
	return result
}

func (cv *ContentVariator) injectZeroWidth(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	zw := zeroWidthChars[rand.IntN(len(zeroWidthChars))]
	pos := rand.IntN(len(words))
	words[pos] = words[pos] + zw
	return strings.Join(words, " ")
}

func (cv *ContentVariator) varyPunctuation(text string) string {
	if !strings.HasSuffix(text, ".") || len(text) < 3 {
		return text
	}
	r := rand.Float64()
	switch {
	case r < 0.05:
		return text + "."
	case r < 0.15:
		if strings.HasSuffix(text, "!") {
			return text
		}
		return text + "!"
	case r < 0.25:
		return strings.TrimRight(text, " .") + ".."
	case r < 0.3:
		return strings.TrimRight(text, " .") + "..."
	}
	return text
}

func (cv *ContentVariator) injectTypo(text string) string {
	typ := rand.IntN(3)
	switch typ {
	case 0:
		return cv.swapAdjacent(text)
	case 1:
		return cv.skipChar(text)
	case 2:
		return cv.doubleChar(text)
	}
	return text
}

func (cv *ContentVariator) swapAdjacent(text string) string {
	if len(text) < 3 {
		return text
	}
	pos := rand.IntN(len(text) - 1)
	b := []byte(text)
	b[pos], b[pos+1] = b[pos+1], b[pos]
	return string(b)
}

func (cv *ContentVariator) skipChar(text string) string {
	if len(text) < 4 {
		return text
	}
	pos := 1 + rand.IntN(len(text)-2)
	return text[:pos] + text[pos+1:]
}

func (cv *ContentVariator) doubleChar(text string) string {
	if len(text) < 2 {
		return text
	}
	pos := rand.IntN(len(text))
	return text[:pos] + string(text[pos]) + text[pos:]
}

func (cv *ContentVariator) addEmojiPadding(text string) string {
	emojis := []string{"✨", "👉", "✅", "💬", "👋", "📱", "⚡", "💪"}
	if rand.Float64() < 0.3 {
		emoji := emojis[rand.IntN(len(emojis))]
		if rand.Float64() < 0.5 {
			return fmt.Sprintf("%s %s", emoji, text)
		}
		return fmt.Sprintf("%s %s", text, emoji)
	}
	return text
}

// GetTypingPauses returns simulated typing pauses for a given text length.
func (cv *ContentVariator) GetTypingPauses(text string) []TypingPause {
	words := len(strings.Fields(text))
	if words < 3 {
		return nil
	}

	pauses := make([]TypingPause, 0)
	if rand.Float64() < 0.3 {
		pauses = append(pauses, TypingPause{
			Duration: time.Duration(1+rand.IntN(3)) * time.Second,
			AtWord:   words / 2,
		})
	}
	if words > 10 && rand.Float64() < 0.2 {
		pauses = append(pauses, TypingPause{
			Duration: time.Duration(2+rand.IntN(5)) * time.Second,
			AtWord:   3 + rand.IntN(words/3),
		})
	}
	return pauses
}
