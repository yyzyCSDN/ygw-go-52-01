package shard

// Manager owns every shard generation for every hash bucket. Each bucket has
// one active shard that accepts writes; older generations stay readable for
// traversal until they are archived.
type Manager struct {
	buckets  int
	capacity int
	gens     [][]*Shard
	active   []int
	nextID   int
}

// NewManager creates a manager with one active shard per bucket.
func NewManager(buckets, capacity int) *Manager {
	if buckets <= 0 {
		buckets = 1
	}
	manager := &Manager{
		buckets:  buckets,
		capacity: capacity,
		gens:     make([][]*Shard, buckets),
		active:   make([]int, buckets),
	}
	for bucket := 0; bucket < buckets; bucket++ {
		shardID := manager.nextID
		manager.nextID++
		manager.gens[bucket] = []*Shard{New(shardID, capacity)}
	}
	return manager
}

// Active returns the shard that currently accepts writes for the bucket.
func (m *Manager) Active(bucket int) *Shard {
	if bucket < 0 || bucket >= m.buckets {
		return nil
	}
	return m.gens[bucket][m.active[bucket]]
}

// Rotate seals the active shard of a bucket and returns the freshly created
// replacement generation. The caller is responsible for switching the router
// so future writes land in the new generation.
func (m *Manager) Rotate(bucket int) *Shard {
	old := m.Active(bucket)
	if old != nil {
		old.Seal()
	}
	shardID := m.nextID
	m.nextID++
	next := New(shardID, m.capacity)
	m.gens[bucket] = append(m.gens[bucket], next)
	m.active[bucket] = len(m.gens[bucket]) - 1
	return next
}

// Shard looks up a shard by its global identifier.
func (m *Manager) Shard(id int) (*Shard, bool) {
	for _, generations := range m.gens {
		for _, shard := range generations {
			if shard.ID == id {
				return shard, true
			}
		}
	}
	return nil, false
}

// BucketShards returns every generation of one bucket, oldest first.
func (m *Manager) BucketShards(bucket int) []*Shard {
	if bucket < 0 || bucket >= m.buckets {
		return nil
	}
	return append([]*Shard(nil), m.gens[bucket]...)
}

// AllShards returns every shard across all buckets.
func (m *Manager) AllShards() []*Shard {
	var out []*Shard
	for _, generations := range m.gens {
		out = append(out, generations...)
	}
	return out
}

// Archive freezes every generation of a bucket except the active one.
func (m *Manager) Archive(bucket int) {
	for index, shard := range m.BucketShards(bucket) {
		if index != m.active[bucket] {
			shard.Archive()
		}
	}
}
