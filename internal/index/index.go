package index

import (
	"fmt"

	"graphstore/internal/model"
)

// Index is the incremental attribute inverted index. It maps attribute
// values to the edges carrying them and keeps edge adjacency so deletes can
// remove every affected entry.
type Index struct {
	byValue map[string]map[string]bool
	byEdge  map[string][]string
	edgeFrom map[string]string
	edgeTo   map[string]string
}

// New creates an empty attribute index.
func New() *Index {
	return &Index{
		byValue:  make(map[string]map[string]bool),
		byEdge:   make(map[string][]string),
		edgeFrom: make(map[string]string),
		edgeTo:   make(map[string]string),
	}
}

// Update indexes every attribute of the edge.
func (ix *Index) Update(edge *model.Edge) {
	if edge == nil {
		return
	}
	ix.edgeFrom[edge.ID] = edge.From
	ix.edgeTo[edge.ID] = edge.To
	for key, value := range edge.Attrs {
		entry := attrEntry(key, value)
		if _, ok := ix.byValue[entry]; !ok {
			ix.byValue[entry] = make(map[string]bool)
		}
		if !ix.byValue[entry][edge.ID] {
			ix.byValue[entry][edge.ID] = true
			ix.byEdge[edge.ID] = append(ix.byEdge[edge.ID], entry)
		}
	}
}

// UpdateVertex indexes the attributes of a vertex. Vertex attributes are
// tracked under a reserved namespace so deletes can clean them up.
func (ix *Index) UpdateVertex(vertex *model.Vertex) {
	if vertex == nil {
		return
	}
	for key, value := range vertex.Attrs {
		entry := attrEntry(vertexAttrPrefix+key, value)
		if _, ok := ix.byValue[entry]; !ok {
			ix.byValue[entry] = make(map[string]bool)
		}
		if !ix.byValue[entry][vertex.ID] {
			ix.byValue[entry][vertex.ID] = true
			ix.byEdge[vertex.ID] = append(ix.byEdge[vertex.ID], entry)
		}
	}
}

// RemoveEdge deletes every indexed entry of the edge. An edge that was
// indexed earlier but is missing now signals an inconsistent index and is
// reported to the caller.
func (ix *Index) RemoveEdge(edge *model.Edge) error {
	if edge == nil {
		return nil
	}
	entries := ix.byEdge[edge.ID]
	if len(entries) == 0 {
		return fmt.Errorf("%w: edge %q has no index entries", model.ErrEntryMissing, edge.ID)
	}
	for _, entry := range entries {
		if values := ix.byValue[entry]; values != nil {
			delete(values, edge.ID)
			if len(values) == 0 {
				delete(ix.byValue, entry)
			}
		}
	}
	delete(ix.byEdge, edge.ID)
	delete(ix.edgeFrom, edge.ID)
	delete(ix.edgeTo, edge.ID)
	return nil
}

// RemoveVertex deletes the vertex's own attribute entries.
func (ix *Index) RemoveVertex(vertexID string) error {
	entries := ix.byEdge[vertexID]
	for _, entry := range entries {
		if values := ix.byValue[entry]; values != nil {
			delete(values, vertexID)
			if len(values) == 0 {
				delete(ix.byValue, entry)
			}
		}
	}
	delete(ix.byEdge, vertexID)
	return nil
}

// EdgeCount returns the number of indexed edges.
func (ix *Index) EdgeCount() int {
	return len(ix.edgeFrom)
}

func attrEntry(key, value string) string {
	return key + "=" + value
}

const vertexAttrPrefix = "vertex:"
