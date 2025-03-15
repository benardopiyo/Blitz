package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

type ChunkInfo struct {
	Offset int64
	Size   int
	Hash   uint64
}

type Index struct {
	FilePath     string
	ChunkSize    int
	Chunks       []ChunkInfo
	HashToChunks map[uint64][]int
}

var (
	loggerErr  = log.New(os.Stdout, "ERROR\t", log.Ltime|log.Llongfile)
	loggerInfo = log.New(os.Stdout, "INFO\t", log.Ltime)
)

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", home)
	http.HandleFunc("/search", search)

	loggerInfo.Println("Server running at http://127.0.0.1:8080/")
	http.ListenAndServe(":8080", nil)
}

func home(w http.ResponseWriter, r *http.Request) {
	temp, err := template.ParseFiles("./cmd/web/static/index.html")
	if err != nil {
		loggerErr.Println(err)
	}
	err = temp.Execute(w, nil)
	if err != nil {
		loggerErr.Println(err)
	}
}
