package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var searcher Searcher

func main() {
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	SuffixArray *suffixarray.Index
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query, ok := r.URL.Query()["q"]
	qr := strings.TrimSpace(query[0])

	if !ok || len(qr) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("search query too short"))
		return
	}
	results := searcher.Search(qr)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("encoding failure"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.SuffixArray = suffixarray.New(dat)
	return nil
}

func (s *Searcher) Search(query string) []string {
	idxs := s.SuffixArray.Lookup([]byte(query), -1)
	results := []string{}
	buf := s.SuffixArray.Bytes()
	for _, idx := range idxs {
		results = append(results, string(buf[idx-250:idx+250]))
	}
	return results
}
