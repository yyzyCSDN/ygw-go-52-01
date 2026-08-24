package shard

import "github.com/cespare/xxhash/v2"

// Router maps hash buckets to the shard generation that currently accepts
// writes for that bucket. It is the single place where the store decides
// where a new edge lands.
type Router struct {
	manager *Manager
	buckets int
	route   []int
}

// NewRouter initialises every bucket to the manager's active shard.
func NewRouter(manager *Manager, buckets int) *Router {
	router := &Router{manager: manager, buckets: buckets, route: make([]int, buckets)}
	for bucket := 0; bucket < buckets; bucket++ {
		active := manager.Active(bucket)
		if active != nil {
			router.route[bucket] = active.ID
		}
	}
	return router
}

// Bucket computes the hash bucket for an edge routing key.
func (r *Router) Bucket(key string) int {
	if r.buckets <= 1 {
		return 0
	}
	return int(xxhash.Sum64String(key) % uint64(r.buckets))
}

// ShardForBucket returns the shard the route table points at for the bucket.
func (r *Router) ShardForBucket(bucket int) *Shard {
	if bucket < 0 || bucket >= r.buckets {
		return nil
	}
	shard, _ := r.manager.Shard(r.route[bucket])
	return shard
}

// ShardForKey resolves the routing key to its target shard.
func (r *Router) ShardForKey(key string) *Shard {
	return r.ShardForBucket(r.Bucket(key))
}

// Switch redirects a bucket to a different shard generation.
func (r *Router) Switch(bucket, shardID int) {
	if bucket < 0 || bucket >= r.buckets {
		return
	}
	if _, ok := r.manager.Shard(shardID); !ok {
		return
	}
	r.route[bucket] = shardID
}
