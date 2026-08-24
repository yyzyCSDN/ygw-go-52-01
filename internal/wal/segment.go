package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Segment is one append-only WAL file. A segment owns exactly one file
// handle while it is open; after rotation the handle must be released so the
// file can be reclaimed by the operating system.
type Segment struct {
	ID     int
	path   string
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
	closed bool
	size   int64
}

// OpenSegment opens or creates the segment file for the given id.
func OpenSegment(dir string, id int) (*Segment, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal: create dir: %w", err)
	}
	path := filepath.Join(dir, segmentFileName(id))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("wal: open segment %d: %w", id, err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("wal: stat segment %d: %w", id, err)
	}
	return &Segment{
		ID:     id,
		path:   path,
		file:   file,
		writer: bufio.NewWriter(file),
		size:   info.Size(),
	}, nil
}

// Path returns the absolute segment file path.
func (s *Segment) Path() string {
	return s.path
}

// IsClosed reports whether the underlying handle has been released.
func (s *Segment) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Append writes one record to the segment. The write is buffered; callers
// use Sync to guarantee durability before acknowledging a change.
func (s *Segment) Append(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("wal: segment %d is closed", s.ID)
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("wal: encode record: %w", err)
	}
	payload = append(payload, '\n')
	if _, err := s.writer.Write(payload); err != nil {
		return fmt.Errorf("wal: write segment %d: %w", s.ID, err)
	}
	s.size += int64(len(payload))
	return nil
}

// Sync flushes the buffered writer to disk.
func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if err := s.writer.Flush(); err != nil {
		return fmt.Errorf("wal: flush segment %d: %w", s.ID, err)
	}
	return nil
}

// Size returns the number of bytes appended to the segment.
func (s *Segment) Size() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

// Close releases the file handle. Closing is idempotent so rotation and
// shutdown can both call it safely.
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if s.writer != nil {
		if err := s.writer.Flush(); err != nil {
			return fmt.Errorf("wal: flush before close segment %d: %w", s.ID, err)
		}
	}
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return fmt.Errorf("wal: close segment %d: %w", s.ID, err)
		}
	}
	s.closed = true
	s.file = nil
	s.writer = nil
	return nil
}

func segmentFileName(id int) string {
	return fmt.Sprintf("wal-%06d.log", id)
}
