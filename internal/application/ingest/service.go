package ingest

import (
	"errors"

	"spotjspf/internal/domain/job"
	"spotjspf/internal/domain/playlist"
	"spotjspf/internal/ports"
)

var ErrNoUsableTracks = errors.New("no usable tracks found in playlist")

type Result struct {
	PlaylistTitle string
	Queued        int
	Skipped       int
	Jobs          []*job.Job
}

type Service struct {
	repo     ports.JobRepository
	enqueuer ports.JobEnqueuer
}

func NewService(repo ports.JobRepository, enqueuer ports.JobEnqueuer) *Service {
	return &Service{repo: repo, enqueuer: enqueuer}
}

func (s *Service) IngestPlaylist(data []byte) (*Result, error) {
	parsed, err := playlist.Parse(data)
	if err != nil {
		return nil, err
	}

	playlistID := job.NewID()
	created := make([]*job.Job, 0, len(parsed.Playlist.Track))
	skipped := 0

	for _, track := range parsed.Playlist.Track {
		j, ok := job.New(playlistID, track)
		if !ok {
			skipped++
			continue
		}
		s.repo.Add(j)
		s.enqueuer.Enqueue(j)
		created = append(created, j)
	}

	if len(created) == 0 {
		return nil, ErrNoUsableTracks
	}

	return &Result{
		PlaylistTitle: parsed.Playlist.Title,
		Queued:        len(created),
		Skipped:       skipped,
		Jobs:          created,
	}, nil
}
