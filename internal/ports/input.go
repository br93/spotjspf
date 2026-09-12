package ports

import "spotjspf/internal/domain/job"

type PlaylistIngester interface {
	IngestPlaylist(data []byte) (*PlaylistIngestResult, error)
}

type PlaylistIngestResult struct {
	PlaylistTitle string
	Queued        int
	Skipped       int
	Jobs          []*job.Job
}
