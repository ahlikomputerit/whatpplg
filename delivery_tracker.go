package antiban

import (
	"sync"
	"time"
)

// DeliveryTracker monitors sent vs delivered message ratios and alerts on low delivery rates.
type DeliveryTracker struct {
	mu sync.Mutex

	cfg        *Config
	sent       []time.Time
	delivered  []time.Time
	lastAlert  time.Time

	onLowDeliveryRate func(rate float64)
}

// NewDeliveryTracker creates a new delivery tracker.
func NewDeliveryTracker(cfg *Config) *DeliveryTracker {
	return &DeliveryTracker{cfg: cfg}
}

// OnLowDeliveryRate registers a callback for when the delivery rate drops below threshold.
func (dt *DeliveryTracker) OnLowDeliveryRate(fn func(rate float64)) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.onLowDeliveryRate = fn
}

// OnMessageSent records a sent message for delivery rate calculation.
func (dt *DeliveryTracker) OnMessageSent() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.sent = append(dt.sent, time.Now())
	dt.prune()
}

// OnDeliveryReceipt records a delivery receipt for rate calculation.
func (dt *DeliveryTracker) OnDeliveryReceipt() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.delivered = append(dt.delivered, time.Now())
	dt.prune()
}

func (dt *DeliveryTracker) prune() {
	cutoff := time.Now().Add(-1 * time.Hour)
	i := 0
	for _, t := range dt.sent {
		if t.After(cutoff) {
			dt.sent[i] = t
			i++
		}
	}
	dt.sent = dt.sent[:i]

	i = 0
	for _, t := range dt.delivered {
		if t.After(cutoff) {
			dt.delivered[i] = t
			i++
		}
	}
	dt.delivered = dt.delivered[:i]
}

func (dt *DeliveryTracker) checkAndAlert() {
	sample := len(dt.sent)
	if sample < dt.cfg.DeliveryMinSamples {
		return
	}

	rate := float64(len(dt.delivered)) / float64(sample)
	if rate < dt.cfg.DeliveryLowThreshold && time.Since(dt.lastAlert) > time.Hour {
		dt.lastAlert = time.Now()
		if dt.onLowDeliveryRate != nil {
			go dt.onLowDeliveryRate(rate)
		}
	}
}

// GetStats returns delivery tracking statistics.
func (dt *DeliveryTracker) GetStats() map[string]any {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.prune()

	sample := len(dt.sent)
	rate := 0.0
	if sample > 0 {
		rate = float64(len(dt.delivered)) / float64(sample)
	}

	return map[string]any{
		"sent":          len(dt.sent),
		"delivered":     len(dt.delivered),
		"delivery_rate": rate,
		"min_samples":   dt.cfg.DeliveryMinSamples,
	}
}

// Reset clears all delivery tracking data.
func (dt *DeliveryTracker) Reset() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.sent = dt.sent[:0]
	dt.delivered = dt.delivered[:0]
	dt.lastAlert = time.Time{}
}
