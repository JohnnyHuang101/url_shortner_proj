package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	urlStore = make(map[string]string)
	mu = sync.Mutex{}
)

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	// API routes
	http.HandleFunc("/shorten", withCORS(shortenHandler))
	http.HandleFunc("/r/", redirectHandler)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}



type ShortenRequest struct {
	URL string `json:"url"`
}
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}


func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	code := generateShortCode()
	mu.Lock()
	urlStore[code] = req.URL

	var list []ShortenResponse
	for c := range urlStore {
		list = append(list, ShortenResponse{
			ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", c),
		})
	}

	mu.Unlock()

	// resp := ShortenResponse{
	// 	ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", code),
	// }
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(list)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Path[len("/r/"):]
	mu.Lock()
	target, ok := urlStore[code]
	mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For local browser fetch
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == "OPTIONS" {
			return
		}
		h(w, r)
	}
}
