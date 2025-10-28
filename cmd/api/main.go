package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Siddarth2230/url-shortener/internal/handler"
	"github.com/Siddarth2230/url-shortener/internal/middleware"
	"github.com/Siddarth2230/url-shortener/internal/repository"
	"github.com/Siddarth2230/url-shortener/internal/service"
	"github.com/Siddarth2230/url-shortener/pkg/cache"
	"github.com/Siddarth2230/url-shortener/pkg/idgen"
)

func main() {
	ctx := context.Background()
	// ============================================================
	// CONNECT TO POSTGRESQL (Layer 3 - Database)
	// ============================================================
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("postgres", "postgres://urlshortener:localdev123@localhost:5433/urlshortener?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Test DB connection
	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("Database ping failed: %v", err)
	}
	defer db.Close()
	log.Println("✓ PostgreSQL connected")

	// Initialize repository
	repo := repository.NewURLRepository(db)

	// ============================================================
	// 2. CONNECT TO REDIS
	// ============================================================
	log.Println("Connecting to Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}
	log.Println("✓ Redis connected")

	// ============================================================
	// SETUP ID GENERATOR (uses Redis counter)
	// ============================================================
	gen := idgen.NewCounterGenerator(redisClient)
	log.Println("✓ ID Generator initialized (Counter-based)")

	// ============================================================
	// SETUP TWO-LAYER CACHE
	// ============================================================

	// Layer 2: Redis cache (shared across all servers)
	redisCacheTTL := 5 * time.Minute
	redisCache := cache.NewRedisCache(redisClient, redisCacheTTL)
	log.Printf("✓ Layer 2 Cache (Redis) initialized (TTL: %v)", redisCacheTTL)

	// Layer 1: LRU cache is created by default
	l1CacheSize := 10000
	log.Printf("✓ Layer 1 Cache (LRU) will be initialized (Capacity: %d)", l1CacheSize)

	// ============================================================
	// INITIALIZE SERVICE WITH TWO-LAYER CACHE
	// ============================================================
	baseURL := getEnv("BASE_URL", "http://localhost:8080")

	svc := service.NewURLServiceWithRedis(
		repo,
		gen,
		baseURL,
		l1CacheSize,
		redisCache,
	)
	log.Println("✓ URL Service initialized with TWO-LAYER caching")

	// ============================================================
	// SETUP HTTP HANDLERS
	// ============================================================
	handlers := handler.NewURLHandler(svc)

	// Setup routes
	r := mux.NewRouter()

	// Apply metrics middleware to all routes
	r.Use(middleware.MetricsMiddleware)

	// API endpoints
	r.HandleFunc("/shorten", handlers.ShortenURL).Methods("POST")
	r.HandleFunc("/{shortCode}", handlers.RedirectURL).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// ============================================================
	// START HTTP SERVER
	// ============================================================

	addr := ":8080"
	log.Printf("🚀 Server starting on %s", addr)
	log.Printf("   POST %s/shorten    - Create short URL", baseURL)
	log.Printf("   GET  %s/{code}     - Redirect to long URL", baseURL)
	log.Printf("   GET  %s/health     - Health check", baseURL)
	log.Println("Server starting on :8080")

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
