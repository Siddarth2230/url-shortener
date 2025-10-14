package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/Siddarth2230/url-shortener/internal/handler"
	"github.com/Siddarth2230/url-shortener/internal/repository"
	"github.com/Siddarth2230/url-shortener/internal/service"
	"github.com/Siddarth2230/url-shortener/pkg/idgen"
)

func main() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://urlshortener:localdev123@localhost/urlshortener?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize layers
	repo := repository.NewURLRepository(db)
	generator := idgen.NewCounterGenerator(nil) // TODO: Pass Redis client
	svc := service.NewURLService(repo, generator)
	handlers := handler.NewURLHandler(svc)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenURL).Methods("POST")
	r.HandleFunc("/{shortCode}", handlers.RedirectURL).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
