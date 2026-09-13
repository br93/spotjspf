package api

import (
	"embed"
	"log"
	"net/http"

	"spotjspf/internal/application/ingest"
	"spotjspf/internal/ports"
)

//go:embed web/index.html
var webFS embed.FS

type Server struct {
	ingest *ingest.Service
	repo   ports.JobRepository
}

func NewServer(ingestSvc *ingest.Service, repo ports.JobRepository) *Server {
	return &Server{ingest: ingestSvc, repo: repo}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/downloads", s.handleDownloads)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		log.Print(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
