package store

import (
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/shard"
)

// TestRotateBucketSwitchesRouteAndRejectsSealedWrites reproduces the reported
// shard-boundary bug. When edge write volume crosses the capacity threshold
// the bucket is rotated; before the fix the route table was not switched, so
// subsequent writes kept landing in the already-sealed old generation. The
// walker then followed those edges only part-way because the path could not
// line up across the shard boundary.
//
// After the fix, RotateBucket must:
//  1. switch the route table to the new generation so new edges route there,
//  2. leave the old generation sealed, and
//  3. (defence-in-depth) make a sealed shard refuse a direct write.
func TestRotateBucketSwitchesRouteAndRejectsSealedWrites(t *testing.T) {
	st, err := New(Options{EdgeBuckets: 2, ShardCapacity: 2})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer st.Close()

	if err := st.PutVertex(model.NewVertex("a", "n")); err != nil {
		t.Fatalf("put vertex a: %v", err)
	}
	if err := st.PutVertex(model.NewVertex("b", "n")); err != nil {
		t.Fatalf("put vertex b: %v", err)
	}
	if err := st.PutVertex(model.NewVertex("c", "n")); err != nil {
		t.Fatalf("put vertex c: %v", err)
	}

	key := model.NewEdge("placeholder", "a", "b", "feeds").Key()
	bucket := routerBucket(st, key)

	oldID, newID, err := st.RotateBucket(bucket)
	if err != nil {
		t.Fatalf("rotate bucket %d: %v", bucket, err)
	}
	if oldID == newID {
		t.Fatalf("rotate returned identical shard ids %d", oldID)
	}

	// The route must now resolve the key to the new generation.
	routed := st.router.ShardForKey(key)
	if routed == nil {
		t.Fatalf("no shard routed for key %q", key)
	}
	if routed.ID != newID {
		t.Fatalf("route points at shard %d, want new generation %d", routed.ID, newID)
	}
	if routed.State != shard.Active {
		t.Fatalf("routed shard %d is %s, want active", routed.ID, routed.State)
	}

	// A fresh edge over the same key must land in the new generation, not the
	// sealed one.
	edge := model.NewEdge("e-a-b", "a", "b", "feeds")
	if err := st.PutEdge(edge); err != nil {
		t.Fatalf("put edge after rotate: %v", err)
	}
	holder, err := st.ShardOfEdge(edge.ID)
	if err != nil {
		t.Fatalf("edge %q not found in any shard after rotate: %v", edge.ID, err)
	}
	if holder != newID {
		t.Fatalf("new edge landed in shard %d, want new generation %d", holder, newID)
	}

	// The sealed generation must refuse a direct write so stale routes surface
	// as an error instead of silently leaking into a frozen shard.
	oldShard, ok := st.Shard(oldID)
	if !ok {
		t.Fatalf("old shard %d missing", oldID)
	}
	leak := model.NewEdge("e-leak", "a", "c", "feeds")
	if err := oldShard.Add(leak); err == nil {
		t.Fatalf("sealed shard %d accepted a write; frozen generation must reject edges", oldID)
	}
}

// routerBucket mirrors the router's bucket computation without exporting it.
func routerBucket(st *Store, key string) int {
	return st.router.Bucket(key)
}
