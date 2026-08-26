package label

import (
	"fmt"
	"strings"

	"graphstore/internal/model"
)

// Index maps vertex labels to entry vertex ids. The index is a derived
// structure: it is populated by the rebuild job and consulted by the walker
// to locate traversal entry points.
type Index struct {
	entries map[string][]string
	cap     int
}

// New creates an empty label index with the given per-label entry cap.
func New(cap int) *Index {
	if cap <= 0 {
		cap = 4096
	}
	return &Index{entries: make(map[string][]string), cap: cap}
}

// Capacity returns the per-label entry cap.
func (ix *Index) Capacity() int {
	return ix.cap
}

// Add registers one vertex under one label. Adding beyond the configured cap
// fails so a rebuild cannot silently drop entries.
func (ix *Index) Add(label, vertexID string) error {
	if strings.TrimSpace(label) == "" || len(label) > 64 {
		return fmt.Errorf("%w: label %q", model.ErrInvalidLabel, label)
	}
	if strings.TrimSpace(vertexID) == "" {
		return fmt.Errorf("%w: empty vertex id", model.ErrInvalidLabel)
	}
	ids := ix.entries[label]
	for _, existing := range ids {
		if existing == vertexID {
			return nil
		}
	}
	if ix.remaining(label) <= 0 {
		return fmt.Errorf("%w: label %q already has %d entries", model.ErrIndexFull, label, len(ids))
	}
	ix.entries[label] = append(ids, vertexID)
	return nil
}

// remaining reports how many more entries the label can hold before the
// per-label cap is reached.
func (ix *Index) remaining(label string) int {
	return ix.cap - len(ix.entries[label])
}

// Remove drops one vertex from one label.
func (ix *Index) Remove(label, vertexID string) {
	ids := ix.entries[label]
	for index, existing := range ids {
		if existing == vertexID {
			ix.entries[label] = append(ids[:index], ids[index+1:]...)
			if len(ix.entries[label]) == 0 {
				delete(ix.entries, label)
			}
			return
		}
	}
}

// RemoveVertex drops the vertex from every label it is registered under.
func (ix *Index) RemoveVertex(vertexID string, labels []string) {
	for _, label := range labels {
		ix.Remove(label, vertexID)
	}
}

// Snapshot returns a deep copy of the current entries.
func (ix *Index) Snapshot() map[string][]string {
	out := make(map[string][]string, len(ix.entries))
	for label, ids := range ix.entries {
		out[label] = append([]string(nil), ids...)
	}
	return out
}

// Replace swaps the whole index content. It is used by rebuild commits and
// rollbacks.
func (ix *Index) Replace(entries map[string][]string) {
	ix.entries = make(map[string][]string, len(entries))
	for label, ids := range entries {
		ix.entries[label] = append([]string(nil), ids...)
	}
}

// EntryCount returns the total number of label-vertex registrations.
func (ix *Index) EntryCount() int {
	total := 0
	for _, ids := range ix.entries {
		total += len(ids)
	}
	return total
}
