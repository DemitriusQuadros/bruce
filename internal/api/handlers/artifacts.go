package handlers

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"bruce/internal/config"
	"bruce/internal/logging"
)

// ArtifactItem represents artifact metadata for API consumers.
type ArtifactItem struct {
	Filename    string    `json:"filename"`
	URL         string    `json:"url"`
	AbsoluteURL string    `json:"absolute_url,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ArtifactsFileServer serves static files from the artifacts directory.
func ArtifactsFileServer(artifactsDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(r.URL.Path)
		if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		fullPath := filepath.Join(artifactsDir, filepath.FromSlash(cleanPath))

		stat, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if stat.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Detect and set MIME type
		ext := filepath.Ext(fullPath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			switch strings.ToLower(ext) {
			case ".html", ".htm":
				contentType = "text/html; charset=utf-8"
			case ".css":
				contentType = "text/css; charset=utf-8"
			case ".js":
				contentType = "application/javascript"
			case ".json":
				contentType = "application/json"
			case ".md":
				contentType = "text/markdown; charset=utf-8"
			case ".svg":
				contentType = "image/svg+xml"
			default:
				contentType = "text/plain; charset=utf-8"
			}
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFile(w, r, fullPath)
	})
}

// ListArtifactsHandler returns all artifacts stored in artifactsDir as JSON.
func ListArtifactsHandler(artifactsDir string, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		baseURL := ""
		if cfg != nil {
			if cfg.Tools.Artifacts.BaseURL != "" {
				baseURL = strings.TrimRight(cfg.Tools.Artifacts.BaseURL, "/")
			} else if cfg.App.BaseURL != "" {
				baseURL = strings.TrimRight(cfg.App.BaseURL, "/")
			}
		}

		var items []ArtifactItem

		err := filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(info.Name(), ".tmp") {
				return nil
			}

			relPath, err := filepath.Rel(artifactsDir, path)
			if err != nil {
				return nil
			}
			slashPath := filepath.ToSlash(relPath)
			relURL := "/artifacts/" + slashPath
			var absURL string
			if baseURL != "" {
				absURL = baseURL + relURL
			}

			items = append(items, ArtifactItem{
				Filename:    slashPath,
				URL:         relURL,
				AbsoluteURL: absURL,
				SizeBytes:   info.Size(),
				UpdatedAt:   info.ModTime().UTC(),
			})
			return nil
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan artifacts")
			return
		}

		if items == nil {
			items = []ArtifactItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}
}

// DeleteArtifactHandler deletes a specific artifact by filename.
func DeleteArtifactHandler(artifactsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := vars["filename"]

		cleanFilename := filepath.Clean(filename)
		if strings.HasPrefix(cleanFilename, "..") || strings.Contains(cleanFilename, "/../") {
			writeError(w, http.StatusBadRequest, "invalid filename")
			return
		}

		targetPath := filepath.Join(artifactsDir, cleanFilename)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}

		if err := os.Remove(targetPath); err != nil {
			logging.Errorf("failed to delete artifact %s: %v", cleanFilename, err)
			writeError(w, http.StatusInternalServerError, "failed to delete artifact")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
