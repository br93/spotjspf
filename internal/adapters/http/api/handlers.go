package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const maxUploadSize = 10 << 20

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large or malformed upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' field in form-data", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "could not read upload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := s.ingest.IngestPlaylist(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("uploaded playlist %q: %d tracks queued, %d skipped", result.PlaylistTitle, result.Queued, result.Skipped)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"playlist_title": result.PlaylistTitle,
		"queued":         result.Queued,
		"skipped":        result.Skipped,
		"jobs":           result.Jobs,
	})
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.repo.List())
}
