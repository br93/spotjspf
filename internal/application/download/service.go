package download

import (
	"context"
	"log"
	"time"

	"spotjspf/internal/domain/job"
	"spotjspf/internal/ports"
)

type Service struct {
	repo       ports.JobRepository
	resolver   ports.MusicBrainzResolver
	downloader ports.Downloader

	mbTimeout       time.Duration
	downloadTimeout time.Duration
}

const (
	MessageStarting   string = "starting..."
	MessageDone       string = "done"
	MessageLookingUp  string = "looking up MusicBrainz -> Spotify..."
	MessageLookUpFail string = "MusicBrainz lookup failed, falling back to search"
	MessageMatched    string = "matched via MusicBrainz, downloading exact track"
	MessageNoSpotify  string = "no Spotify link on MusicBrainz, falling back to search"
)

func NewService(repo ports.JobRepository, resolver ports.MusicBrainzResolver, downloader ports.Downloader) *Service {
	return &Service{
		repo:            repo,
		resolver:        resolver,
		downloader:      downloader,
		mbTimeout:       15 * time.Second,
		downloadTimeout: 20 * time.Minute,
	}
}

func (s *Service) Execute(j *job.Job) {
	now := time.Now()
	s.repo.Update(j.ID, func(current *job.Job) {
		current.Status = job.StatusDownloading
		current.StartedAt = &now
		current.Message = MessageStarting
	})

	if j.MatchSource == job.MatchMusicBrainzPending {
		s.resolveMusicBrainz(j)
	}

	current := s.repo.Get(j.ID)
	if current == nil {
		return
	}

	lastLine := ""
	onProgress := func(line string) {
		lastLine = line
		s.repo.Update(current.ID, func(updating *job.Job) {
			updating.Message = line
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.downloadTimeout)
	defer cancel()

	err := s.downloader.Download(ctx, current.Query, onProgress)
	finished := time.Now()

	if err != nil {
		msg := lastLine
		if msg == "" {
			msg = err.Error()
		}
		log.Printf("job %s (%s) failed: %v", current.ID, current.Name, err)
		s.repo.Update(current.ID, func(j *job.Job) {
			j.Status = job.StatusFailed
			j.FinishedAt = &finished
			j.Message = msg
		})
		return
	}

	s.repo.Update(current.ID, func(j *job.Job) {
		j.Status = job.StatusCompleted
		j.FinishedAt = &finished
		if lastLine != "" {
			j.Message = lastLine
		} else {
			j.Message = MessageDone
		}
	})
}

func (s *Service) resolveMusicBrainz(j *job.Job) {
	if s.resolver == nil || j.MBRecordingID == "" {
		s.repo.Update(j.ID, func(current *job.Job) {
			current.MatchSource = job.MatchSearch
		})
		return
	}

	s.repo.Update(j.ID, func(current *job.Job) {
		current.Message = MessageLookingUp
	})

	ctx, cancel := context.WithTimeout(context.Background(), s.mbTimeout)
	defer cancel()

	spotifyURL, found, err := s.resolver.ResolveSpotifyURL(ctx, j.MBRecordingID)
	switch {
	case err != nil:
		log.Printf("musicbrainz lookup failed for %s (%s): %v", j.MBRecordingID, j.Name, err)
		s.repo.Update(j.ID, func(current *job.Job) {
			current.MatchSource = job.MatchSearch
			current.Message = MessageLookUpFail
		})
	case found:
		s.repo.Update(j.ID, func(current *job.Job) {
			current.Query = spotifyURL
			current.MatchSource = job.MatchMusicBrainz
			current.Message = MessageMatched
		})
	default:
		s.repo.Update(j.ID, func(current *job.Job) {
			current.MatchSource = job.MatchSearch
			current.Message = MessageNoSpotify
		})
	}
}
