package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"spotjspf/internal/adapters/http/api"
	"spotjspf/internal/adapters/musicbrainz"
	"spotjspf/internal/adapters/spotdl"
	"spotjspf/internal/adapters/storage/jsonfile"
	"spotjspf/internal/application/download"
	"spotjspf/internal/application/ingest"
	"spotjspf/internal/config"
	"spotjspf/internal/ports"
)

func main() {
	cfg := config.Load()

	var statePath string
	if cfg.DataDir != "" {
		statePath = filepath.Join(cfg.DataDir, "downloads.json")
	}

	repo := jsonfile.NewRepository(cfg.MaxHistory, statePath)

	var resolver ports.MusicBrainzResolver

	if cfg.MusicBrainzEnabled {
		resolver = musicbrainz.NewClient(cfg.MusicBrainzUserAgent, time.Duration(cfg.MusicBrainzRateLimitMS)*time.Millisecond)
		if cfg.MusicBrainzUserAgent == "spotjspf/1.0" {
			log.Println("note: using a placeholder MB_USER_AGENT; MusicBrainz asks that this identify your actual app/contact - set MB_USER_AGENT to override")
		}
	} else {
		log.Println("MusicBrainz lookup disabled (MB_LOOKUP_ENABLED=false); tracks with only a MusicBrainz ID will fall back to text search")
	}

	runner := spotdl.NewRunner(cfg.SpotdlBinary, cfg.DownloadDir, cfg.SpotdlFormat, cfg.SpotdlArgs)

	downloadService := download.NewService(repo, resolver, runner)
	dispatcher := download.NewDispatcher(downloadService, cfg.Concurrency, 1000)
	ingestService := ingest.NewService(repo, dispatcher)

	apiServer := api.NewServer(ingestService, repo)

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("spotjspf listening on %s (downloads -> %s, concurrency=%d, format=%s)",
			cfg.ListenAddr, cfg.DownloadDir, cfg.Concurrency, cfg.SpotdlFormat)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
