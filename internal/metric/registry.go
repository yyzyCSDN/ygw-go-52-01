package metric

// Registry owns every counter exposed by the graph store. The HTTP layer
// renders the registry so operators can watch write and traversal load.
type Registry struct {
	EdgeWrites         *Counter
	EdgeDeletes        *Counter
	VertexWrites       *Counter
	VertexDeletes      *Counter
	WalAppends         *Counter
	WalRotations       *Counter
	WalReplays         *Counter
	WalSegmentsOpen    *Gauge
	WalkStarts         *Counter
	WalkCancels        *Counter
	WalkPaths          *Counter
	WalkVisitedMarks   *Gauge
	ShardRotations     *Counter
	IndexRebuilds      *Counter
	IndexRebuildErrors *Counter
	SnapshotCaptures   *Counter
	HTTPRequests       *Counter
}

// New creates a registry with all counters initialised.
func New() *Registry {
	return &Registry{
		EdgeWrites:         &Counter{},
		EdgeDeletes:        &Counter{},
		VertexWrites:       &Counter{},
		VertexDeletes:      &Counter{},
		WalAppends:         &Counter{},
		WalRotations:       &Counter{},
		WalReplays:         &Counter{},
		WalSegmentsOpen:    &Gauge{},
		WalkStarts:         &Counter{},
		WalkCancels:        &Counter{},
		WalkPaths:          &Counter{},
		WalkVisitedMarks:   &Gauge{},
		ShardRotations:     &Counter{},
		IndexRebuilds:      &Counter{},
		IndexRebuildErrors: &Counter{},
		SnapshotCaptures:   &Counter{},
		HTTPRequests:       &Counter{},
	}
}

// Snapshot returns a flat map of metric name to value.
func (r *Registry) Snapshot() map[string]int64 {
	return map[string]int64{
		"edge_writes":          r.EdgeWrites.Value(),
		"edge_deletes":         r.EdgeDeletes.Value(),
		"vertex_writes":        r.VertexWrites.Value(),
		"vertex_deletes":       r.VertexDeletes.Value(),
		"wal_appends":          r.WalAppends.Value(),
		"wal_rotations":        r.WalRotations.Value(),
		"wal_replays":          r.WalReplays.Value(),
		"wal_segments_open":    r.WalSegmentsOpen.Value(),
		"walk_starts":          r.WalkStarts.Value(),
		"walk_cancels":         r.WalkCancels.Value(),
		"walk_paths":           r.WalkPaths.Value(),
		"walk_visited_marks":   r.WalkVisitedMarks.Value(),
		"shard_rotations":      r.ShardRotations.Value(),
		"index_rebuilds":       r.IndexRebuilds.Value(),
		"index_rebuild_errors": r.IndexRebuildErrors.Value(),
		"snapshot_captures":    r.SnapshotCaptures.Value(),
		"http_requests":        r.HTTPRequests.Value(),
	}
}
