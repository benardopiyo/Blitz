package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"math/bits"
	"os"
)

// lookupCommand handles the lookup command
// It searches an index file for chunks matching a given SimHash value and displays their contents
// Parameters:
//
//	indexFile: Path to the previously generated index file
//	queryHash: SimHash value to search for (as uint64)
//
// Returns:
//
//	error: nil on success, error if operation fails
func lookupCommand(indexFile string, queryHash uint64) error {
	// Validate parameters
	if indexFile == "" {
		return fmt.Errorf("error: index file is required")
		// Ensures an index file path was provided via -i flag
	}
	if queryHash == 0 {
		return fmt.Errorf("error: simhash value is required")
		// Ensures a non-zero hash value was provided via -h flag
	}

	// Load the index from file into memory
	index, err := loadIndex(indexFile)
	if err != nil {
		return err
		// Returns any error from loading the index file
	}

	// Find chunks matching the query hash
	matchingChunks, err := lookupQuery(index, queryHash)
	if err != nil {
		return err
		// Returns any error from the lookup operation
	}

	// Handle case where no matches are found
	if len(matchingChunks) == 0 {
		fmt.Println("No matches found for query.")
		return fmt.Errorf("SimHash not found. Ensure the file was indexed before looking up")
		// Returns error to indicate no matches, but still considers it a valid operation
	}

	// Display matching chunks
	for i, chunk := range matchingChunks {
		// Retrieve the actual text content for each matching chunk
		content, err := getChunkContent(index.FilePath, chunk.Offset, chunk.Size)
		if err != nil {
			return err
			// Returns any error from reading chunk content
		}

		// Print chunk information and content
		fmt.Printf("Query found in chunk at byte offset: %d\n", chunk.Offset)
		fmt.Println("Chunk content:")
		fmt.Println(content)

		// Add separator between multiple chunks (but not after the last one)
		if i < len(matchingChunks)-1 {
			fmt.Println("\n---")
		}
	}

	// Print summary
	fmt.Println("\n---")
	fmt.Printf("\nQuery found %d indexed chunk(s).\n", len(matchingChunks))
	return nil
	// Successful completion with at least one match
}

// getChunkContent retrieves the content of a chunk from the original file
// It reads a specific portion of a file based on offset and size
// Parameters:
//
//	filePath: Path to the original text file
//	offset: Starting position in bytes (int64) where the chunk begins
//	size: Number of bytes to read for this chunk
//
// Returns:
//
//	string: The content of the chunk as a string
//	error: nil on success, error if file operations fail
func getChunkContent(filePath string, offset int64, size int) (string, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
		// Wraps the original error with context
	}
	defer file.Close() // Ensure file is closed after function completes

	// Move file pointer to the chunk's starting position
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("error seeking in file: %w", err)
		// Returns error if seeking to offset fails
	}

	// Allocate buffer for reading the chunk
	data := make([]byte, size)

	// Read the specified number of bytes
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading chunk: %w", err)
		// Returns error for read failures, but allows EOF
	}
	// n contains the actual number of bytes read, which may be less than size
	// at end of file

	// Convert read bytes to string and return
	return string(data[:n]), nil
	// Uses only the actual bytes read (n) rather than full buffer
}

// lookupQuery finds chunks that might contain the query text
// It searches the index for chunks matching the query hash exactly or approximately
// Parameters:
//
//	index: Pointer to the loaded Index structure containing chunk information
//	queryHash: SimHash value (uint64) to search for
//
// Returns:
//
//	[]ChunkInfo: Slice of matching chunk metadata
//	error: nil on success (currently no error conditions defined)
func lookupQuery(index *Index, queryHash uint64) ([]ChunkInfo, error) {
	// Note: Commented code suggests original intent to compute SimHash from text
	// queryHash := simhash.Simhash(simhash.NewWordFeatureSet([]byte(queryHashs)))
	// Current implementation assumes queryHash is pre-computed

	// Initialize slice for matching chunks
	matchingChunks := make([]ChunkInfo, 0)

	// Step 1: Look for exact matches
	// Get all chunk indices that exactly match the query hash
	for _, chunkIdx := range index.HashToChunks[queryHash] {
		matchingChunks = append(matchingChunks, index.Chunks[chunkIdx])
		// Adds corresponding ChunkInfo from Chunks slice
	}

	// Step 2: If no exact matches, perform fuzzy matching
	if len(matchingChunks) == 0 {
		const maxHammingDistance = 10 // Threshold for similarity
		// Hard-coded maximum Hamming distance for approximate matches

		// Iterate through all hashes in the index
		for hash, chunkIndices := range index.HashToChunks {
			// Calculate Hamming distance between query and stored hash
			distance := HammingDistance(queryHash, hash)
			if distance <= maxHammingDistance {
				// If sufficiently similar, add all chunks with this hash
				for _, chunkIdx := range chunkIndices {
					matchingChunks = append(matchingChunks, index.Chunks[chunkIdx])
				}
			}
		}
	}

	// Return all matches (exact or fuzzy), empty slice if none found
	return matchingChunks, nil
	// No error conditions currently implemented
}

// HammingDistance calculates the bit-level distance between two hashes
// It computes the number of differing bits between two 64-bit values
// Parameters:
//
//	a: First hash value (uint64)
//	b: Second hash value (uint64)
//
// Returns:
//
//	int: Number of bits that differ between a and b (0 to 64)
func HammingDistance(a, b uint64) int {
	// XOR the two hashes to identify differing bits
	// (0 where bits match, 1 where they differ)
	// Then count the number of 1s in the result
	return bits.OnesCount64(a ^ b)
}

// loadIndex loads an index from a file
// It reads and deserializes an Index structure from a binary file
// Parameters:
//
//	indexPath: Path to the index file to load
//
// Returns:
//
//	*Index: Pointer to the loaded Index structure
//	error: nil on success, error if file operations or decoding fail
func loadIndex(indexPath string) (*Index, error) {
	// Open the index file for reading
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("error opening index file: %w", err)
		// Wraps file opening error with context
	}
	defer file.Close() // Ensure file is closed after function completes

	// Initialize empty Index struct to populate
	var index Index

	// Create decoder for binary GOB format
	decoder := gob.NewDecoder(file)

	// Decode the file contents into the Index struct
	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("error decoding index: %w", err)
		// Wraps decoding error with context
	}

	// Return pointer to loaded index
	return &index, nil
}
