package main

// ChunkInfo holds information about a text chunk
// It represents metadata for a single chunk of text from the indexed file
type ChunkInfo struct {
	Offset int64
	// Offset represents the starting position of the chunk in the original file
	// Measured in bytes from the beginning of the file
	// Type int64 allows for large file support

	Size int
	// Size is the length of the chunk in bytes
	// Typically matches the configured chunk size, except possibly for the last chunk
	// Type int is sufficient as individual chunks are unlikely to exceed 2GB

	Hash uint64
	// Hash is the SimHash value calculated for this chunk
	// Stored as uint64 to accommodate 64-bit hash values
	// Used for quick comparison and lookup operations
}

// Index represents the in-memory index of chunks
// It maintains a complete index structure for a text file
type Index struct {
	FilePath string
	// FilePath is the path to the original text file that was indexed
	// Useful for reference and potential file operations

	ChunkSize int
	// ChunkSize is the configured size of each chunk in bytes
	// Stored as int since it's provided by the user and validated
	// Typically matches the -s flag value (default 4096)

	Chunks []ChunkInfo
	// Chunks is a slice containing all chunk metadata
	// Each element describes one chunk of the original file
	// Ordered by position in the file

	HashToChunks map[uint64][]int
	// HashToChunks maps SimHash values to slice of chunk indices
	// Key: Hash value of a chunk
	// Value: Slice of indices into the Chunks array
	// Enables fast lookup of chunks by their hash value
	// Multiple chunks may share the same hash (hence the slice)
}
