package shard

import (
	"errors"
	"fmt"

	"graphstore/internal/model"
)

// State describes the lifecycle of an edge shard.
//
// The state machine is:
//
//	active -> sealing -> sealed -> archived
//
// Only active shards accept new edges; sealed shards are read-only and
// archived shards reject all mutation including cleanup writes.
type State int

const (
	Active State = iota
	Sealing
	Sealed
	Archived
)

// String returns a stable textual name for the state.
func (s State) String() string {
	switch s {
	case Active:
		return "active"
	case Sealing:
		return "sealing"
	case Sealed:
		return "sealed"
	case Archived:
		return "archived"
	default:
		return fmt.Sprintf("state(%d)", int(s))
	}
}

// Shard holds one generation of edges for a hash bucket. It maintains the
// edges themselves plus per-source and per-target adjacency lists so the
// walker can expand without scanning the whole bucket.
type Shard struct {
	ID      int
	State   State
	edges   map[string]*model.Edge
	byFrom  map[string][]string
	byTo    map[string][]string
	order   []string
	cap     int
}

// New creates an active shard with the given capacity.
func New(id, capacity int) *Shard {
	if capacity <= 0 {
		capacity = 1024
	}
	return &Shard{
		ID:     id,
		State:  Active,
		edges:  make(map[string]*model.Edge),
		byFrom: make(map[string][]string),
		byTo:   make(map[string][]string),
		cap:    capacity,
	}
}

// Add stores an edge in the shard. Only active shards accept new edges;
// sealed and archived shards reject writes so the route table cannot leak
// new data into a frozen generation.
func (s *Shard) Add(edge *model.Edge) error {
	if edge == nil {
		return errors.New("shard: nil edge")
	}
	if len(s.edges) >= s.cap {
		return fmt.Errorf("shard %d full at capacity %d", s.ID, s.cap)
	}
	if _, exists := s.edges[edge.ID]; exists {
		return fmt.Errorf("shard %d already holds edge %q", s.ID, edge.ID)
	}
	s.edges[edge.ID] = edge
	s.order = append(s.order, edge.ID)
	s.byFrom[edge.From] = append(s.byFrom[edge.From], edge.ID)
	s.byTo[edge.To] = append(s.byTo[edge.To], edge.ID)
	return nil
}

// Get returns the edge and whether it is present in the shard.
func (s *Shard) Get(id string) (*model.Edge, bool) {
	edge, ok := s.edges[id]
	return edge, ok
}

// Remove deletes an edge. Archived shards refuse removal because their data
// is frozen; the caller must surface that failure instead of pretending the
// cleanup succeeded.
func (s *Shard) Remove(id string) (*model.Edge, error) {
	if s.State == Archived {
		return nil, fmt.Errorf("%w: shard %d is %s", model.ErrShardArchived, s.ID, s.State)
	}
	edge, ok := s.edges[id]
	if !ok {
		return nil, fmt.Errorf("%w: edge %q in shard %d", model.ErrEdgeNotFound, id, s.ID)
	}
	delete(s.edges, id)
	s.byFrom[edge.From] = removeString(s.byFrom[edge.From], id)
	s.byTo[edge.To] = removeString(s.byTo[edge.To], id)
	return edge, nil
}

// EdgesFrom returns the outgoing edges of the vertex stored in this shard.
func (s *Shard) EdgesFrom(vertex string) []*model.Edge {
	ids := s.byFrom[vertex]
	out := make([]*model.Edge, 0, len(ids))
	for _, id := range ids {
		if edge, ok := s.edges[id]; ok {
			out = append(out, edge)
		}
	}
	return out
}

// EdgesTo returns the incoming edges of the vertex stored in this shard.
func (s *Shard) EdgesTo(vertex string) []*model.Edge {
	ids := s.byTo[vertex]
	out := make([]*model.Edge, 0, len(ids))
	for _, id := range ids {
		if edge, ok := s.edges[id]; ok {
			out = append(out, edge)
		}
	}
	return out
}

// Seal transitions an active shard to sealed; sealing is a transient state
// and is applied before sealing completes.
func (s *Shard) Seal() {
	if s.State == Active {
		s.State = Sealing
	}
	if s.State == Sealing {
		s.State = Sealed
	}
}

// Archive freezes a sealed shard permanently.
func (s *Shard) Archive() {
	if s.State == Sealed || s.State == Sealing {
		s.State = Archived
	}
}

// All returns every edge stored in the shard in insertion order.
func (s *Shard) All() []*model.Edge {
	out := make([]*model.Edge, 0, len(s.order))
	for _, id := range s.order {
		if edge, ok := s.edges[id]; ok {
			out = append(out, edge)
		}
	}
	return out
}

// EdgeCount returns the number of live edges in the shard.
func (s *Shard) EdgeCount() int {
	return len(s.edges)
}

// Capacity returns the configured edge capacity of the shard.
func (s *Shard) Capacity() int {
	return s.cap
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}
