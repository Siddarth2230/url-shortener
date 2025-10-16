package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
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

	// optional: ensure DB is reachable early
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	// Initialize repository
	repo := repository.NewURLRepository(db)

	// ---------- Redis-backed Counter Generator ----------
	// Make sure you have a function idgen.NewCounterGenerator(redisClient *redis.Client) *idgen.CounterGenerator
	// that returns a type implementing idgen.Generator (i.e., has Generate() (string, error)).
	redisClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
	})

	// Try pinging Redis so we fail fast if it's down
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	defer func() {
		_ = redisClient.Close()
	}()

	// Instantiate the generator (call the constructor, don't pass the constructor itself)
	gen := idgen.NewCounterGenerator(redisClient) // must return a value that implements idgen.Generator

	// ----------------------------------------------------

	// Initialize service and handlers
	svc := service.NewURLService(repo, gen, "http://localhost:8080", 10000)
	handlers := handler.NewURLHandler(svc)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenURL).Methods("POST")
	r.HandleFunc("/{shortCode}", handlers.RedirectURL).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
