package model

// Vertex is a graph vertex with a stable identifier, an optional label used
// as a traversal entry point, and a set of attributes.
type Vertex struct {
	ID    string
	Label string
	Attrs map[string]string
}

// NewVertex builds a vertex with an empty attribute map so callers can
// populate attributes without nil-map assignments.
func NewVertex(id, label string) *Vertex {
	return &Vertex{ID: id, Label: label, Attrs: map[string]string{}}
}
