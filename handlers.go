package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

const maxUploadSize = 10 << 20

type Server struct {
	store      *Store
	downloader *Downloader
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

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

	playlist, err := ParseJSPF(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playlistID := randomID()
	created := make([]*Job, 0, len(playlist.Playlist.Track))
	skipped := 0

	for _, track := range playlist.Playlist.Track {
		job := s.buildJob(playlistID, track)
		if job == nil {
			skipped++
			continue
		}
		s.store.Add(job)
		s.downloader.Enqueue(job)
		created = append(created, job)
	}

	if len(created) == 0 {
		http.Error(w, "no usable tracks found in playlist", http.StatusBadRequest)
		return
	}

	log.Printf("uploaded playlist %q: %d tracks queued, %d skipped", playlist.Playlist.Title, len(created), skipped)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"playlist_title": playlist.Playlist.Title,
		"queued":         len(created),
		"skipped":        skipped,
		"jobs":           created,
	})
}

func (s *Server) buildJob(playlistID string, track Track) *Job {
	job := &Job{
		ID:         randomID(),
		PlaylistID: playlistID,
		Name:       track.DisplayName(),
		Status:     StatusQueued,
		Message:    "waiting in queue",
		CreatedAt:  time.Now(),
	}

	if ref, ok := track.DirectSpotifyRef(); ok {
		job.Query = ref
		job.MatchSource = MatchSpotifyDirect
		return job
	}

	fallback, err := track.FallbackQuery()
	if err != nil {
		return nil
	}
	job.Query = fallback

	if mbid, ok := track.MusicBrainzRecordingID(); ok {
		job.MBRecordingID = mbid
		job.MatchSource = MatchMusicBrainzPending
		job.Message = "waiting in queue (will check MusicBrainz for a Spotify match)"
	} else {
		job.MatchSource = MatchSearch
	}

	return job
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.store.List())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
