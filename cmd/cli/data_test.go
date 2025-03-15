package main

import (
	"encoding/gob"
	"os"
	"testing"
)

// setupTestData creates test files for the tests.
func setupTestData(t *testing.T) {
	err := os.Mkdir("testdata", 0755)
	if err != nil && !os.IsExist(err) {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	// Create a valid index file
	validIndex := Index{
		HashToChunks: map[uint64][]int{123: {0, 1}},
		Chunks:       []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}},
	}
	validFile, err := os.Create("testdata/valid_index.gob")
	if err != nil {
		t.Fatalf("Failed to create valid index file: %v", err)
	}
	encoder := gob.NewEncoder(validFile)
	err = encoder.Encode(validIndex)
	if err != nil {
		t.Fatalf("Failed to encode valid index: %v", err)
	}
	validFile.Close()

	// Create an invalid index file (corrupted)
	invalidFile, err := os.Create("testdata/invalid_index.gob")
	if err != nil {
		t.Fatalf("Failed to create invalid index file: %v", err)
	}
	_, err = invalidFile.Write([]byte("corrupted data"))
	if err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}
	invalidFile.Close()

	// Create an empty file
	emptyFile, err := os.Create("testdata/empty.gob")
	if err != nil {
		t.Fatalf("Failed to create empty index file: %v", err)
	}
	emptyFile.Close()
}
