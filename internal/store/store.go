package store

import (
	"fmt"
	"path/filepath"

	"graphstore/internal/index"
	"graphstore/internal/label"
	"graphstore/internal/metric"
	"graphstore/internal/model"
	"graphstore/internal/shard"
	"graphstore/internal/wal"
)

// Options configures a new graph store.
type Options struct {
	EdgeBuckets    int
	ShardCapacity  int
	LabelCap       int
	EnableWAL      bool
	WalDir         string
}

// Store is the graph storage engine. Vertices live in a memory map; edges
// live in hash-sharded generations managed by the shard manager and routed
// by the router. Writes flow through the WAL and update the attribute index.
type Store struct {
	opts       Options
	manager    *shard.Manager
	router     *shard.Router
	vertices   map[string]*model.Vertex
	vertexSeq  []string
	wal        *wal.WAL
	labelIndex *label.Index
	attrIndex  *index.Index
	metrics    *metric.Registry
	closed     bool
	routeVersion []int
}

// New creates a store with the supplied options.
func New(opts Options) (*Store, error) {
	if opts.EdgeBuckets <= 0 {
		opts.EdgeBuckets = 4
	}
	if opts.ShardCapacity <= 0 {
		opts.ShardCapacity = 1024
	}
	if opts.LabelCap <= 0 {
		opts.LabelCap = 4096
	}
	manager := shard.NewManager(opts.EdgeBuckets, opts.ShardCapacity)
	router := shard.NewRouter(manager, opts.EdgeBuckets)
	store := &Store{
		opts:       opts,
		manager:    manager,
		router:     router,
		vertices:   make(map[string]*model.Vertex),
		labelIndex: label.New(opts.LabelCap),
		attrIndex:  index.New(),
		metrics:    metric.New(),
		routeVersion: make([]int, opts.EdgeBuckets),
	}
	if opts.EnableWAL {
		walDir := opts.WalDir
		if walDir == "" {
			walDir = filepath.Join(".", "graphstore-wal")
		}
		walLog, err := wal.Open(walDir)
		if err != nil {
			return nil, fmt.Errorf("store: open wal: %w", err)
		}
		store.wal = walLog
		if err := store.replayWAL(); err != nil {
			walLog.Close()
			return nil, fmt.Errorf("store: replay wal: %w", err)
		}
	}
	return store, nil
}

// Close shuts down the store and releases the WAL handles.
func (s *Store) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.wal != nil {
		return s.wal.Close()
	}
	return nil
}

// ActiveShard returns the write shard of a bucket.
func (s *Store) ActiveShard(bucket int) *shard.Shard {
	return s.manager.Active(bucket)
}

// Shard returns a shard by id.
func (s *Store) Shard(id int) (*shard.Shard, bool) {
	return s.manager.Shard(id)
}

// EdgeShards returns every shard generation across all buckets.
func (s *Store) EdgeShards() []*shard.Shard {
	return s.manager.AllShards()
}

// RotateBucket seals the active shard of a bucket, creates the replacement
// generation and switches the route table so subsequent writes land in the
// new shard. The old and new shard ids are returned.
func (s *Store) RotateBucket(bucket int) (int, int, error) {
	if bucket < 0 || bucket >= s.opts.EdgeBuckets {
		return 0, 0, fmt.Errorf("store: bucket %d out of range", bucket)
	}
	if s.closed {
		return 0, 0, model.ErrStoreClosed
	}
	old := s.router.ShardForBucket(bucket)
	if old == nil {
		return 0, 0, fmt.Errorf("store: bucket %d has no routed shard", bucket)
	}
	next := s.manager.Rotate(bucket)
	if err := s.switchRoute(bucket, next.ID); err != nil {
		return 0, 0, err
	}
	if s.wal != nil {
		if err := s.wal.Rotate(); err != nil {
			return 0, 0, fmt.Errorf("store: rotate wal: %w", err)
		}
	}
	s.metrics.ShardRotations.Inc()
	s.metrics.WalRotations.Inc()
	if old != nil {
		return old.ID, next.ID, nil
	}
	return 0, next.ID, nil
}

// switchRoute redirects one bucket to a shard generation and records the
// route revision. The route table and the shard manager must stay aligned so
// new writes never land in a sealed generation.
func (s *Store) switchRoute(bucket, shardID int) error {
	if _, ok := s.manager.Shard(shardID); !ok {
		return fmt.Errorf("store: cannot route bucket %d to missing shard %d", bucket, shardID)
	}
	s.router.Switch(bucket, shardID)
	s.routeVersion[bucket]++
	return nil
}

// RouteVersion returns how many times the bucket has been rerouted.
func (s *Store) RouteVersion(bucket int) int {
	if bucket < 0 || bucket >= len(s.routeVersion) {
		return 0
	}
	return s.routeVersion[bucket]
}

// ArchiveBucket freezes every non-active generation of a bucket.
func (s *Store) ArchiveBucket(bucket int) {
	s.manager.Archive(bucket)
}

// ShardOfEdge locates the shard that currently holds the edge.
func (s *Store) ShardOfEdge(edgeID string) (int, error) {
	for _, shard := range s.manager.AllShards() {
		if _, ok := shard.Get(edgeID); ok {
			return shard.ID, nil
		}
	}
	return 0, fmt.Errorf("%w: edge %q", model.ErrEdgeNotFound, edgeID)
}

// Metrics returns the metric registry of the store.
func (s *Store) Metrics() *metric.Registry {
	return s.metrics
}

// LabelIndex returns the label entry index.
func (s *Store) LabelIndex() *label.Index {
	return s.labelIndex
}

// AttrIndex returns the incremental attribute index.
func (s *Store) AttrIndex() *index.Index {
	return s.attrIndex
}

// WAL returns the write-ahead log, which may be nil when disabled.
func (s *Store) WAL() *wal.WAL {
	return s.wal
}

// EdgeBuckets returns the configured bucket count.
func (s *Store) EdgeBuckets() int {
	return s.opts.EdgeBuckets
}
