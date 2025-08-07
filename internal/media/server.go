package media

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"fmt"
	"log"

	"github.com/gorilla/mux"
	"gosocial/internal/dbmongo"
)

type HTTPServer struct {
	storage *dbmongo.MediaStorage
}

func NewHTTPServer(mongoClient *dbmongo.MongoClient) *HTTPServer {
	return &HTTPServer{
		storage: dbmongo.NewMediaStorage(mongoClient),
	}
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	// Main endpoint: GET /media/{fileId}
	router.HandleFunc("/media/{fileId}", s.serveFile).Methods("GET")

	// Health check
	router.HandleFunc("/health", s.health).Methods("GET")

	router.ServeHTTP(w, r)
}

func (s *HTTPServer) serveFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileId := vars["fileId"]

	// Use your existing DownloadFile method
	fileReader, mediaFile, err := s.storage.DownloadFile(r.Context(), fileId)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set content type based on file extension
	contentType := s.getContentType(mediaFile.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", mediaFile.Size))

	// Stream file directly to response
	_, err = io.Copy(w, fileReader)
	if err != nil {
		log.Printf("Error streaming file: %v", err)
	}
}

func (s *HTTPServer) getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

func (s *HTTPServer) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Media server is healthy"))
}

