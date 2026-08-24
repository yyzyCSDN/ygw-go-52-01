package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Replay reads every record of every segment in order and invokes fn for
// each record. Replaying is read-only: no segment handle is left open.
func (w *WAL) Replay(fn func(Record) error) error {
	segments := w.Segments()
	sort.Slice(segments, func(i, j int) bool { return segments[i].ID < segments[j].ID })
	for _, segment := range segments {
		if err := replaySegment(segment, fn); err != nil {
			return err
		}
	}
	return nil
}

func replaySegment(segment *Segment, fn func(Record) error) error {
	file, err := os.Open(segment.Path())
	if err != nil {
		return fmt.Errorf("wal: open segment %d for replay: %w", segment.ID, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var record Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("wal: decode record in segment %d: %w", segment.ID, err)
		}
		if err := fn(record); err != nil {
			return err
		}
	}
	return scanner.Err()
}
