package store

import (
	"sort"

	"graphstore/internal/model"
)

// GetVertex returns the vertex and whether it exists.
func (s *Store) GetVertex(id string) (*model.Vertex, bool) {
	vertex, ok := s.vertices[id]
	return vertex, ok
}

// VertexCount returns the number of stored vertices.
func (s *Store) VertexCount() int {
	return len(s.vertices)
}

// EdgeCount returns the number of stored edges across all shards.
func (s *Store) EdgeCount() int {
	total := 0
	for _, shard := range s.manager.AllShards() {
		total += shard.EdgeCount()
	}
	return total
}

// AllVertices returns the vertices in insertion order.
func (s *Store) AllVertices() []*model.Vertex {
	out := make([]*model.Vertex, 0, len(s.vertexSeq))
	for _, id := range s.vertexSeq {
		if vertex, ok := s.vertices[id]; ok {
			out = append(out, vertex)
		}
	}
	return out
}

// GetEdge returns an edge and whether it exists.
func (s *Store) GetEdge(id string) (*model.Edge, bool) {
	for _, shard := range s.manager.AllShards() {
		if edge, ok := shard.Get(id); ok {
			return edge, true
		}
	}
	return nil, false
}

// EdgesFrom returns every outgoing edge of the vertex across all shards.
func (s *Store) EdgesFrom(vertex string) []*model.Edge {
	var out []*model.Edge
	for _, shard := range s.manager.AllShards() {
		out = append(out, shard.EdgesFrom(vertex)...)
	}
	return out
}

// EdgesFromByShard groups the outgoing edges of a vertex by shard id.
func (s *Store) EdgesFromByShard(vertex string) map[int][]*model.Edge {
	out := make(map[int][]*model.Edge)
	for _, shard := range s.manager.AllShards() {
		edges := shard.EdgesFrom(vertex)
		if len(edges) > 0 {
			out[shard.ID] = append(out[shard.ID], edges...)
		}
	}
	return out
}

// EdgesTo returns every incoming edge of the vertex across all shards.
func (s *Store) EdgesTo(vertex string) []*model.Edge {
	var out []*model.Edge
	for _, shard := range s.manager.AllShards() {
		out = append(out, shard.EdgesTo(vertex)...)
	}
	return out
}

// LookupByAttr resolves edges by attribute through the inverted index.
func (s *Store) LookupByAttr(key, value string) []string {
	return s.attrIndex.LookupByAttr(key, value)
}

// LookupLabel resolves traversal entry vertices by label.
func (s *Store) LookupLabel(label string) []string {
	return s.labelIndex.Lookup(label)
}

// ShardOfVertex assigns a vertex a home shard by hashing its id.
func (s *Store) ShardOfVertex(vertexID string) int {
	bucket := s.router.Bucket(vertexID)
	if shard := s.router.ShardForBucket(bucket); shard != nil {
		return shard.ID
	}
	return 0
}

// SortedEdgeIDs returns every edge id in lexical order.
func (s *Store) SortedEdgeIDs() []string {
	ids := make([]string, 0, s.EdgeCount())
	for _, shard := range s.manager.AllShards() {
		for _, edge := range shard.All() {
			ids = append(ids, edge.ID)
		}
	}
	sort.Strings(ids)
	return ids
}
