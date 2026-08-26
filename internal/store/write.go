package store

import (
	"fmt"

	"graphstore/internal/model"
	"graphstore/internal/wal"
)

// PutVertex stores a vertex and updates the incremental attribute index.
func (s *Store) PutVertex(vertex *model.Vertex) error {
	if s.closed {
		return model.ErrStoreClosed
	}
	if vertex == nil || vertex.ID == "" {
		return fmt.Errorf("store: vertex id is required")
	}
	if _, exists := s.vertices[vertex.ID]; exists {
		return fmt.Errorf("store: vertex %q already exists", vertex.ID)
	}
	if s.wal != nil {
		if err := s.wal.Append(wal.Record{
			Op:    "vertex",
			ID:    vertex.ID,
			Label: vertex.Label,
			Attrs: vertex.Attrs,
		}); err != nil {
			return fmt.Errorf("store: append vertex wal: %w", err)
		}
		s.metrics.WalAppends.Inc()
	}
	s.vertices[vertex.ID] = vertex
	s.vertexSeq = append(s.vertexSeq, vertex.ID)
	s.attrIndex.UpdateVertex(vertex)
	s.metrics.VertexWrites.Inc()
	return nil
}

// PutEdge routes the edge by hash to its bucket's active shard, appends the
// change to the WAL and updates the attribute index.
func (s *Store) PutEdge(edge *model.Edge) error {
	if s.closed {
		return model.ErrStoreClosed
	}
	if edge == nil || edge.ID == "" {
		return fmt.Errorf("store: edge id is required")
	}
	if _, ok := s.vertices[edge.From]; !ok {
		return fmt.Errorf("%w: source %q", model.ErrVertexNotFound, edge.From)
	}
	if _, ok := s.vertices[edge.To]; !ok {
		return fmt.Errorf("%w: target %q", model.ErrVertexNotFound, edge.To)
	}
	target := s.router.ShardForKey(edge.Key())
	if target == nil {
		return fmt.Errorf("store: no shard for edge %q", edge.Key())
	}
	if err := target.Add(edge); err != nil {
		return err
	}
	if s.wal != nil {
		if err := s.wal.Append(wal.Record{
			Op:    "edge",
			ID:    edge.ID,
			From:  edge.From,
			To:    edge.To,
			Type:  edge.Type,
			Attrs: edge.Attrs,
		}); err != nil {
			return fmt.Errorf("store: append edge wal: %w", err)
		}
		s.metrics.WalAppends.Inc()
	}
	s.attrIndex.Update(edge)
	s.metrics.EdgeWrites.Inc()
	return nil
}

// PutEdgeToBucket writes an edge into an explicit bucket, bypassing routing.
// It is used by the demo to build deterministic cross-shard topologies.
func (s *Store) PutEdgeToBucket(edge *model.Edge, bucket int) error {
	if s.closed {
		return model.ErrStoreClosed
	}
	if bucket < 0 || bucket >= s.opts.EdgeBuckets {
		return fmt.Errorf("store: bucket %d out of range", bucket)
	}
	target := s.router.ShardForBucket(bucket)
	if target == nil {
		return fmt.Errorf("store: no shard for bucket %d", bucket)
	}
	if err := target.Add(edge); err != nil {
		return err
	}
	s.attrIndex.Update(edge)
	s.metrics.EdgeWrites.Inc()
	return nil
}
