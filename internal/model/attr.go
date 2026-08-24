package model

// AttrSet is a helper for building attribute maps in callers.
type AttrSet map[string]string

// Set stores one attribute and returns the set for chaining.
func (a AttrSet) Set(key, value string) AttrSet {
	a[key] = value
	return a
}

// Merge copies all attributes from the source into the receiver.
func (a AttrSet) Merge(source map[string]string) AttrSet {
	for key, value := range source {
		a[key] = value
	}
	return a
}
