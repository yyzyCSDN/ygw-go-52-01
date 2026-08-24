package model

import (
	"sort"
	"testing"
)

func sortedKeys(attrs map[string]string) []string {
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestAttrSetMergeAndKeys(t *testing.T) {
	attrs := AttrSet{}.Set("region", "east").Set("tier", "1")
	attrs.Merge(map[string]string{"owner": "team-a"})
	keys := sortedKeys(attrs)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %v", keys)
	}
	if keys[0] != "owner" || keys[1] != "region" || keys[2] != "tier" {
		t.Fatalf("keys not sorted: %v", keys)
	}
}

func TestAttrEqual(t *testing.T) {
	left := map[string]string{"a": "1", "b": "2"}
	right := map[string]string{"b": "2", "a": "1"}
	if !equalMaps(left, right) {
		t.Fatalf("attribute maps should be equal")
	}
	right["c"] = "3"
	if equalMaps(left, right) {
		t.Fatalf("attribute maps should differ after adding a key")
	}
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || other != value {
			return false
		}
	}
	return true
}
