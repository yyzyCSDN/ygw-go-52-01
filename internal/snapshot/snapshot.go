package snapshot

import "time"

// WalkState captures the parts of a traversal session that survive shard
// rotation and vertex deletion: the visited marks and the frontier.
type WalkState struct {
	Visited  []string
	Frontier []string
	Depth    int
}

// Snapshot is a point-in-time view of the store and one walk session. It is
// used to align session state with deletions that happen mid-traversal.
type Snapshot struct {
	VertexCount int
	ShardCount  int
	EdgeCount   int
	State       WalkState
	TakenAt     int64
}

// New builds a snapshot for the supplied counts.
func New(vertexCount, shardCount, edgeCount int, state WalkState) *Snapshot {
	return &Snapshot{
		VertexCount: vertexCount,
		ShardCount:  shardCount,
		EdgeCount:   edgeCount,
		State:       state,
		TakenAt:     time.Now().UnixNano(),
	}
}

// AgeMillis returns how many milliseconds elapsed since the snapshot.
func (s *Snapshot) AgeMillis(now int64) int64 {
	return (now - s.TakenAt) / int64(time.Millisecond)
}
