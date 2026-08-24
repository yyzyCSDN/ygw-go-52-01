package model

// Edge connects two vertices in one direction and carries a type plus
// attributes. Edges are routed to shards by the hash of Key.
type Edge struct {
	ID    string
	From  string
	To    string
	Type  string
	Attrs map[string]string
}

// NewEdge builds an edge with an empty attribute map.
func NewEdge(id, from, to, edgeType string) *Edge {
	return &Edge{ID: id, From: from, To: to, Type: edgeType, Attrs: map[string]string{}}
}

// WithAttr returns a copy of the edge with one extra attribute.
func (e *Edge) WithAttr(key, value string) *Edge {
	attrs := make(map[string]string, len(e.Attrs)+1)
	for k, val := range e.Attrs {
		attrs[k] = val
	}
	attrs[key] = value
	return &Edge{ID: e.ID, From: e.From, To: e.To, Type: e.Type, Attrs: attrs}
}

// Key returns the routing key used by the shard router.
func (e *Edge) Key() string {
	return e.From + ">" + e.Type + ">" + e.To
}
