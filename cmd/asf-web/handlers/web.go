package handlers

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HandleWebUI serves the main dashboard HTML
func HandleWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		serveFile(w, "web/index.html", "text/html")
		return
	}

	// 404 for other paths not in static
	http.NotFound(w, r)
}

// HandleStatic serves static assets (CSS, JS, etc)
func HandleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/static/"):]
	filePath := filepath.Join("web/static", path)

	// Security: prevent directory traversal with proper path-boundary check
	baseDir := filepath.Clean("web/static")
	cleanPath := filepath.Clean(filePath)
	rel, err := filepath.Rel(baseDir, cleanPath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	serveFile(w, filePath, getContentType(filePath))
}

func serveFile(w http.ResponseWriter, filePath string, contentType string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		log.Printf("[Web] Failed to read %s: %v", filePath, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func getContentType(filePath string) string {
	switch filepath.Ext(filePath) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// EmbedFS returns an embedded file system for web assets (if using go:embed in future)
// For now, just serve from disk
func GetWebFS() fs.FS {
	return os.DirFS("web")
}
