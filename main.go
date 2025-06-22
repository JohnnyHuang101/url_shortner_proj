package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	// "sync"
	"database/sql"
	"log"

	"github.com/JohnnyHuang101/url-shortner/cache"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var c *cache.Cache

func main() {

	var err error
	db, err = sql.Open("sqlite3", "./urls.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hashcode TEXT NOT NULL, 
			url TEXT NOT NULL
		);
	`)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()

	// Serve static files
	r.Handle("/", http.FileServer(http.Dir("./static")))

	// API routes
	r.HandleFunc("/shorten", withCORS(shortenHandler))
	r.HandleFunc("/recents", withCORS(getRecents))
	r.HandleFunc("/r/{code}", redirectHandler)
	r.HandleFunc("/r/{code}/preview", withCORS(preview))
	r.HandleFunc("/mru", withCORS(mru))

	c = cache.NewCache()
	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func getRecents(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Only Get allowed", http.StatusMethodNotAllowed)
		return
	}

	var list []ShortenResponse

	rows, err := db.Query("SELECT hashcode FROM urls ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {

		var code string
		if err := rows.Scan(&code); err != nil {
			log.Fatal(err)
		}

		list = append(list, ShortenResponse{
			ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", code),
		})
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(list)
}

func isValidURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)

	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return true
}

type ShortenRequest struct {
	URL string `json:"url"`
}
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}
type K struct {
	K int `json:"k"`
}

// func writeJSONError(w http.ResponseWriter, msg string, code int) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(code)
// 	json.NewEncoder(w).Encode(map[string]string{"error": msg})
// }

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

	if real := isValidURL(req.URL); !real {
		// log.Fatal("womp womp")
		http.Error(w, "The URL you provided is not valid", http.StatusBadRequest)
		return
	}

	var exist bool
	var code string

	for count := 0; count < 100; count++ {

		code = generateShortCode()
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE hashcode = ?)", code).Scan(&exist)

		if err != nil {
			log.Fatal(err)
		}

		if !exist {
			break
		}
	}

	if exist {
		log.Fatal("Hashing table maxed out! Unsafe to generate new short URL!")
	}

	_, err := db.Exec("INSERT INTO urls (hashcode, url) VALUES (?, ?)", code, req.URL)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Data successfully entered into DB")
	var list []ShortenResponse

	rows, err := db.Query("SELECT hashcode FROM urls ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {

		var code string
		if err := rows.Scan(&code); err != nil {
			log.Fatal(err)
		}

		list = append(list, ShortenResponse{
			ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", code),
		})
	}

	// if err = rows.Err(); err != nil {
	// 	log.Fatal(err)
	// }

	// for c := range urlStore {
	// 	list = append(list, ShortenResponse{
	// 		ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", c),
	// 	})
	// }

	// resp := ShortenResponse{
	// 	ShortURL: fmt.Sprintf("http://localhost:8080/r/%s", code),
	// }

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(list)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]

	if val, ok := c.Get(code); ok {
		log.Print("Cache evoked!")
		http.Redirect(w, r, val, http.StatusFound)
		return
	}

	var original string
	err := db.QueryRow("SELECT url FROM urls WHERE hashcode = ?", code).Scan(&original)

	c.Set(code, original)

	if err != nil { //could be not found or db connection refused etc.

		if err == sql.ErrNoRows {
			http.Error(w, "The original url has been lost due to database degredation! ", http.StatusBadRequest)
			return
		}
		log.Fatal(err)
	}

	http.Redirect(w, r, original, http.StatusFound)
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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

func preview(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	code := vars["code"]

	// fmt.Println(code)

	var original string
	err := db.QueryRow("SELECT url FROM urls WHERE hashcode = ?", code).Scan(&original)

	if err != nil {

		if err == sql.ErrNoRows { // couldnt find stored URL
			http.Error(w, "The original url has been lost due to database degredation!", http.StatusBadRequest)
			return
		}
		http.Error(w, "Something went wrong in the database", http.StatusBadRequest)

		log.Println("Error:", err)

	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: original,
	})
}

func mru(w http.ResponseWriter, r *http.Request) {

	var k K

	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	var list []cache.CacheEntry

	list = c.TopK(k.K)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(list)
}
