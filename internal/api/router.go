// Package api sets up the Gorilla Mux HTTP router for Bruce.
package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "bruce/docs"
	"bruce/internal/ai"
	"bruce/internal/api/handlers"
	"bruce/internal/config"
	"bruce/web"
)

// NewRouter creates and returns the application HTTP router.
// startTime is used to calculate uptime for the /health endpoint.
// asynqmonHandler is the Asynqmon dashboard handler mounted at /monitor.
// registry is the LLM provider registry for the /api/v1/providers endpoint.
func NewRouter(startTime time.Time, asynqmonHandler http.Handler, registry *ai.ProviderRegistry, cfg *config.Config) *mux.Router {
	r := mux.NewRouter()

	// Apply middleware stack (innermost to outermost).
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// Health check (no /api/v1 prefix — used by Docker healthcheck).
	r.HandleFunc("/health", handlers.HealthHandler(startTime)).Methods(http.MethodGet)

	// API routes under /api/v1.
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/config", handlers.ConfigHandler()).Methods(http.MethodGet, http.MethodPut)
	api.HandleFunc("/providers", handlers.ProvidersHandler(registry)).Methods(http.MethodGet)
	api.HandleFunc("/sessions", handlers.SessionsHandler()).Methods(http.MethodGet)
	api.HandleFunc("/sessions/{id}", handlers.SessionsHandler()).Methods(http.MethodGet, http.MethodPatch)
	api.HandleFunc("/sessions/{id}/messages", handlers.MessagesHandler()).Methods(http.MethodGet)
	api.HandleFunc("/connectors", handlers.ConnectorsHandler()).Methods(http.MethodGet)

	// Chat routes (web chat interface — synchronous LLM calls).
	chatHandler := handlers.ChatHandler(registry, cfg)
	api.HandleFunc("/chat/sessions", chatHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/chat/sessions/{id}", chatHandler).Methods("GET", "DELETE", "OPTIONS")
	api.HandleFunc("/chat/sessions/{id}/messages", chatHandler).Methods("GET", "POST", "OPTIONS")

	// Asynqmon dashboard — must be before the SPA catch-all.
	r.PathPrefix("/monitor").Handler(asynqmonHandler)

	// Swagger UI.
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Static assets — embedded at compile time from web/public/.
	// Any path not matched above falls through here.
	// Unknown paths (e.g. /some/spa/route) return index.html so the JS router handles them.
	staticFS, err := fs.Sub(web.Public, "public")
	if err != nil {
		panic("web/public embed misconfigured: " + err.Error())
	}
	r.PathPrefix("/").HandlerFunc(spaHandler(staticFS))

	return r
}

// recoveryMiddleware catches panics and returns HTTP 500.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"internal server error","code":500}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code written.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before delegating to the wrapped writer.
func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// loggingMiddleware logs HTTP request details (method, path, status, duration).
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start).Milliseconds()
		log.Printf("%s %s %d %dms", r.Method, r.URL.Path, wrapped.status, duration)
	})
}

// corsMiddleware adds CORS headers to allow all origins (safe for private LAN).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

		// Handle OPTIONS preflight requests.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// spaHandler serves files from the embedded FS.
// If the requested file does not exist, it falls back to index.html
// so that client-side routing works correctly.
func spaHandler(static fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(static))
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the leading slash to get the relative path inside the FS.
		path := r.URL.Path
		if path == "/" || path == "" {
			// Explicitly serve index.html for the root.
			http.ServeFileFS(w, r, static, "index.html")
			return
		}

		// Try to open the requested file.
		f, err := static.Open(path[1:]) // strip leading "/"
		if err != nil {
			// File not found → serve index.html for SPA client-side routing.
			http.ServeFileFS(w, r, static, "index.html")
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	}
}
