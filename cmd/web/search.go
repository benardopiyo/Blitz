package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/mfonda/simhash"
)

var (
	userUploadFile    = "./cmd/web/uploads/uploaded_text.txt"
	userUploadIndexed = "./cmd/web/uploads/uploaded_text.idx"
)

func search(w http.ResponseWriter, r *http.Request) {
	loggerInfo.Println("Called one")
	if r.Method != "POST" {
		loggerErr.Println("NOt allowed")
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = r.ParseMultipartForm(40 << 57)
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, "Unable to read file content", http.StatusInternalServerError)
		return
	}

	outFile, err := os.Create(userUploadFile)
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	err = os.WriteFile(userUploadFile, fileContent, 0o644)
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, "Failed to write file to disk", http.StatusInternalServerError)
		return
	}

	userSerach := r.FormValue("searchText")

	userSerachSimHash := simhash.Simhash(simhash.NewWordFeatureSet([]byte(userSerach)))
	userChunkSize := len(userSerach)

	err = indexCommand(userUploadFile, userChunkSize, userUploadIndexed)
	if err != nil {
		loggerErr.Println(err)
		http.Error(w, "Failed to write file to disk", http.StatusInternalServerError)
		return
	}

	cont, err := lookupCommandWeb(userUploadIndexed, userSerachSimHash)
	if err != nil {
		loggerInfo.Println(cont)
		loggerErr.Println(err)
		http.Error(w, "Failed to write file to disk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cont)
}

func getChunkContent(filePath string, offset int64, size int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("error seeking in file: %w", err)
	}

	data := make([]byte, size)

	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading chunk: %w", err)
	}

	return string(data[:n]), nil
}

func lookupCommandWeb(indexFile string, queryHash uint64) (string, error) {
	if indexFile == "" {
		return "", fmt.Errorf("error: index file is required")
	}
	if queryHash == 0 {
		return "", fmt.Errorf("error: simhash value is required")
	}

	index, err := loadIndex(indexFile)
	if err != nil {
		return "", err
	}

	matchingChunks, err := lookupQuery(index, queryHash)
	if err != nil {
		return "", err
	}

	if len(matchingChunks) == 0 {
		fmt.Println("No matches found for query.")
		return "", fmt.Errorf("SimHash not found. Ensure the file was indexed before looking up")
	}

	var content []string

	for i, chunk := range matchingChunks {
		conten, err := getChunkContent(index.FilePath, chunk.Offset, chunk.Size)
		if err != nil {
			return "", err
		}

		conten += fmt.Sprintf("Query found in chunk at byte offset: %d\n", chunk.Offset)
		conten += "Chunk content:"

		if i < len(matchingChunks)-1 {
			conten += "\n---\n"
		}
		content = append(content, conten)
	}

	con := fmt.Sprintf("\n--\nQuery found %d indexed chunk(s).\n", len(matchingChunks))
	content = append(content, con)
	return strings.Join(content, " "), nil
}

func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

func lookupQuery(index *Index, queryHash uint64) ([]ChunkInfo, error) {
	matchingChunks := make([]ChunkInfo, 0)

	for _, chunkIdx := range index.HashToChunks[queryHash] {
		matchingChunks = append(matchingChunks, index.Chunks[chunkIdx])
	}

	if len(matchingChunks) == 0 {
		const maxHammingDistance = 10

		for hash, chunkIndices := range index.HashToChunks {
			distance := HammingDistance(queryHash, hash)
			if distance <= maxHammingDistance {
				for _, chunkIdx := range chunkIndices {
					matchingChunks = append(matchingChunks, index.Chunks[chunkIdx])
				}
			}
		}
	}

	return matchingChunks, nil
}

func indexCommand(inputFile string, chunkSize int, outputFile string) error {
	if inputFile == "" {
		return fmt.Errorf("error: input file is required")
	}

	if outputFile == "" {
		outputFile = filepath.Base(inputFile) + ".idx"
	}

	index, err := createIndex(inputFile, chunkSize)
	if err != nil {
		return err
	}

	err = saveIndex(index, outputFile)
	if err != nil {
		return err
	}

	return nil
}

func saveIndex(index *Index, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating index file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	if err := encoder.Encode(index); err != nil {
		return fmt.Errorf("error encoding index: %w", err)
	}

	generateHashLogs(index)

	return nil
}

func generateHashLogs(index *Index) {
	if _, err := os.Stat("hashlogs.txt"); err == nil {
		os.Remove("hashlogs.txt")
	}

	file, err := os.OpenFile("hashlogs.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	var firstTenHashes []ChunkInfo
	if len(index.Chunks) > 10 {
		firstTenHashes = index.Chunks[:9]
	} else if len(index.Chunks) < 10 {
		firstTenHashes = index.Chunks[:]
	}

	for _, line := range firstTenHashes {
		hash := fmt.Sprintf("%x", line.Hash)
		offset := strconv.Itoa(int(line.Offset))
		_, err := writer.WriteString(hash + "=>" + offset + "\n")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	err = writer.Flush()
	if err != nil {
		fmt.Println("Error flushing buffer:", err)
		return
	}
}

func createIndex(filePath string, chunkSize int) (*Index, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w. Check the file path and try again", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	fileSize := info.Size()
	estimatedChunks := int(fileSize / int64(chunkSize))
	if fileSize%int64(chunkSize) != 0 {
		estimatedChunks++
	}

	index := &Index{
		FilePath:     filePath,
		ChunkSize:    chunkSize,
		Chunks:       make([]ChunkInfo, 0, estimatedChunks),
		HashToChunks: make(map[uint64][]int),
	}

	numWorkers := runtime.NumCPU()
	jobs := make(chan struct {
		data   []byte
		offset int64
	}, numWorkers*2)
	results := make(chan ChunkInfo, numWorkers*10)
	var wg sync.WaitGroup

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

		data := make([]byte, n)
		copy(data, buffer[:n])

		jobs <- struct {
			data   []byte
			offset int64
		}{data, offset}

		offset += int64(n)
	}

	close(jobs)
	wg.Wait()
	close(results)
	resultWg.Wait()

	return index, nil
}

func loadIndex(indexPath string) (*Index, error) {
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("error opening index file: %w", err)
	}
	defer file.Close()

	var index Index

	decoder := gob.NewDecoder(file)

	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("error decoding index: %w", err)
	}

	return &index, nil
}
