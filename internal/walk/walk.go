package walk

import (
	"context"
	"fmt"

	"graphstore/internal/merge"
	"graphstore/internal/metric"
	"graphstore/internal/model"
	"graphstore/internal/snapshot"
	"graphstore/internal/store"
)

// Options configures the walker.
type Options struct {
	BatchSize int
}

// Walker runs depth-limited traversals over the store. It owns one session
// whose visited marks are reset between traversals.
type Walker struct {
	store    *store.Store
	metrics  *metric.Registry
	capturer *snapshot.Capturer
	opts     Options
	session  *Session
}

// New creates a walker bound to one store.
func New(st *store.Store, opts Options) *Walker {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 64
	}
	return &Walker{
		store:    st,
		metrics:  st.Metrics(),
		capturer: snapshot.NewCapturer(func(vertexID string) bool {
			_, ok := st.GetVertex(vertexID)
			return ok
		}),
		opts: opts,
	}
}

// Walk performs a depth-limited traversal from the start vertex and returns
// every path found, merged across shard boundaries.
func (w *Walker) Walk(ctx context.Context, start string, maxDepth int) ([]*model.Path, error) {
	w.ensureSession()
	if _, ok := w.store.GetVertex(start); !ok {
		return []*model.Path{}, fmt.Errorf("%w: %q", model.ErrVertexNotFound, start)
	}
	w.metrics.WalkStarts.Inc()
	w.session.Transition(StateExpanding)
	visited := w.session.Marks()
	w.session.Mark(start)
	startShard := w.store.ShardOfVertex(start)
	state := w.capturer.Capture(visited, []string{start}, 0)
	snap := snapshot.New(w.store.VertexCount(), len(w.store.EdgeShards()), w.store.EdgeCount(), state)
	w.session.AttachSnapshot(snap)
	w.metrics.SnapshotCaptures.Inc()
	queue := []frontierItem{{
		vertex: start,
		depth:  0,
		path:   model.NewPath(start),
		shard:  startShard,
	}}
	results := make([]*model.Path, 0, 64)
	var continuations []merge.Continuation
	for len(queue) > 0 {
		if err := stopWalk(ctx, w.metrics); err != nil {
			return nil, err
		}
		current := queue[0]
		queue = queue[1:]
		path := current.path
		path.Depth = current.depth
		results = append(results, &path)
		if current.depth >= maxDepth {
			continue
		}
		groups := w.store.EdgesFromByShard(current.vertex)
		sameShard := groups[current.shard]
		for _, edge := range sameShard {
			if err := stopWalk(ctx, w.metrics); err != nil {
				return nil, err
			}
			if visited[edge.To] {
				continue
			}
			w.session.Mark(edge.To)
			queue = append(queue, frontierItem{
				vertex: edge.To,
				depth:  current.depth + 1,
				path:   current.path.Append(edge.To, edge.ID),
				shard:  current.shard,
			})
		}
		for shardID, edges := range groups {
			if shardID == current.shard {
				continue
			}
			for _, edge := range edges {
				if err := stopWalk(ctx, w.metrics); err != nil {
					return nil, err
				}
				continuations = append(continuations, crossShardContinuation(current, edge, shardID))
			}
		}
	}
	w.session.Transition(StateDraining)
	output, err := merge.Merge(ctx, &merge.Input{
		Paths:         results,
		Continuations: continuations,
		MaxDepth:      maxDepth,
		Store:         w.store,
		Visited:       visited,
		BatchSize:     w.opts.BatchSize,
	})
	if err != nil {
		return nil, err
	}
	w.capturer.PruneDeletes(visited)
	w.metrics.WalkPaths.Add(int64(len(output.Paths)))
	w.metrics.WalkVisitedMarks.Set(int64(w.session.MarkCount()))
	return NormalizePaths(output.Paths), nil
}

// crossShardContinuation hands a path that crossed a shard boundary to the
// merger, carrying the depth reached so far so the traversal budget survives
// the hop between generations.
func crossShardContinuation(current frontierItem, edge *model.Edge, shardID int) merge.Continuation {
	return merge.Continuation{
		Vertex: edge.To,
		Depth:  current.depth + 1,
		Path:   current.path.Append(edge.To, edge.ID),
		Shard:  shardID,
	}
}

// WalkFromLabel starts a traversal from every entry vertex registered under
// the label. An empty label index yields an empty result, never a nil slice.
func (w *Walker) WalkFromLabel(ctx context.Context, label string, maxDepth int) ([]*model.Path, error) {
	entries := w.store.LookupLabel(label)
	results := make([]*model.Path, 0, len(entries))
	for _, entry := range entries {
		if err := stopWalk(ctx, w.metrics); err != nil {
			return nil, err
		}
		paths, err := w.Walk(ctx, entry, maxDepth)
		if err != nil {
			return nil, err
		}
		results = append(results, paths...)
	}
	return NormalizePaths(results), nil
}

// Batch slices a traversal result into pages of the requested size.
func (w *Walker) Batch(paths []*model.Path, pageSize int) [][]*model.Path {
	if pageSize <= 0 {
		pageSize = 3
	}
	if len(paths) == 0 {
		return [][]*model.Path{}
	}
	batches := make([][]*model.Path, 0, (len(paths)+pageSize-1)/pageSize)
	for start := 0; start < len(paths); start += pageSize {
		end := start + pageSize
		if end > len(paths) {
			end = len(paths)
		}
		batches = append(batches, paths[start:end])
	}
	return batches
}

// Close finishes the current session. The session is reset so the next walk
// starts with clean visited marks.
func (w *Walker) Close() {
	if w.session != nil {
		w.session.Close()
		w.session.Reset()
	}
}

// Session returns the active traversal session of the walker.
func (w *Walker) Session() *Session {
	return w.session
}

func (w *Walker) ensureSession() {
	if w.session == nil {
		w.session = newSession()
		return
	}
	w.session.Reset()
}

type frontierItem struct {
	vertex string
	depth  int
	path   model.Path
	shard  int
}
