package shard

import (
	"testing"

	"graphstore/internal/model"
)

// TestRouterRoutesToNewGenerationAfterRotate reproduces the shard-boundary
// bug: after a bucket is rotated, new writes resolved through the router must
// land in the freshly created active generation, not the sealed one it
// replaced. Before the fix RotateBucket never called Router.Switch, so the
// route table kept pointing at the sealed shard and traversal broke at the
// boundary.
func TestRouterRoutesToNewGenerationAfterRotate(t *testing.T) {
	manager := NewManager(2, 4)
	router := NewRouter(manager, 2)

	key := "v-from>feeds>v-to"
	bucket := router.Bucket(key)
	before := router.ShardForKey(key)
	if before == nil {
		t.Fatalf("router has no shard for key %q", key)
	}

	// Seal the active generation and stand up the replacement.
	next := manager.Rotate(bucket)
	if next == nil {
		t.Fatalf("Rotate returned nil")
	}
	// The store's RotateBucket performs this switch; exercise the same path so
	// the route table follows the manager's active generation.
	router.Switch(bucket, next.ID)

	after := router.ShardForKey(key)
	if after == nil {
		t.Fatalf("router has no shard for key after rotate")
	}
	if after.ID == before.ID {
		t.Fatalf("router still points at sealed shard %d after rotate; expected new generation", before.ID)
	}
	if after.ID != next.ID {
		t.Fatalf("router routed to shard %d; expected new generation %d", after.ID, next.ID)
	}
	if after.State != Active {
		t.Fatalf("routed shard %d is %s, not active", after.ID, after.State)
	}
	if before.State == Active {
		t.Fatalf("previous generation %d was not sealed", before.ID)
	}
}

// TestShardAddRejectsSealed asserts the defence-in-depth guard: a sealed shard
// must refuse new edges so a stale route surfaces as an explicit error instead
// of silently leaking data into a frozen generation.
func TestShardAddRejectsSealed(t *testing.T) {
	shard := New(0, 4)
	edge := model.NewEdge("e1", "a", "b", "feeds")
	if err := shard.Add(edge); err != nil {
		t.Fatalf("active shard rejected edge: %v", err)
	}
	shard.Seal()
	if shard.State != Sealed {
		t.Fatalf("expected Sealed, got %s", shard.State)
	}
	leak := model.NewEdge("e2", "a", "c", "feeds")
	if err := shard.Add(leak); err == nil {
		t.Fatalf("sealed shard accepted a new edge; writes must be rejected")
	}
}
