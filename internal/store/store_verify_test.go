package store_test

import (
	"fmt"
	"testing"

	"github.com/cespare/xxhash/v2"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// edgeInBucket builds an edge whose routing key hashes into the bucket. The
// type suffix varies because the routing key is built from endpoints and
// type, not from the edge id.
func edgeInBucket(from, to string, bucket int) *model.Edge {
	for i := 0; i < 20000; i++ {
		edge := model.NewEdge(fmt.Sprintf("e-%d", i), from, to, fmt.Sprintf("knows-%d", i))
		if int(xxhash.Sum64String(edge.Key())%4) == bucket {
			return edge
		}
	}
	panic("no edge key found for bucket")
}

// TestEdgeWriteRoutesNewShardAfterSplit verifies that after a bucket rotates,
// the route table points new writes at the replacement shard and the sealed
// generation stops accepting edges.
func TestEdgeWriteRoutesNewShardAfterSplit(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i := 0; i < 4; i++ {
		if err := st.PutVertex(model.NewVertex(fmt.Sprintf("v%d", i), "")); err != nil {
			t.Fatal(err)
		}
	}
	before := edgeInBucket("v0", "v1", 0)
	if err := st.PutEdgeToBucket(before, 0); err != nil {
		t.Fatal(err)
	}
	oldID, newID, err := st.RotateBucket(0)
	if err != nil {
		t.Fatal(err)
	}
	if oldID == newID {
		t.Fatal("rotation must produce a replacement shard")
	}
	after := edgeInBucket("v2", "v3", 0)
	if err := st.PutEdge(after); err != nil {
		t.Fatal(err)
	}
	shardID, err := st.ShardOfEdge(after.ID)
	if err != nil {
		t.Fatal(err)
	}
	if shardID != newID {
		t.Fatalf("edge written after rotation landed in shard %d, want %d", shardID, newID)
	}
	old, ok := st.Shard(oldID)
	if !ok {
		t.Fatalf("old shard %d missing", oldID)
	}
	if old.EdgeCount() != 1 {
		t.Fatalf("sealed shard must not receive new edges, got %d edges", old.EdgeCount())
	}
}
