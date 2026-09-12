package job

import (
	"crypto/rand"
	"encoding/hex"
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

const (
	MessageWaiting         string = "waiting in queue"
	MessageWaitingDetailed string = "waiting in queue (will check MusicBrainz for a Spotify match)"
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

func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type TrackInfo interface {
	DisplayName() string
	DirectSpotifyRef() (query string, ok bool)
	MusicBrainzRecordingID() (mbid string, ok bool)
	FallbackQuery() (query string, err error)
}

func New(playlistID string, track TrackInfo) (j *Job, ok bool) {
	j = &Job{
		ID:         NewID(),
		PlaylistID: playlistID,
		Name:       track.DisplayName(),
		Status:     StatusQueued,
		Message:    MessageWaiting,
		CreatedAt:  time.Now(),
	}

	if ref, ok := track.DirectSpotifyRef(); ok {
		j.Query = ref
		j.MatchSource = MatchSpotifyDirect
		return j, true
	}

	fallback, err := track.FallbackQuery()
	if err != nil {
		return nil, false
	}
	j.Query = fallback

	if mbid, ok := track.MusicBrainzRecordingID(); ok {
		j.MBRecordingID = mbid
		j.MatchSource = MatchMusicBrainzPending
		j.Message = MessageWaitingDetailed
	} else {
		j.MatchSource = MatchSearch
	}

	return j, true
}
