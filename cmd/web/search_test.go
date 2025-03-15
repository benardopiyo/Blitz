package main

import (
	"os"
	"testing"
)

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
