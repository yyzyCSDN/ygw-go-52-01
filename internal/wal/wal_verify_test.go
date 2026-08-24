package wal_test

import (
	"fmt"
	"testing"

	"graphstore/internal/wal"
)

// TestWALSegmentHandleClosed verifies rotation releases the previous segment
// handle so exactly one segment stays open.
func TestWALSegmentHandleClosed(t *testing.T) {
	dir := t.TempDir()
	log, err := wal.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	for i := 0; i < 3; i++ {
		if err := log.Append(wal.Record{Op: "edge", ID: fmt.Sprintf("e%d", i)}); err != nil {
			t.Fatal(err)
		}
		if err := log.Rotate(); err != nil {
			t.Fatal(err)
		}
	}
	if got := log.OpenHandles(); got != 1 {
		t.Fatalf("expected exactly one open segment handle after rotation, got %d", got)
	}
}
