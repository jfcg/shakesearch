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

	"github.com/jfcg/sorty"
)

var searcher Searcher

const (
	MaxContext = 50 // context bytes in results
	MinQuery   = 2  // minimum query length in bytes
)

func main() {
	// limit goroutines for sorty to 2
	sorty.Mxg = 2

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

	if !ok || len(qr) < MinQuery {
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

func (s *Searcher) Search(query string) (results []string) {

	// Also query lower/upper/title cases (if different)
	lower := strings.ToLower(query)
	queries := [...]string{query, lower,
		strings.ToUpper(query),
		strings.Title(lower)}

	for i := 1; i < len(queries); i++ {
		for k := 0; k < i; k++ {
			if queries[i] == queries[k] {
				queries[i] = ""
				break
			}
		}
	}

	var idxs []int
	for i := 0; i < len(queries); i++ {
		if len(queries[i]) == 0 {
			continue
		}
		idxs = append(idxs, s.SuffixArray.Lookup([]byte(queries[i]), -1)...)
	}
	// we know indices are all different since queries are all distinct

	// sort indices for sequential results and better locality
	sorty.SortI(idxs)

	buf := s.SuffixArray.Bytes()
	for _, idx := range idxs {
		a := idx - MaxContext
		if a < 0 {
			a = 0
		}
		b := idx + MaxContext
		if b > len(buf) {
			b = len(buf)
		}
		results = append(results, string(buf[a:b]))
	}
	return
}
