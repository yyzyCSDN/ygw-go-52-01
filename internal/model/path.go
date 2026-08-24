package model

// Path is a walk result: the ordered list of visited vertices, the edges used
// to step between them, and the depth at which the walk stopped.
type Path struct {
	Vertices []string
	Edges    []string
	Depth    int
}

// NewPath starts a path at the given root vertex.
func NewPath(root string) Path {
	return Path{
		Vertices: []string{root},
		Edges:    []string{},
		Depth:    0,
	}
}

// Append returns a new path extended by one hop across the given edge.
func (p Path) Append(vertexID, edgeID string) Path {
	vertices := make([]string, len(p.Vertices)+1)
	copy(vertices, p.Vertices)
	vertices[len(p.Vertices)] = vertexID

	edges := make([]string, len(p.Edges)+1)
	copy(edges, p.Edges)
	edges[len(p.Edges)] = edgeID

	return Path{
		Vertices: vertices,
		Edges:    edges,
		Depth:    p.Depth + 1,
	}
}
