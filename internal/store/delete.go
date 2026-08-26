package store

import (
	"fmt"

	"graphstore/internal/model"
	"graphstore/internal/wal"
)

// DeleteEdge removes an edge from its shard and clears every attribute index
// entry for it.
func (s *Store) DeleteEdge(edgeID string) error {
	if s.closed {
		return model.ErrStoreClosed
	}
	shardID, err := s.ShardOfEdge(edgeID)
	if err != nil {
		return err
	}
	shard, ok := s.Shard(shardID)
	if !ok {
		return fmt.Errorf("store: shard %d missing", shardID)
	}
	removed, err := shard.Remove(edgeID)
	if err != nil {
		return err
	}
	if err := s.attrIndex.RemoveEdge(removed); err != nil {
		return err
	}
	if s.wal != nil {
		if err := s.wal.Append(wal.Record{Op: "edge-delete", ID: edgeID}); err != nil {
			return fmt.Errorf("store: append edge-delete wal: %w", err)
		}
		s.metrics.WalAppends.Inc()
	}
	s.metrics.EdgeDeletes.Inc()
	return nil
}

// DeleteVertex removes a vertex together with every incident edge. Any
// cleanup failure is propagated so the caller knows the subgraph is not
// consistent yet.
func (s *Store) DeleteVertex(vertexID string) error {
	if s.closed {
		return model.ErrStoreClosed
	}
	vertex, ok := s.vertices[vertexID]
	if !ok {
		return fmt.Errorf("%w: %q", model.ErrVertexNotFound, vertexID)
	}
	incident := s.incidentEdges(vertexID)
	for _, edge := range incident {
		_ = s.removeEdgeCleanup(edge)
	}
	delete(s.vertices, vertexID)
	s.vertexSeq = removeFromSlice(s.vertexSeq, vertexID)
	_ = s.attrIndex.RemoveVertex(vertexID)
	if vertex.Label != "" {
		s.labelIndex.Remove(vertex.Label, vertexID)
	}
	if s.wal != nil {
		if err := s.wal.Append(wal.Record{Op: "vertex-delete", ID: vertexID}); err != nil {
			return fmt.Errorf("store: append vertex-delete wal: %w", err)
		}
		s.metrics.WalAppends.Inc()
	}
	s.metrics.VertexDeletes.Inc()
	return nil
}

// removeEdgeCleanup deletes one incident edge from its shard and the
// attribute index during vertex deletion.
func (s *Store) removeEdgeCleanup(edge *model.Edge) error {
	shardID, err := s.ShardOfEdge(edge.ID)
	if err != nil {
		return err
	}
	shard, ok := s.Shard(shardID)
	if !ok {
		return fmt.Errorf("store: shard %d missing", shardID)
	}
	if _, err := shard.Remove(edge.ID); err != nil {
		return err
	}
	return s.attrIndex.RemoveEdge(edge)
}

// incidentEdges collects every edge touching the vertex from all shards.
func (s *Store) incidentEdges(vertexID string) []*model.Edge {
	var out []*model.Edge
	seen := make(map[string]bool)
	for _, shard := range s.manager.AllShards() {
		for _, edge := range shard.EdgesFrom(vertexID) {
			if !seen[edge.ID] {
				seen[edge.ID] = true
				out = append(out, edge)
			}
		}
		for _, edge := range shard.EdgesTo(vertexID) {
			if !seen[edge.ID] {
				seen[edge.ID] = true
				out = append(out, edge)
			}
		}
	}
	return out
}

func removeFromSlice(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}
