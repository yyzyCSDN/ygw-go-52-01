package model

import "testing"

func TestPathAppendAndLast(t *testing.T) {
	path := NewPath("a")
	path = path.Append("b", "e1")
	path = path.Append("c", "e2")
	if len(path.Vertices) != 3 {
		t.Fatalf("expected 3 vertices, got %d", len(path.Vertices))
	}
	if path.Vertices[2] != "c" {
		t.Fatalf("expected last c, got %s", path.Vertices[2])
	}
	if len(path.Edges) != 2 || path.Edges[1] != "e2" {
		t.Fatalf("unexpected edges: %v", path.Edges)
	}
	if path.Depth != 2 {
		t.Fatalf("expected depth 2, got %d", path.Depth)
	}
}

func TestVertexAttrFallback(t *testing.T) {
	vertex := NewVertex("v1", "")
	vertex.Attrs["region"] = "west"
	if vertex.Attrs["region"] != "west" {
		t.Fatalf("expected west, got %s", vertex.Attrs["region"])
	}
}
