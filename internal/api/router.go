// Package api sets up the Gorilla Mux HTTP router for Bruce.
package api

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "bruce/docs"
	"bruce/internal/api/handlers"
	"bruce/web"
)

// NewRouter creates and returns the application HTTP router.
// startTime is used to calculate uptime for the /health endpoint.
// asynqmonHandler is the Asynqmon dashboard handler mounted at /monitor.
func NewRouter(startTime time.Time, asynqmonHandler http.Handler) *mux.Router {
	r := mux.NewRouter()

	// API routes — registered first so the catch-all below does not intercept them.
	r.HandleFunc("/health", handlers.HealthHandler(startTime)).Methods(http.MethodGet)
	r.HandleFunc("/api/config", handlers.ConfigHandler()).Methods(http.MethodGet, http.MethodPut)
	r.HandleFunc("/api/sessions", handlers.SessionsHandler()).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/api/sessions/{id}", handlers.SessionsHandler()).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	r.HandleFunc("/api/sessions/{id}/messages", handlers.MessagesHandler()).Methods(http.MethodGet, http.MethodPost)

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
