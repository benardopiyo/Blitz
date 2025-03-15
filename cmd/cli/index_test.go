package main

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

func TestIndexCommand(t *testing.T) {
	file := "test.txt"
	content := "This is a sample test file."
	os.WriteFile(file, []byte(content), 0644)
	defer os.Remove(file)

	outputFile := "test.idx"
	defer os.Remove(outputFile)

	if err := indexCommand(file, 10, outputFile); err != nil {
		t.Fatalf("indexCommand failed: %v", err)
	}
}

// TestGenerateHashLogs verifies that generateHashLogs correctly creates a hash log file.
func TestGenerateHashLogs(t *testing.T) {
	index := &Index{
		Chunks: []ChunkInfo{
			{Hash: 0x1a2b3c, Offset: 100}, {Hash: 0x4d5e6f, Offset: 200},
			{Hash: 0x7a8b9c, Offset: 300}, {Hash: 0xabcdef, Offset: 400},
		},
	}

	generateHashLogs(index)
	fileName := "hashlogs.txt"
	defer os.Remove(fileName)

	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("Error opening log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i, chunk := range index.Chunks {
		if i >= 10 {
			break
		}
		scanner.Scan()
		expected := fmt.Sprintf("%x=>%d", chunk.Hash, chunk.Offset)
		if scanner.Text() != expected {
			t.Errorf("Mismatch in log content: expected %q, got %q", expected, scanner.Text())
		}
	}
}

func TestCreateIndex(t *testing.T) {
	file := "test.txt"
	content := "Hello, world! This is a test file."
	os.WriteFile(file, []byte(content), 0644)
	defer os.Remove(file)

	index, err := createIndex(file, 10)
	if err != nil {
		t.Fatalf("createIndex failed: %v", err)
	}
	if len(index.Chunks) == 0 {
		t.Fatalf("Expected chunks, got %d", len(index.Chunks))
	}
}

func TestValidateChunkSize(t *testing.T) {
	// Define test cases with descriptive names and expected outcomes.
	tests := []struct {
		name      string // Description of the test case
		size      int    // Input chunk size
		expectErr bool   // Expected error outcome (true if an error is expected, false otherwise)
	}{
		{
			name:      "Valid chunk size",
			size:      1024,
			expectErr: false,
		},
		{
			name:      "Zero chunk size",
			size:      0,
			expectErr: true,
		},
		{
			name:      "Negative chunk size",
			size:      -512,
			expectErr: true,
		},
	}

	// Iterate over each test case and execute the test logic.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function under test
			err := validateChunkSize(tc.size)

			// Check if the error outcome matches expectations
			if tc.expectErr && err == nil {
				t.Errorf("Expected error for size %d, but got nil", tc.size)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Did not expect error for size %d, but got: %v", tc.size, err)
			}
		})
	}
}

func TestSaveIndex(t *testing.T) {
	index := &Index{}
	file := "test_index.idx"
	defer os.Remove(file)
	if err := saveIndex(index, file); err != nil {
		t.Fatalf("saveIndex failed: %v", err)
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist", file)
	}
}
