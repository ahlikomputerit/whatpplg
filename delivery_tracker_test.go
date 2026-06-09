package antiban

import (
	"testing"
)

func TestDeliveryTracker_InitialState(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	stats := dt.GetStats()
	if stats["sent"].(int) != 0 {
		t.Fatalf("expected 0 sent, got %d", stats["sent"])
	}
	if stats["delivered"].(int) != 0 {
		t.Fatalf("expected 0 delivered, got %d", stats["delivered"])
	}
}

func TestDeliveryTracker_OnMessageSent(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	dt.OnMessageSent()
	stats := dt.GetStats()
	if stats["sent"].(int) != 1 {
		t.Fatalf("expected 1 sent, got %d", stats["sent"])
	}
}

func TestDeliveryTracker_OnDeliveryReceipt(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	dt.OnDeliveryReceipt()
	stats := dt.GetStats()
	if stats["delivered"].(int) != 1 {
		t.Fatalf("expected 1 delivered, got %d", stats["delivered"])
	}
}

func TestDeliveryTracker_DeliveryRate(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	dt.OnMessageSent()
	dt.OnMessageSent()
	dt.OnDeliveryReceipt()
	stats := dt.GetStats()
	rate := stats["delivery_rate"].(float64)
	if rate != 0.5 {
		t.Fatalf("expected delivery rate 0.5, got %f", rate)
	}
}

func TestDeliveryTracker_Reset(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	dt.OnMessageSent()
	dt.Reset()
	stats := dt.GetStats()
	if stats["sent"].(int) != 0 {
		t.Fatalf("expected 0 sent after reset, got %d", stats["sent"])
	}
}

func TestDeliveryTracker_OnLowDeliveryRate(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	fired := false
	dt.OnLowDeliveryRate(func(rate float64) {
		fired = true
	})
	_ = fired
	dt.OnMessageSent()
	dt.OnMessageSent()
	_ = dt.GetStats()
}

func TestDeliveryTracker_GetStats(t *testing.T) {
	dt := NewDeliveryTracker(newTestCfg())
	stats := dt.GetStats()
	if _, ok := stats["sent"]; !ok {
		t.Fatal("expected sent field in stats")
	}
	if _, ok := stats["delivered"]; !ok {
		t.Fatal("expected delivered field in stats")
	}
	if _, ok := stats["delivery_rate"]; !ok {
		t.Fatal("expected delivery_rate field in stats")
	}
	if _, ok := stats["min_samples"]; !ok {
		t.Fatal("expected min_samples field in stats")
	}
}
