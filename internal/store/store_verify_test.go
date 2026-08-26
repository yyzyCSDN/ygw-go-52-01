package store_test

import (
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// TestVertexDeleteCleanupErrorPropagated verifies vertex deletion surfaces a
// failure while cleaning up incident edges instead of reporting success.
func TestVertexDeleteCleanupErrorPropagated(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for _, id := range []string{"v-a", "v-b", "v-c"} {
		if err := st.PutVertex(model.NewVertex(id, "")); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.PutEdgeToBucket(model.NewEdge("e-ab", "v-a", "v-b", "link"), 0); err != nil {
		t.Fatal(err)
	}
	if err := st.PutEdgeToBucket(model.NewEdge("e-ac", "v-a", "v-c", "link"), 1); err != nil {
		t.Fatal(err)
	}
	// Freeze bucket 0 so its edge can no longer be removed.
	if _, _, err := st.RotateBucket(0); err != nil {
		t.Fatal(err)
	}
	st.ArchiveBucket(0)
	if err := st.DeleteVertex("v-a"); err == nil {
		t.Fatal("vertex deletion must report the incident edge cleanup failure")
	}
}
