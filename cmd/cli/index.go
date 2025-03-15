package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/mfonda/simhash"
)

// indexCommand handles the index command
// It creates and saves an index from a text file using specified chunk size
// Parameters:
//
//	inputFile: Path to the text file to index
//	chunkSize: Size in bytes for each chunk
//	outputFile: Path where the index file will be saved
//
// Returns:
//
//	error: nil on success, error if operation fails
func indexCommand(inputFile string, chunkSize int, outputFile string) error {
	// Validate parameters
	if inputFile == "" {
		return fmt.Errorf("error: input file is required")
		// Ensures an input file path was provided via -i flag
	}

	// Set default output filename if not provided
	if outputFile == "" {
		outputFile = filepath.Base(inputFile) + ".idx"
		// Uses input filename with .idx extension (e.g., "text.txt" -> "text.idx")
	}

	// Validate chunk size
	err := validateChunkSize(chunkSize)
	if err != nil {
		return err
		// Returns any error from chunk size validation
	}

	// Create the index
	fmt.Printf("Indexing %s (chunk size: %d bytes)...\n", inputFile, chunkSize)
	// Inform user of indexing operation start
	index, err := createIndex(inputFile, chunkSize)
	if err != nil {
		return err
		// Returns any error from index creation
	}

	// Save the index to file
	err = saveIndex(index, outputFile)
	if err != nil {
		return err
		// Returns any error from saving the index
	}

	// Report success
	fmt.Printf("Indexed %d chunks, saved to %s\n", len(index.Chunks), outputFile)
	return nil
	// Successful completion
}

// validateChunkSize validates the chunk size parameter
// It ensures the chunk size is a positive value
// Parameters:
//
//	size: Chunk size in bytes to validate (int)
//
// Returns:
//
//	error: nil if valid, error if size is invalid
func validateChunkSize(size int) error {
	// Check if size is positive
	if size <= 0 {
		return fmt.Errorf("invalid chunk size: %d. Provide a valid chunk size (e.g.1024)", size)
		// Returns error with specific message including invalid value
	}
	return nil
	// Indicates valid chunk size
}

// saveIndex saves the index to a file
// It serializes and writes the Index structure to a binary file
// Parameters:
//
//	index: Pointer to the Index structure to save
//	outputPath: Path where the index file will be saved
//
// Returns:
//
//	error: nil on success, error if file operations or encoding fail
func saveIndex(index *Index, outputPath string) error {
	// Create or overwrite the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating index file: %w", err)
		// Wraps file creation error with context
	}
	defer file.Close() // Ensure file is closed after function completes

	// Create encoder for binary GOB format
	encoder := gob.NewEncoder(file)

	// Serialize and write the index to file
	if err := encoder.Encode(index); err != nil {
		return fmt.Errorf("error encoding index: %w", err)
		// Wraps encoding error with context
	}

	// Generate additional hash logs (side effect)
	generateHashLogs(index)
	// Note: Purpose unclear without implementation details

	return nil
	// Successful completion
}

// createIndex processes a file and creates an index
// It reads a file in chunks, computes SimHash values, and builds an Index structure
// Parameters:
//
//	filePath: Path to the text file to index
//	chunkSize: Size in bytes for each chunk
//
// Returns:
//
//	*Index: Pointer to the created Index structure
//	error: nil on success, error if file operations fail
func createIndex(filePath string, chunkSize int) (*Index, error) {
	// Open the input file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w. Check the file path and try again", err)
	}
	defer file.Close()

	// Get file size for capacity estimation
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	fileSize := info.Size()
	estimatedChunks := int(fileSize / int64(chunkSize))
	if fileSize%int64(chunkSize) != 0 {
		estimatedChunks++ // Account for partial final chunk
	}

	// Initialize Index with pre-allocated capacity
	index := &Index{
		FilePath:     filePath,
		ChunkSize:    chunkSize,
		Chunks:       make([]ChunkInfo, 0, estimatedChunks),
		HashToChunks: make(map[uint64][]int),
	}

	// Set up worker pool for parallel processing
	numWorkers := runtime.NumCPU() // Use number of CPU cores
	jobs := make(chan struct {
		data   []byte
		offset int64
	}, numWorkers*2) // Buffered channel for job queue
	results := make(chan ChunkInfo, numWorkers*10) // Buffered channel for results
	var wg sync.WaitGroup

	// Start worker goroutines to compute hashes
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				text := string(job.data)
				hash := simhash.Simhash(simhash.NewWordFeatureSet([]byte(text)))
				results <- ChunkInfo{
					Offset: job.offset,
					Size:   len(job.data),
					Hash:   hash,
				}
			}
		}()
	}

	// Process results concurrently
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for chunkInfo := range results {
			chunkIdx := len(index.Chunks)
			index.Chunks = append(index.Chunks, chunkInfo)
			index.HashToChunks[chunkInfo.Hash] = append(index.HashToChunks[chunkInfo.Hash], chunkIdx)
		}
	}()

	// Read file and dispatch chunks to workers
	buffer := make([]byte, chunkSize)
	var offset int64 = 0
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			close(jobs)
			return nil, fmt.Errorf("error reading file: %w", err)
		}

		// Copy buffer to prevent race conditions
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Send chunk to workers
		jobs <- struct {
			data   []byte
			offset int64
		}{data, offset}

		offset += int64(n)
	}

	// Cleanup and wait for completion
	close(jobs)
	wg.Wait()
	close(results)
	resultWg.Wait()

	return index, nil
}

// generateHashLogs creates a log file with hash values and offsets
// It writes up to the first 10 chunk hashes from the index to a text file
// Parameters:
//
//	index: Pointer to the Index structure containing chunk information
func generateHashLogs(index *Index) {
	// Check if hashlogs.txt exists and remove it if it does
	if _, err := os.Stat("hashlogs.txt"); err == nil {
		os.Remove("hashlogs.txt")
		// Ensures fresh log file each time
	}

	// Open/Create file with write permissions
	file, err := os.OpenFile("hashlogs.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
		// Prints error and exits if file creation fails
	}
	defer file.Close() // Ensure file closure

	// Create buffered writer for efficient writing
	writer := bufio.NewWriter(file)

	// Select up to first 10 chunks
	var firstTenHashes []ChunkInfo
	if len(index.Chunks) > 10 {
		firstTenHashes = index.Chunks[:9] // Takes first 9 (not 10, possible bug)
	} else if len(index.Chunks) < 10 {
		firstTenHashes = index.Chunks[:] // Takes all if less than 10
	}
	// Note: Condition for exactly 10 chunks is missing

	// Write hash-offset pairs to file
	for _, line := range firstTenHashes {
		hash := fmt.Sprintf("%x", line.Hash)     // Convert uint64 hash to hexadecimal
		offset := strconv.Itoa(int(line.Offset)) // Convert int64 offset to string
		_, err := writer.WriteString(hash + "=>" + offset + "\n")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
			// Prints error and exits if write fails
		}
	}

	// Flush buffer to ensure all data is written to disk
	err = writer.Flush()
	if err != nil {
		fmt.Println("Error flushing buffer:", err)
		return
		// Prints error and exits if flush fails
	}
}
