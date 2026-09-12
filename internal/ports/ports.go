package ports

import (
	"context"
	"spotjspf/internal/domain/job"
)

type JobRepository interface {
	Add(j *job.Job)
	Update(id string, fn func(*job.Job))
	Get(id string) *job.Job
	List() []*job.Job
}

type MusicBrainzResolver interface {
	ResolveSpotifyURL(ctx context.Context, mbid string) (spotifyURL string, found bool, err error)
}

type Downloader interface {
	Download(ctx context.Context, query string, onProgress func(line string)) error
}

type JobEnqueuer interface {
	Enqueue(j *job.Job)
}
