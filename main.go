package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration (from environment variables)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	// Initialize components
	LoadTemplates()      // Load HTML templates first
	ConnectDB()          // Connect to MongoDB
	InitGrammarChecker() // Check for LanguageTool JAR

	// Ensure DB disconnect on exit
	defer DisconnectDB()

	// Setup Router
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)                 // Inject request ID
	r.Use(middleware.RealIP)                    // Use X-Forwarded-For or X-Real-IP
	r.Use(middleware.Logger)                    // Log requests
	r.Use(middleware.Recoverer)                 // Recover from panics
	r.Use(middleware.Timeout(60 * time.Second)) // Set request timeout

	// --- Static Files ---
	// Serve files from the 'static' directory
	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// --- Routes ---
	r.Get("/", handleIndex)                    // Main page
	r.Post("/notes", handleUpload)             // Upload new note
	r.Get("/notes/{id}", handleGetNoteContent) // Get note content (HTML, Markdown, Details) via query param `type`
	r.Delete("/notes/{id}", handleDeleteNote)  // Delete a note

	// --- Server Setup ---
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// --- Graceful Shutdown ---
	// Channel to listen for OS signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel to listen for server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s...", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Block until we receive a signal or an error
	select {
	case sig := <-stopChan:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)
	case err := <-errChan:
		log.Printf("Server error: %v. Shutting down...", err)
	}

	// Create a context with a timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Adjust timeout as needed
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	} else {
		log.Println("Server shutdown complete.")
	}

	// DisconnectDB is called via defer in main
}
