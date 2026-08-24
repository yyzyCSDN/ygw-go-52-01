package store

import (
	"fmt"

	"graphstore/internal/model"
	"graphstore/internal/wal"
)

// replayWAL restores the in-memory graph from the persisted segments. It is
// invoked once at startup when the store opens an existing write-ahead log
// directory, before any live writes are accepted.
func (s *Store) replayWAL() error {
	if s.wal == nil {
		return nil
	}
	return s.wal.Replay(func(record wal.Record) error {
		switch record.Op {
		case "vertex":
			vertex := &model.Vertex{ID: record.ID, Label: record.Label, Attrs: record.Attrs}
			s.vertices[record.ID] = vertex
			s.vertexSeq = append(s.vertexSeq, record.ID)
			s.attrIndex.UpdateVertex(vertex)
		case "edge":
			edge := &model.Edge{ID: record.ID, From: record.From, To: record.To, Type: record.Type, Attrs: record.Attrs}
			target := s.router.ShardForKey(edge.Key())
			if target == nil {
				return fmt.Errorf("store: replay edge %q has no target shard", record.ID)
			}
			if err := target.Add(edge); err != nil {
				return err
			}
			s.attrIndex.Update(edge)
		case "edge-delete":
			if err := s.replayRemoveEdge(record.ID); err != nil {
				return err
			}
		case "vertex-delete":
			for _, edge := range s.incidentEdges(record.ID) {
				if err := s.replayRemoveEdge(edge.ID); err != nil {
					return err
				}
			}
			delete(s.vertices, record.ID)
			s.vertexSeq = removeFromSlice(s.vertexSeq, record.ID)
			_ = s.attrIndex.RemoveVertex(record.ID)
		default:
			return fmt.Errorf("store: replay unknown record op %q", record.Op)
		}
		s.metrics.WalReplays.Inc()
		return nil
	})
}

// replayRemoveEdge removes an edge during replay. Missing edges and missing
// index entries are tolerated because the WAL is the recovery source of
// truth and the index is rebuilt from the same records.
func (s *Store) replayRemoveEdge(edgeID string) error {
	shardID, err := s.ShardOfEdge(edgeID)
	if err != nil {
		return nil
	}
	shard, ok := s.Shard(shardID)
	if !ok {
		return fmt.Errorf("store: replay shard %d missing", shardID)
	}
	removed, err := shard.Remove(edgeID)
	if err != nil {
		return err
	}
	_ = s.attrIndex.RemoveEdge(removed)
	return nil
}
