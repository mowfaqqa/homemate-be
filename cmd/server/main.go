package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"homemate-backend/internal/waitlist"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store, closeStore := configureStore()
	defer closeStore()
	handler := waitlist.NewHandler(store)
	adminHandler := waitlist.NewAdminHandler(store, os.Getenv("ADMIN_PASSWORD"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("POST /api/waitlist", handler)
	mux.Handle("OPTIONS /api/waitlist", handler)
	mux.Handle("GET /api/admin/waitlist", adminHandler)
	mux.Handle("GET /api/admin/waitlist/export", adminHandler)
	mux.Handle("OPTIONS /api/admin/waitlist", adminHandler)
	mux.Handle("OPTIONS /api/admin/waitlist/export", adminHandler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           cors(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HomeMate backend listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func configureStore() (waitlist.Store, func()) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Print("DATABASE_URL is not set; using non-persistent in-memory storage")
		return waitlist.NewMemoryStore(), func() {}
	}

	store, err := waitlist.NewPostgresStore(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	log.Print("Using PostgreSQL waitlist storage")
	return store, store.Close
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := r.Header.Get("Origin")
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}

		if allowedOrigin == "*" || requestOrigin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Password")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
