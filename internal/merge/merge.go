package merge

import (
	"context"
	"errors"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// Continuation is a path that crossed a shard boundary. The walker hands it
// to the merger together with the depth reached so far; the merger keeps
// expanding it while honouring the same depth budget.
type Continuation struct {
	Vertex string
	Depth  int
	Path   model.Path
	Shard  int
}

// Input bundles everything the merger needs for one traversal result.
type Input struct {
	Paths         []*model.Path
	Continuations []Continuation
	MaxDepth      int
	Store         *store.Store
	Visited       map[string]bool
	BatchSize     int
}

// Output is the merged result plus its paginated batches.
type Output struct {
	Paths   []*model.Path
	Batches [][]*model.Path
}

// Merge combines per-shard paths and cross-shard continuations into one
// ordered result and slices it into batches. Cancellation is checked while
// continuations are expanded so a disconnected client stops the traversal.
func Merge(ctx context.Context, input *Input) (*Output, error) {
	if input == nil {
		return nil, errors.New("merge: nil input")
	}
	paths := make([]*model.Path, 0, len(input.Paths)+len(input.Continuations))
	paths = append(paths, input.Paths...)
	queue := make([]Continuation, len(input.Continuations))
	copy(queue, input.Continuations)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := queue[0]
		queue = queue[1:]
		if input.Visited[current.Vertex] {
			continue
		}
		input.Visited[current.Vertex] = true
		path := current.Path
		path.Depth = current.Depth
		paths = append(paths, &path)
		if current.Depth >= input.MaxDepth {
			continue
		}
		groups := input.Store.EdgesFromByShard(current.Vertex)
		for _, edges := range groups {
			for _, edge := range edges {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if input.Visited[edge.To] {
					continue
				}
				queue = append(queue, Continuation{
					Vertex: edge.To,
					Depth:  1,
					Path:   current.Path.Append(edge.To, edge.ID),
					Shard:  current.Shard,
				})
			}
		}
	}
	if paths == nil {
		paths = []*model.Path{}
	}
	batchSize := input.BatchSize
	if batchSize <= 0 {
		batchSize = 64
	}
	batches := BatchPaths(paths, batchSize)
	return &Output{Paths: paths, Batches: batches}, nil
}
