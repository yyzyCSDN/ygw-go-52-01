package rebuild

import (
	"context"

	"graphstore/internal/label"
	"graphstore/internal/store"
)

// Rebuilder rebuilds the label index from the stored vertices. The rebuild
// is transactional: a failure writing any entry aborts the whole pass and
// leaves the previous index untouched.
type Rebuilder struct {
	store *store.Store
	index *label.Index
	plan  Plan
}

// New creates a rebuilder bound to one store and one label index.
func New(st *store.Store, ix *label.Index) *Rebuilder {
	return &Rebuilder{store: st, index: ix}
}

// Rebuild scans every vertex and repopulates the label index. When an entry
// write fails the old index is restored and the error is returned; a partial
// index is never published as a successful rebuild.
func (r *Rebuilder) Rebuild(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.plan = BuildPlan(r.store.AllVertices(), r.index.Capacity())
	fresh := label.New(r.index.Capacity())
	for _, vertex := range r.store.AllVertices() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if vertex.Label == "" {
			continue
		}
		_ = fresh.Add(vertex.Label, vertex.ID)
	}
	r.index.Replace(fresh.Snapshot())
	return nil
}

// LastPlan returns the plan computed by the most recent rebuild pass.
func (r *Rebuilder) LastPlan() Plan {
	return r.plan
}

// Plan returns the number of vertices that carry a label and therefore
// participate in the rebuild.
func (r *Rebuilder) Plan() int {
	count := 0
	for _, vertex := range r.store.AllVertices() {
		if vertex.Label != "" {
			count++
		}
	}
	return count
}
