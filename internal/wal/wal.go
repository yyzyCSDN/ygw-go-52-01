package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Record is one change applied through the write pipeline. The WAL persists
// records so the store can be replayed after a restart.
type Record struct {
	Op     string            `json:"op"`
	ID     string            `json:"id"`
	From   string            `json:"from,omitempty"`
	To     string            `json:"to,omitempty"`
	Type   string            `json:"type,omitempty"`
	Label  string            `json:"label,omitempty"`
	Attrs  map[string]string `json:"attrs,omitempty"`
	Seq    uint64            `json:"seq"`
}

// WAL appends records to a sequence of segment files and replays them in
// order. Only the current segment keeps an open handle; every earlier
// segment is closed as soon as rotation finishes.
type WAL struct {
	dir      string
	segments []*Segment
	open     []*Segment
	current  *Segment
	nextID   int
	seq      uint64
	mu       sync.Mutex
	closed   bool
}

// Open creates a WAL rooted at dir, discovering existing segments.
func Open(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal: create dir: %w", err)
	}
	wal := &WAL{dir: dir}
	if err := wal.discover(); err != nil {
		return nil, err
	}
	return wal, nil
}

func (w *WAL) discover() error {
	matches, err := filepath.Glob(filepath.Join(w.dir, "wal-*.log"))
	if err != nil {
		return fmt.Errorf("wal: list segments: %w", err)
	}
	for _, path := range matches {
		var id int
		if _, scanErr := fmt.Sscanf(filepath.Base(path), "wal-%d.log", &id); scanErr != nil {
			continue
		}
		w.segments = append(w.segments, &Segment{ID: id, path: path})
		if id >= w.nextID {
			w.nextID = id + 1
		}
	}
	return nil
}

// ensureCurrent opens the current segment for writing, creating it when the
// directory is empty.
func (w *WAL) ensureCurrent() error {
	if w.current != nil {
		return nil
	}
	segment, err := OpenSegment(w.dir, w.nextID)
	if err != nil {
		return err
	}
	w.nextID++
	w.current = segment
	w.segments = append(w.segments, segment)
	w.open = append(w.open, segment)
	return nil
}

// Append writes one record to the current segment and assigns its sequence.
func (w *WAL) Append(record Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("wal: closed")
	}
	if err := w.ensureCurrent(); err != nil {
		return err
	}
	w.seq++
	record.Seq = w.seq
	if err := w.current.Append(record); err != nil {
		return err
	}
	return w.current.Sync()
}

// Rotate starts a new segment. The previous segment is closed so its file
// handle is released and the file can be archived or deleted.
func (w *WAL) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("wal: closed")
	}
	if err := w.ensureCurrent(); err != nil {
		return err
	}
	next, err := OpenSegment(w.dir, w.nextID)
	if err != nil {
		return err
	}
	w.nextID++
	prev := w.current
	w.current = next
	w.segments = append(w.segments, next)
	w.open = append(w.open, next)
	// The rotated-out segment is no longer the write target, so release its
	// file handle now. Without this every rotation leaks a handle and the
	// segment file stays open until the whole WAL shuts down, which is why
	// the open-handle count climbs and old segments cannot be deleted.
	if err := prev.Close(); err != nil {
		return fmt.Errorf("wal: close rotated segment %d: %w", prev.ID, err)
	}
	w.open = removeSegment(w.open, prev)
	return nil
}

// OpenHandles returns the number of segment files with an open handle.
func (w *WAL) OpenHandles() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.open)
}

// Segments returns the ordered segment files of the WAL.
func (w *WAL) Segments() []*Segment {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]*Segment(nil), w.segments...)
}

// SegmentCount returns the number of segments ever created.
func (w *WAL) SegmentCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.segments)
}

// Seq returns the current sequence number.
func (w *WAL) Seq() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.seq
}

// Close shuts the WAL down and releases every open handle.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	var firstErr error
	for _, segment := range w.open {
		if err := segment.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.open = nil
	w.current = nil
	return firstErr
}

// removeSegment returns open with the first occurrence of target removed,
// preserving the order of the remaining entries.
func removeSegment(open []*Segment, target *Segment) []*Segment {
	for i, seg := range open {
		if seg == target {
			return append(open[:i], open[i+1:]...)
		}
	}
	return open
}
