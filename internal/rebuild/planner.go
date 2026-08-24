package rebuild

import (
	"sort"

	"graphstore/internal/model"
)

// Plan describes one rebuild pass: the labels to cover and the expected
// entry count per label.
type Plan struct {
	Labels     []string
	PerLabel   map[string]int
	Total      int
	IndexLimit int
}

// BuildPlan inspects the stored vertices and returns the work items of the
// upcoming rebuild.
func BuildPlan(vertices []*model.Vertex, indexLimit int) Plan {
	counts := make(map[string]int)
	for _, vertex := range vertices {
		if vertex.Label != "" {
			counts[vertex.Label]++
		}
	}
	labels := make([]string, 0, len(counts))
	total := 0
	for labelName, count := range counts {
		labels = append(labels, labelName)
		total += count
	}
	sort.Strings(labels)
	return Plan{Labels: labels, PerLabel: counts, Total: total, IndexLimit: indexLimit}
}

// Feasible reports whether every label fits within the index entry limit.
func (p Plan) Feasible() bool {
	for _, count := range p.PerLabel {
		if count > p.IndexLimit {
			return false
		}
	}
	return true
}
