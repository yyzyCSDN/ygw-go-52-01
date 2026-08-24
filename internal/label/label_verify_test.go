package label_test

import (
	"context"
	"fmt"
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/rebuild"
	"graphstore/internal/store"
)

// TestIndexRebuildErrorNotSwallowed verifies a rebuild that hits an entry
// write failure reports the error and leaves the previous index untouched.
func TestIndexRebuildErrorNotSwallowed(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i := 0; i < 3; i++ {
		if err := st.PutVertex(model.NewVertex(fmt.Sprintf("v%d", i), "team")); err != nil {
			t.Fatal(err)
		}
	}
	rebuilder := rebuild.New(st, st.LabelIndex())
	if err := rebuilder.Rebuild(context.Background()); err == nil {
		t.Fatal("rebuild must report the index entry write failure")
	}
	if got := st.LabelIndex().Lookup("team"); len(got) != 0 {
		t.Fatalf("index changed after a failed rebuild: %v", got)
	}
}
