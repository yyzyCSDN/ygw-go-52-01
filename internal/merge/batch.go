package merge

import "graphstore/internal/model"

// BatchPaths slices an ordered path list into pages of the requested size.
// The pages must tile the input exactly: no path is duplicated and none is
// skipped.
func BatchPaths(paths []*model.Path, batchSize int) [][]*model.Path {
	if batchSize <= 0 {
		batchSize = 64
	}
	if len(paths) == 0 {
		return [][]*model.Path{}
	}
	batchCount := (len(paths) + batchSize - 1) / batchSize
	batches := make([][]*model.Path, 0, batchCount)
	for start := 0; start < len(paths); start += batchSize {
		end := start + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batches = append(batches, paths[start:end])
	}
	return batches
}
