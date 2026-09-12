package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

//go:embed web/index.html
var webFS embed.FS

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		log.Printf("warning: invalid bool for %s=%q, using default %v", key, v, fallback)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("warning: invalid int for %s=%q, using default %d", key, v, fallback)
	}
	return fallback
}

func main() {
	listenAddr := getEnv("PORT", ":8080")
	downloadDir := getEnv("HOST_DOWNLOAD_DIR", "./downloads")
	dataDir := getEnv("HOST_DATA_DIR", "./data")
	format := getEnv("SPOTDL_FORMAT", "mp3")
	extraArgs := getEnv("SPOTDL_ARGS", "")
	binary := getEnv("SPOTDL_BINARY", "spotdl")
	concurrency := getEnvInt("CONCURRENCY", 2)
	maxHistory := getEnvInt("MAX_HISTORY", 200)

	mbEnabled := getEnvBool("MB_LOOKUP_ENABLED", true)
	mbUserAgent := getEnv("MB_USER_AGENT", "spotjspf/1.0")
	mbRateLimitMS := getEnvInt("MB_RATE_LIMIT_MS", 1000)

	var statePath string
	if dataDir != "" {
		statePath = filepath.Join(dataDir, "downloads.json")
	}

	var mb *MusicBrainzClient
	if mbEnabled {
		mb = NewMusicBrainzClient(mbUserAgent, time.Duration(mbRateLimitMS)*time.Millisecond)
		if mbUserAgent == "spotjspf/1.0" {
			log.Println("note: using a placeholder MB_USER_AGENT; MusicBrainz asks that this identify your actual app/contact — set MB_USER_AGENT to override")
		}
	} else {
		log.Println("MusicBrainz lookup disabled (MB_LOOKUP_ENABLED=false); tracks with only a MusicBrainz ID will fall back to text search")
	}

	store := NewStore(maxHistory, statePath)
	downloader := NewDownloader(store, downloadDir, format, extraArgs, binary, concurrency, 1000, mb)
	srv := &Server{store: store, downloader: downloader}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", srv.handleUpload)
	mux.HandleFunc("/api/downloads", srv.handleDownloads)
	mux.HandleFunc("/healthz", srv.handleHealth)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := webFS.ReadFile("web/index.html")
		if err != nil {
			log.Printf(err.Error())
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("spotjspf listening on %s (downloads -> %s, concurrency=%d, format=%s)",
			listenAddr, downloadDir, concurrency, format)
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
