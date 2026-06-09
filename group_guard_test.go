package antiban

import (
	"testing"
)

func TestGroupOperationGuard_New(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if gog == nil {
		t.Fatal("expected non-nil GroupOperationGuard")
	}
}

func TestGroupOperationGuard_Check_Add(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if !gog.Check(OpGroupAdd) {
		t.Fatal("expected allow for first group add")
	}
}

func TestGroupOperationGuard_Check_Remove(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if !gog.Check(OpGroupRemove) {
		t.Fatal("expected allow for first group remove")
	}
}

func TestGroupOperationGuard_Check_Create(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if !gog.Check(OpGroupCreate) {
		t.Fatal("expected allow for first group create")
	}
}

func TestGroupOperationGuard_Check_Invite(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if !gog.Check(OpGroupInvite) {
		t.Fatal("expected allow for first group invite")
	}
}

func TestGroupOperationGuard_Check_InvalidOp(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	if gog.Check(GroupOperation(99)) {
		t.Fatal("expected block for invalid operation")
	}
}

func TestGroupOperationGuard_Check_Limit(t *testing.T) {
	cfg := newTestCfg()
	cfg.MaxGroupAddsPer10m = 2
	gog := NewGroupOperationGuard(cfg)

	if !gog.Check(OpGroupAdd) {
		t.Fatal("expected allow for 1st add")
	}
	if !gog.Check(OpGroupAdd) {
		t.Fatal("expected allow for 2nd add")
	}
	if gog.Check(OpGroupAdd) {
		t.Fatal("expected block for 3rd add")
	}
}

func TestGroupOperationGuard_Reset(t *testing.T) {
	cfg := newTestCfg()
	cfg.MaxGroupAddsPer10m = 1
	gog := NewGroupOperationGuard(cfg)
	gog.Check(OpGroupAdd)
	gog.Reset()
	if !gog.Check(OpGroupAdd) {
		t.Fatal("expected allow after reset")
	}
}

func TestGroupOperationGuard_GetStats(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	gog.Check(OpGroupAdd)
	stats := gog.GetStats()
	if stats["add"].(int) != 1 {
		t.Fatalf("expected 1 add in stats, got %d", stats["add"])
	}
}

func TestGroupOperationGuard_GetStats_Zero(t *testing.T) {
	gog := NewGroupOperationGuard(newTestCfg())
	stats := gog.GetStats()
	if len(stats) != 0 {
		t.Fatalf("expected empty stats, got %v", stats)
	}
}
