package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	// "sync"
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {

	// var err error
	db, err := sql.Open("sqlite3", "./urls.db")
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	//
	//
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			hashcode TEXT PRIMARY KEY, 
			url TEXT NOT NULL
		);
	`)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

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

	var list []ShortenResponse

	rows, err := db.Query("SELECT hashcode FROM urls LIMIT 5")
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
	code := r.URL.Path[len("/r/"):]

	var original string
	err := db.QueryRow("SELECT url FROM urls WHERE hashcode == (?)", code).Scan(&original)

	if err != nil { //could be not found or db connection refused etc.
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
