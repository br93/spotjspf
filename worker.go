package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type Downloader struct {
	store       *Store
	queue       chan *Job
	downloadDir string
	format      string
	extraArgs   []string
	binary      string
	mb          *MusicBrainzClient // nil if MusicBrainz lookup is disabled
}

func NewDownloader(store *Store, downloadDir, format, extraArgsRaw, binary string, concurrency, queueSize int, mb *MusicBrainzClient) *Downloader {
	var extra []string
	if strings.TrimSpace(extraArgsRaw) != "" {
		extra = strings.Fields(extraArgsRaw)
	}
	d := &Downloader{
		store:       store,
		queue:       make(chan *Job, queueSize),
		downloadDir: downloadDir,
		format:      format,
		extraArgs:   extra,
		binary:      binary,
		mb:          mb,
	}
	for i := 0; i < concurrency; i++ {
		go d.worker()
	}
	return d
}

func (d *Downloader) Enqueue(j *Job) {
	d.queue <- j
}

func (d *Downloader) worker() {
	for job := range d.queue {
		d.run(job)
	}
}

func (d *Downloader) run(job *Job) {
	now := time.Now()
	d.store.Update(job.ID, func(j *Job) {
		j.Status = StatusDownloading
		j.StartedAt = &now
		j.Message = "starting..."
	})

	if job.MatchSource == MatchMusicBrainzPending {
		d.resolveMusicBrainz(job)
	}

	job = d.store.Get(job.ID)
	if job == nil {
		return
	}

	outputTemplate := d.downloadDir + "/{artists} - {title}.{output-ext}"
	args := []string{"download", job.Query, "--output", outputTemplate, "--format", d.format}
	args = append(args, d.extraArgs...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, d.binary, args...)
	cmd.Dir = d.downloadDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		d.fail(job, fmt.Sprintf("could not attach stdout: %v", err))
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		d.fail(job, fmt.Sprintf("could not start spotdl: %v", err))
		return
	}

	lastLine := ""
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lastLine = line
		d.store.Update(job.ID, func(j *Job) {
			j.Message = line
		})
	}

	waitErr := cmd.Wait()
	finished := time.Now()

	if waitErr != nil {
		msg := lastLine
		if msg == "" {
			msg = waitErr.Error()
		}
		log.Printf("job %s (%s) failed: %v", job.ID, job.Name, waitErr)
		d.store.Update(job.ID, func(j *Job) {
			j.Status = StatusFailed
			j.FinishedAt = &finished
			j.Message = msg
		})
		return
	}

	d.store.Update(job.ID, func(j *Job) {
		j.Status = StatusCompleted
		j.FinishedAt = &finished
		if lastLine != "" {
			j.Message = lastLine
		} else {
			j.Message = "done"
		}
	})
}

func (d *Downloader) resolveMusicBrainz(job *Job) {
	if d.mb == nil || job.MBRecordingID == "" {
		d.store.Update(job.ID, func(j *Job) {
			j.MatchSource = MatchSearch
		})
		return
	}

	d.store.Update(job.ID, func(j *Job) {
		j.Message = "looking up MusicBrainz -> Spotify..."
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	spotifyURL, found, err := d.mb.ResolveSpotifyURL(ctx, job.MBRecordingID)
	switch {
	case err != nil:
		log.Printf("musicbrainz lookup failed for %s (%s): %v", job.MBRecordingID, job.Name, err)
		d.store.Update(job.ID, func(j *Job) {
			j.MatchSource = MatchSearch
			j.Message = "MusicBrainz lookup failed, falling back to search"
		})
	case found:
		d.store.Update(job.ID, func(j *Job) {
			j.Query = spotifyURL
			j.MatchSource = MatchMusicBrainz
			j.Message = "matched via MusicBrainz, downloading exact track"
		})
	default:
		d.store.Update(job.ID, func(j *Job) {
			j.MatchSource = MatchSearch
			j.Message = "no Spotify link on MusicBrainz, falling back to search"
		})
	}
}

func (d *Downloader) fail(job *Job, msg string) {
	now := time.Now()
	log.Printf("job %s (%s) failed: %s", job.ID, job.Name, msg)
	d.store.Update(job.ID, func(j *Job) {
		j.Status = StatusFailed
		j.FinishedAt = &now
		j.Message = msg
	})
}
