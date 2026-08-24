package wal

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRotateReleasesSegmentHandle reproduces the handle-leak symptom: before
// the fix every Rotate left the previous segment's file handle open, so
// OpenHandles climbed without bound and the segment files could not be
// removed on platforms that lock open files (Windows). After the fix only the
// current segment holds a handle and rotated segments are deletable.
func TestRotateReleasesSegmentHandle(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	// First Append forces segment 0 to be opened.
	if err := w.Append(Record{Op: "vertex", ID: "v1"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if got := w.OpenHandles(); got != 1 {
		t.Fatalf("after first append: want 1 open handle, got %d", got)
	}
	prevPath := w.current.Path()

	// Rotate away segment 0 and write into the new segment a few times.
	for i := 0; i < 3; i++ {
		if err := w.Rotate(); err != nil {
			t.Fatalf("rotate %d: %v", i, err)
		}
		if err := w.Append(Record{Op: "vertex", ID: "v"}); err != nil {
			t.Fatalf("append after rotate %d: %v", i, err)
		}
		// Only the live segment should hold a handle regardless of how many
		// rotations have happened.
		if got := w.OpenHandles(); got != 1 {
			t.Fatalf("after rotate %d: want 1 open handle, got %d", i, got)
		}
	}

	// The rotated-out segment must now be deletable — the whole point of
	// closing handles promptly. On Windows this fails if the handle leaks.
	if err := os.Remove(prevPath); err != nil {
		t.Fatalf("remove rotated segment %q: %v", prevPath, err)
	}
}

// TestRotatePreservesSegmentCount ensures rotation still records every segment
// for replay even though handles are released.
func TestRotatePreservesSegmentCount(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	for i := 0; i < 4; i++ {
		if err := w.Append(Record{Op: "vertex", ID: "v"}); err != nil {
			t.Fatalf("append: %v", err)
		}
		if err := w.Rotate(); err != nil {
			t.Fatalf("rotate: %v", err)
		}
	}
	if got, want := w.SegmentCount(), 5; got != want {
		t.Fatalf("segment count: want %d, got %d", want, got)
	}
	if got := w.OpenHandles(); got != 1 {
		t.Fatalf("open handles: want 1, got %d", got)
	}
}

// TestSegmentCloseIdempotent verifies that closing a segment twice (as can
// happen when Rotate closes a segment and Close later walks the open list) is
// harmless.
func TestSegmentCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	seg, err := OpenSegment(dir, 0)
	if err != nil {
		t.Fatalf("open segment: %v", err)
	}
	if err := seg.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := seg.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

// TestReplayAfterRotation replays a WAL that has been rotated to confirm that
// closing handles on rotation does not break recovery (replay opens files
// independently by path).
func TestReplayAfterRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}

	want := []string{"a", "b", "c"}
	if err := w.Append(Record{Op: "vertex", ID: want[0]}); err != nil {
		t.Fatalf("append 0: %v", err)
	}
	if err := w.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := w.Append(Record{Op: "vertex", ID: want[1]}); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := w.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := w.Append(Record{Op: "vertex", ID: want[2]}); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	// Handles are released for rotated segments; only the current one stays open.
	if got := w.OpenHandles(); got != 1 {
		t.Fatalf("open handles before replay: want 1, got %d", got)
	}

	var got []string
	if err := w.Replay(func(r Record) error {
		got = append(got, r.ID)
		return nil
	}); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("replay: want %d records, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("replay record %d: want %q, got %q", i, want[i], got[i])
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// All segment files exist on disk for the next restart.
	matches, err := filepath.Glob(filepath.Join(dir, "wal-*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("segment files on disk: want 3, got %d", len(matches))
	}
}
