package walk

import (
	"sort"

	"graphstore/internal/model"
)

// NormalizePaths guarantees a non-nil empty result so downstream consumers
// can iterate or index the slice without a nil dereference.
func NormalizePaths(paths []*model.Path) []*model.Path {
	if paths == nil {
		return []*model.Path{}
	}
	return paths
}

// DistinctVertices returns the unique vertices visited by the paths.
func DistinctVertices(paths []*model.Path) []string {
	seen := make(map[string]bool)
	for _, path := range paths {
		for _, vertex := range path.Vertices {
			seen[vertex] = true
		}
	}
	vertices := make([]string, 0, len(seen))
	for vertex := range seen {
		vertices = append(vertices, vertex)
	}
	sort.Strings(vertices)
	return vertices
}

// MaxDepthReached returns the deepest path in the result.
func MaxDepthReached(paths []*model.Path) int {
	maxDepth := 0
	for _, path := range paths {
		if path.Depth > maxDepth {
			maxDepth = path.Depth
		}
	}
	return maxDepth
}
