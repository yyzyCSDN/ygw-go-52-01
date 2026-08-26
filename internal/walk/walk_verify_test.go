package walk_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
	"graphstore/internal/walk"
)

// TestWalkHonorsContextCancel verifies a cancelled traversal stops and
// returns the context error instead of running to completion.
func TestWalkHonorsContextCancel(t *testing.T) {
	st, err := store.New(store.Options{EdgeBuckets: 4, ShardCapacity: 100, LabelCap: 32})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i := 0; i < 10; i++ {
		if err := st.PutVertex(model.NewVertex(fmt.Sprintf("v%d", i), "")); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 9; i++ {
		edge := model.NewEdge(fmt.Sprintf("e%d", i), fmt.Sprintf("v%d", i), fmt.Sprintf("v%d", i+1), "feeds")
		if err := st.PutEdgeToBucket(edge, 0); err != nil {
			t.Fatal(err)
		}
	}
	walker := walk.New(st, walk.Options{BatchSize: 64})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = walker.Walk(ctx, "v0", 9)
	if err == nil {
		t.Fatal("cancelled walk must return an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
