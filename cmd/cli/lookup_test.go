package main

import (
	"os"
	"reflect"
	"testing"
)

func Test_lookupCommand(t *testing.T) {
	type args struct {
		indexFile string
		queryHash uint64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "index file not found",
			args: args{
				indexFile: "testdata/nonexistent.gob",
				queryHash: 123,
			},
			wantErr: true,
		},
		{
			name: "invalid index file",
			args: args{
				indexFile: "testdata/invalid_index.gob",
				queryHash: 123,
			},
			wantErr: true,
		},
		{
			name: "no matches found",
			args: args{
				indexFile: "testdata/valid_index.gob",
				queryHash: 456,
			},
			wantErr: true,
		},
		{
			name: "empty index file",
			args: args{
				indexFile: "testdata/empty.gob",
				queryHash: 123,
			},
			wantErr: true,
		},
		{
			name: "empty index file path",
			args: args{
				indexFile: "",
				queryHash: 123,
			},
			wantErr: true,
		},
		{
			name: "zero query hash",
			args: args{
				indexFile: "testdata/valid_index.gob",
				queryHash: 0,
			},
			wantErr: true,
		},
	}

	setupTestData(t) // Setup test data

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := lookupCommand(tt.args.indexFile, tt.args.queryHash); (err != nil) != tt.wantErr {
				t.Errorf("lookupCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		a, b   uint64
		expect int
	}{
		{0b0000, 0b0000, 0}, // No difference
		{0b0000, 0b1111, 4}, // All bits different
		{0b1010, 0b0101, 4}, // Completely inverted
		{0b1100, 0b1010, 2}, // Three bits differ
		{0b1111, 0b0111, 1}, // One bit differ
	}

	for _, tt := range tests {
		result := HammingDistance(tt.a, tt.b)
		if result != tt.expect {
			t.Errorf("HammingDistance(%b, %b) = %d; want %d", tt.a, tt.b, result, tt.expect)
		}
	}
}

func Test_lookupQuery(t *testing.T) {
	type args struct {
		index     *Index
		queryHash uint64
	}
	tests := []struct {
		name    string
		args    args
		want    []ChunkInfo
		wantErr bool
	}{
		{
			name: "exact match found",
			args: args{
				index: &Index{
					HashToChunks: map[uint64][]int{100: {0, 1}},
					Chunks:       []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}},
				},
				queryHash: 100,
			},
			want:    []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}},
			wantErr: false,
		},
		{
			name: "no exact match, fuzzy match found",
			args: args{
				index: &Index{
					HashToChunks: map[uint64][]int{110: {2}},
					Chunks:       []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}, {Offset: 210, Size: 300}},
				},
				queryHash: 112, // Hamming distance 2
			},
			want:    []ChunkInfo{{Offset: 210, Size: 300}},
			wantErr: false,
		},
		{
			name: "empty index",
			args: args{
				index:     &Index{HashToChunks: map[uint64][]int{}, Chunks: []ChunkInfo{}},
				queryHash: 100,
			},
			want:    []ChunkInfo{},
			wantErr: false,
		},
		{
			name: "fuzzy match multiple chunks",
			args: args{
				index: &Index{
					HashToChunks: map[uint64][]int{110: {2, 3}},
					Chunks:       []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}, {Offset: 210, Size: 300}, {Offset: 310, Size: 400}},
				},
				queryHash: 112, // Hamming distance 2
			},
			want:    []ChunkInfo{{Offset: 210, Size: 300}, {Offset: 310, Size: 400}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lookupQuery(tt.args.index, tt.args.queryHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("lookupQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lookupQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetChunkContent(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "testfile.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Ensure cleanup after test execution

	// Write sample content to the file
	expectedContent := "Hello, Go! This is a test file."
	if _, err := tempFile.Write([]byte(expectedContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Define test cases
	tests := []struct {
		name      string
		offset    int64
		size      int
		expect    string
		expectErr bool
	}{
		{"Valid chunk", 0, 5, "Hello", false},
		{"Partial read", 7, 3, "Go!", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := getChunkContent(tempFile.Name(), tc.offset, tc.size)
			if (err != nil) != tc.expectErr {
				t.Errorf("Unexpected error status: got %v, want error: %v", err, tc.expectErr)
			}
			if content != tc.expect {
				t.Errorf("Expected %q, but got %q", tc.expect, content)
			}
		})
	}
}

// Test_loadIndex tests the loadIndex function.
func Test_loadIndex(t *testing.T) {
	type args struct {
		indexPath string
	}
	tests := []struct {
		name    string
		args    args
		want    *Index
		wantErr bool
	}{
		{
			name:    "valid index file",
			args:    args{indexPath: "testdata/valid_index.gob"},
			want:    &Index{HashToChunks: map[uint64][]int{123: {0, 1}}, Chunks: []ChunkInfo{{Offset: 10, Size: 100}, {Offset: 110, Size: 200}}},
			wantErr: false,
		},
		{
			name:    "invalid index file",
			args:    args{indexPath: "testdata/invalid_index.gob"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "file not found",
			args:    args{indexPath: "testdata/nonexistent.gob"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty file",
			args:    args{indexPath: "testdata/empty.gob"},
			want:    nil,
			wantErr: true,
		},
	}

	// Setup test data
	setupTestData(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadIndex(tt.args.indexPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
