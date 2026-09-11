package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusDownloading Status = "downloading"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

type MatchSource string

const (
	MatchSpotifyDirect      MatchSource = "spotify_direct"
	MatchMusicBrainzPending MatchSource = "musicbrainz_pending"
	MatchMusicBrainz        MatchSource = "musicbrainz"
	MatchSearch             MatchSource = "search"
)

type Job struct {
	ID            string      `json:"id"`
	PlaylistID    string      `json:"playlist_id"`
	Name          string      `json:"name"`
	Query         string      `json:"query"`
	MBRecordingID string      `json:"mb_recording_id,omitempty"`
	MatchSource   MatchSource `json:"match_source"`
	Status        Status      `json:"status"`
	Message       string      `json:"message"`
	CreatedAt     time.Time   `json:"created_at"`
	StartedAt     *time.Time  `json:"started_at,omitempty"`
	FinishedAt    *time.Time  `json:"finished_at,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	jobs     []*Job
	maxItems int
	filePath string
}

func NewStore(maxItems int, filePath string) *Store {
	s := &Store{maxItems: maxItems, filePath: filePath}
	s.load()
	return s
}

func (s *Store) load() {
	if s.filePath == "" {
		return
	}
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	var jobs []*Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("warning: could not parse existing state file %s: %v", s.filePath, err)
		return
	}
	s.jobs = jobs
}

func (s *Store) persist() {
	if s.filePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		log.Printf("warning: could not create data dir: %v", err)
		return
	}
	data, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		log.Printf("warning: could not marshal state: %v", err)
		return
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("warning: could not write state file: %v", err)
		return
	}
	_ = os.Rename(tmp, s.filePath)
}

func (s *Store) Add(j *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append([]*Job{j}, s.jobs...)
	if len(s.jobs) > s.maxItems {
		s.jobs = s.jobs[:s.maxItems]
	}
	s.persist()
}

func (s *Store) Update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.ID == id {
			fn(j)
			break
		}
	}
	s.persist()
}

func (s *Store) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.jobs {
		if j.ID == id {
			cp := *j
			return &cp
		}
	}
	return nil
}

func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, len(s.jobs))
	copy(out, s.jobs)
	return out
}
