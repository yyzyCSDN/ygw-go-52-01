package main

import (
	"fmt"

	"graphstore/internal/model"
	"graphstore/internal/store"
)

// seedDemo populates a small topology that exercises every write path:
// vertices, labelled entry points, hash-routed edges, WAL records and the
// attribute index.
func seedDemo(st *store.Store) error {
	vertices := []*model.Vertex{
		model.NewVertex("v-station-a", "station"),
		model.NewVertex("v-station-b", "station"),
		model.NewVertex("v-relay-1", "relay"),
		model.NewVertex("v-relay-2", "relay"),
		model.NewVertex("v-sink", "sink"),
	}
	vertices[0].Attrs["region"] = "east"
	vertices[1].Attrs["region"] = "west"
	for _, vertex := range vertices {
		if _, ok := st.GetVertex(vertex.ID); ok {
			continue
		}
		if err := st.PutVertex(vertex); err != nil {
			return fmt.Errorf("seed vertex %s: %w", vertex.ID, err)
		}
	}
	edges := []*model.Edge{
		model.NewEdge("e-a-r1", "v-station-a", "v-relay-1", "feeds").WithAttr("bandwidth", "10m"),
		model.NewEdge("e-r1-r2", "v-relay-1", "v-relay-2", "feeds").WithAttr("bandwidth", "8m"),
		model.NewEdge("e-r2-sink", "v-relay-2", "v-sink", "feeds").WithAttr("bandwidth", "4m"),
		model.NewEdge("e-b-r2", "v-station-b", "v-relay-2", "feeds").WithAttr("bandwidth", "6m"),
	}
	for _, edge := range edges {
		if _, ok := st.GetEdge(edge.ID); ok {
			continue
		}
		if err := st.PutEdge(edge); err != nil {
			return fmt.Errorf("seed edge %s: %w", edge.ID, err)
		}
	}
	// Exercise the explicit-bucket write path and the shard lifecycle: one
	// bucket is rotated and its previous generation archived.
	extra := model.NewEdge("e-a-sink", "v-station-a", "v-sink", "bypass").WithAttr("bandwidth", "2m")
	if _, ok := st.GetEdge(extra.ID); !ok {
		if err := st.PutEdgeToBucket(extra, 0); err != nil {
			return fmt.Errorf("seed explicit-bucket edge: %w", err)
		}
	}
	if _, _, err := st.RotateBucket(1); err != nil {
		return fmt.Errorf("seed rotate bucket: %w", err)
	}
	st.ArchiveBucket(1)
	return nil
}
