package model

import "errors"

// Sentinel errors shared across the storage, indexing and traversal layers.
var (
	ErrVertexNotFound = errors.New("vertex not found")
	ErrEdgeNotFound   = errors.New("edge not found")
	ErrShardSealed    = errors.New("shard is sealed and cannot accept writes")
	ErrShardArchived  = errors.New("shard is archived and cannot be modified")
	ErrIndexFull      = errors.New("label index entry limit reached")
	ErrSessionClosed  = errors.New("walk session is closed")
	ErrStoreClosed    = errors.New("store is closed")
	ErrEntryMissing   = errors.New("index entry is missing")
	ErrInvalidLabel   = errors.New("label is empty or too long")
	ErrDepthExceeded  = errors.New("walk depth limit exceeded")
)
