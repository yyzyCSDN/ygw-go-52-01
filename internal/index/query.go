package index

import "sort"

// LookupByAttr returns the ids of edges carrying the attribute value.
func (ix *Index) LookupByAttr(key, value string) []string {
	values := ix.byValue[attrEntry(key, value)]
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
