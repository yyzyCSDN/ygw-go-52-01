package label

import "sort"

// Lookup returns the entry vertex ids registered under the label.
func (ix *Index) Lookup(label string) []string {
	ids := ix.entries[label]
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

// Labels returns the sorted set of known labels.
func (ix *Index) Labels() []string {
	labels := make([]string, 0, len(ix.entries))
	for label := range ix.entries {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}
