package metric

import (
	"sync"
	"testing"
)

func TestRegistryCounters(t *testing.T) {
	registry := New()
	registry.EdgeWrites.Inc()
	registry.EdgeWrites.Add(4)
	registry.WalkStarts.Inc()
	registry.WalSegmentsOpen.Set(2)
	snapshot := registry.Snapshot()
	if snapshot["edge_writes"] != 5 {
		t.Fatalf("expected 5 edge writes, got %d", snapshot["edge_writes"])
	}
	if snapshot["walk_starts"] != 1 {
		t.Fatalf("expected 1 walk start, got %d", snapshot["walk_starts"])
	}
	if snapshot["wal_segments_open"] != 2 {
		t.Fatalf("expected 2 open segments, got %d", snapshot["wal_segments_open"])
	}
}

func TestCounterConcurrentInc(t *testing.T) {
	counter := &Counter{}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}
	wg.Wait()
	if counter.Value() != 32 {
		t.Fatalf("expected 32 increments, got %d", counter.Value())
	}
}
