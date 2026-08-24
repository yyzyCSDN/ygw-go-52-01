package rebuild

import (
	"context"
	"errors"
	"testing"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// Reproduces the bug: a rebuild that overflows the per-label cap must return an
// error and leave the previously committed index untouched (rollback).
func TestVerifyRebuildRollbackOnCapOverflow(t *testing.T) {
	st, err := store.New(store.Options{LabelCap: 2})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	// three vertices under the same label -> exceeds cap of 2
	for _, id := range []string{"a", "b", "c"} {
		if err := st.PutVertex(model.NewVertex(id, "station")); err != nil {
			t.Fatalf("put %s: %v", id, err)
		}
	}
	// seed a known-good index first so rollback can be observed
	rb := New(st, st.LabelIndex())
	if err := st.LabelIndex().Add("station", "seed"); err != nil {
		t.Fatalf("seed add: %v", err)
	}
	if got := st.LookupLabel("station"); len(got) != 1 || got[0] != "seed" {
		t.Fatalf("seed index wrong: %v", got)
	}
	err = rb.Rebuild(context.Background())
	if err == nil {
		t.Fatalf("expected rebuild to fail on cap overflow, got nil")
	}
	if !errors.Is(err, model.ErrIndexFull) {
		t.Fatalf("expected ErrIndexFull, got %v", err)
	}
	// rollback: the previously committed index must remain in place, not the
	// partial two-entry index that a buggy rebuild would have published.
	got := st.LookupLabel("station")
	if len(got) != 1 || got[0] != "seed" {
		t.Fatalf("rollback failed: index mutated to %v (expected [seed])", got)
	}
}

// A clean rebuild (no overflow) must still publish the fresh index.
func TestVerifyRebuildSuccessReplacesIndex(t *testing.T) {
	st, err := store.New(store.Options{LabelCap: 16})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	for _, id := range []string{"a", "b"} {
		if err := st.PutVertex(model.NewVertex(id, "station")); err != nil {
			t.Fatalf("put %s: %v", id, err)
		}
	}
	rb := New(st, st.LabelIndex())
	if err := rb.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	got := st.LookupLabel("station")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %v", got)
	}
}
