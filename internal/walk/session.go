package walk

import "graphstore/internal/snapshot"

// SessionState is the lifecycle of one traversal session.
//
//	open -> expanding -> draining -> closed
//
// A closed session must be reset before it is reused, otherwise the visited
// marks of the previous traversal keep blocking vertices.
type SessionState int

const (
	StateOpen SessionState = iota
	StateExpanding
	StateDraining
	StateClosed
)

// String returns a stable name for the session state.
func (s SessionState) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateExpanding:
		return "expanding"
	case StateDraining:
		return "draining"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Session tracks the visited marks and the state of one traversal. Walker
// owns a single session and resets it between traversals.
type Session struct {
	visited map[string]bool
	state   SessionState
	snap    *snapshot.Snapshot
}

// newSession creates a session with an empty visited map.
func newSession() *Session {
	return &Session{visited: make(map[string]bool), state: StateOpen}
}

// Marks returns the visited marks of the session.
func (s *Session) Marks() map[string]bool {
	return s.visited
}

// State returns the current session state.
func (s *Session) State() SessionState {
	return s.state
}

// Mark records a visited vertex.
func (s *Session) Mark(vertexID string) {
	s.visited[vertexID] = true
}

// Visited reports whether the vertex was visited in this session.
func (s *Session) Visited(vertexID string) bool {
	return s.visited[vertexID]
}

// Transition moves the session to the next lifecycle state.
func (s *Session) Transition(next SessionState) {
	s.state = next
}

// Reset clears the visited marks and reopens the session for reuse.
func (s *Session) Reset() {
	s.visited = make(map[string]bool)
	s.state = StateOpen
	s.snap = nil
}

// MarkCount returns how many vertices are currently marked as visited.
func (s *Session) MarkCount() int {
	return len(s.visited)
}

// Close marks the session closed. The marks are kept until the next reset so
// a drained result can be inspected after the fact.
func (s *Session) Close() {
	s.state = StateClosed
}

// AttachSnapshot records the store snapshot captured at traversal start.
func (s *Session) AttachSnapshot(snap *snapshot.Snapshot) {
	s.snap = snap
}

// Snapshot returns the snapshot captured when the session began.
func (s *Session) Snapshot() *snapshot.Snapshot {
	return s.snap
}
