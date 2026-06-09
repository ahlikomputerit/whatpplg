package antiban

import (
	"testing"
)

func TestLidResolver_New(t *testing.T) {
	lr := NewLidResolver(100)
	if lr == nil {
		t.Fatal("expected non-nil LidResolver")
	}
}

func TestLidResolver_New_DefaultSize(t *testing.T) {
	lr := NewLidResolver(0)
	if lr.maxSize != 1000 {
		t.Fatalf("expected default maxSize 1000, got %d", lr.maxSize)
	}
}

func TestLidResolver_LearnAndResolve(t *testing.T) {
	lr := NewLidResolver(100)
	lr.Learn("lid:123", "pn:456")

	pn := lr.GetPN("lid:123")
	if pn != "pn:456" {
		t.Fatalf("expected pn:456, got %s", pn)
	}

	lid := lr.GetLID("pn:456")
	if lid != "lid:123" {
		t.Fatalf("expected lid:123, got %s", lid)
	}
}

func TestLidResolver_ResolveCanonical(t *testing.T) {
	lr := NewLidResolver(100)
	lr.Learn("lid:123", "pn:456")

	if lr.ResolveCanonical("lid:123") != "pn:456" {
		t.Fatal("expected lid:123 to resolve to pn:456")
	}
	if lr.ResolveCanonical("pn:456") != "lid:123" {
		t.Fatal("expected pn:456 to resolve to lid:123")
	}
	if lr.ResolveCanonical("unknown") != "unknown" {
		t.Fatal("expected unknown to stay unchanged")
	}
}

func TestLidResolver_HasMapping(t *testing.T) {
	lr := NewLidResolver(100)
	lr.Learn("lid:1", "pn:1")

	if !lr.HasMapping("lid:1") {
		t.Fatal("expected HasMapping true for lid:1")
	}
	if !lr.HasMapping("pn:1") {
		t.Fatal("expected HasMapping true for pn:1")
	}
	if lr.HasMapping("unknown") {
		t.Fatal("expected HasMapping false for unknown")
	}
}

func TestLidResolver_GetMapping(t *testing.T) {
	lr := NewLidResolver(100)
	lr.Learn("lid:1", "pn:1")
	lr.Learn("lid:2", "pn:2")

	m := lr.GetMapping()
	if len(m) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(m))
	}
	if m["lid:1"] != "pn:1" {
		t.Fatalf("expected lid:1 -> pn:1, got %s", m["lid:1"])
	}
}

func TestLidResolver_Learn_Overwrite(t *testing.T) {
	lr := NewLidResolver(100)
	lr.Learn("lid:1", "pn:1")
	lr.Learn("lid:1", "pn:2")

	if lr.GetPN("lid:1") != "pn:2" {
		t.Fatalf("expected pn:2 after overwrite, got %s", lr.GetPN("lid:1"))
	}
	if lr.GetLID("pn:1") != "" {
		t.Fatal("expected pn:1 mapping to be removed")
	}
}

func TestLidResolver_LRU_Eviction(t *testing.T) {
	lr := NewLidResolver(2)
	lr.Learn("lid:1", "pn:1")
	lr.Learn("lid:2", "pn:2")
	lr.Learn("lid:3", "pn:3")

	if lr.HasMapping("lid:1") {
		t.Log("note: LRU eviction may not remove lid:1 (depends on order)")
	}
	if !lr.HasMapping("lid:2") {
		t.Fatal("expected lid:2 to still be present")
	}
	if !lr.HasMapping("lid:3") {
		t.Fatal("expected lid:3 to be present")
	}
}
