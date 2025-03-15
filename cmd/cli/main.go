package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Argumnets represents the command-line arguments for the text indexing application.
// It stores all configurable parameters passed via flags.
type Argumnets struct {
	command string
	// command specifies the operation to perform
	// Valid values: "index" (create index) or "lookup" (search index)

	inputFile string
	// inputFile is the path to the input file
	// For "index": path to the source text file to be indexed
	// For "lookup": path to the previously generated index file

	chunkSize string
	// chunkSize defines the size of text chunks in bytes
	// Stored as string from command line, converted to int during processing
	// Default value: "4096" (4KB)

	outputFile string
	// outputFile is the path for the index file
	// For "index": destination path where the generated index will be saved
	// Not used in "lookup" command

	queryHash string
	// queryHash is the SimHash value to search for
	// Used in "lookup" command only
	// Expected to be a hexadecimal string representation
}

// main is the entry point of the text indexing application.
// It parses command-line flags and executes either an indexing or lookup operation
// based on the provided command.
func main() {
	// Define and initialize command-line arguments structure
	var args Argumnets

	// Define command-line flags
	flag.StringVar(&args.command, "c", "", "Command (index or lookup)")
	// -c: Specifies the operation to perform (either "index" or "lookup")

	flag.StringVar(&args.inputFile, "i", "", "Input file or index file path")
	// -i: Path to input text file (for indexing) or index file (for lookup)

	flag.StringVar(&args.chunkSize, "s", "4096", "Size of each chunk in bytes (default: 4096 bytes).")
	// -s: Size of text chunks in bytes (defaults to 4096 if not specified)

	flag.StringVar(&args.outputFile, "o", "", "Path to save the generated index file|Path to the previously generated index file.")
	// -o: Output path for index file (used in index command)

	flag.StringVar(&args.queryHash, "h", "", "The SimHash value of the chunk to search for")
	// -h: SimHash value to search for (used in lookup command)

	// Parse all defined flags from command line
	flag.Parse()

	var err error

	// Validate and convert chunk size from string to integer
	chunkSize, err := strconv.Atoi(args.chunkSize)
	if err != nil {
		fmt.Println("invalid chunk size. Provide a valid chunk size (e.g.,1024).")
		return
	}

	// Process the specified command
	switch args.command {
	case "index":
		// Execute indexing operation
		// Creates an index file from the input text file using specified chunk size
		err = indexCommand(args.inputFile, chunkSize, args.outputFile)

	case "lookup":
		// Convert query hash from hexadecimal string to uint64
		numHash, errr := strconv.ParseUint(args.queryHash, 16, 64)
		if errr != nil {
			fmt.Println("Error: Invalid SimHash value")
			fmt.Println("Ensure the file was indexed before looking up.")
			return
		}
		// Execute lookup operation using the provided hash
		err = lookupCommand(args.inputFile, numHash)

	default:
		// Display usage information if invalid or no command is provided
		fmt.Println("TextIndex - Fast & Scalable Text Indexer")
		fmt.Println("\nUsage:")
		fmt.Println("  Index:  textindex -c index -i <input_file.txt> -s <chunk_size> -o <index_file.idx>")
		fmt.Println("  Lookup: textindex -c lookup -i <index_file.idx> -h <simhash_value>")
		fmt.Println("\nExamples:")
		fmt.Println("  textindex -c index -i jungle_book_by_kipling.txt -s 512 -o jungle_book.index")
		fmt.Println("  textindex -c lookup -i jungle_book.index -h 8u9ryi3rujoef")
		return
	}

	// Handle any errors from command execution
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
