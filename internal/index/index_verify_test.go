package index_test

import (
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// TestDeleteEdgeClearsIndexEntry verifies deleting an edge removes every
// attribute index entry for it.
func TestDeleteEdgeClearsIndexEntry(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutVertex(model.NewVertex("v-a", "")); err != nil {
		t.Fatal(err)
	}
	if err := st.PutVertex(model.NewVertex("v-b", "")); err != nil {
		t.Fatal(err)
	}
	edge := model.NewEdge("e1", "v-a", "v-b", "link").WithAttr("region", "west")
	if err := st.PutEdge(edge); err != nil {
		t.Fatal(err)
	}
	if got := st.LookupByAttr("region", "west"); len(got) != 1 {
		t.Fatalf("edge must be indexed before deletion, got %v", got)
	}
	if err := st.DeleteEdge("e1"); err != nil {
		t.Fatal(err)
	}
	if got := st.LookupByAttr("region", "west"); len(got) != 0 {
		t.Fatalf("deleted edge still returned by the index: %v", got)
	}
}
