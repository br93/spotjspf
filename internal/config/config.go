package config

import (
	"log"
	"os"
	"strconv"
)

type Env string

const (
	ListenAddrEnv             Env = "PORT"
	DownloadDirEnv            Env = "HOST_DOWNLOAD_DIR"
	DataDirEnv                Env = "HOST_DATA_DIR"
	SpotdlFormatEnv           Env = "SPOTDL_FORMAT"
	SpotdlArgsEnv             Env = "SPOTDL_ARGS"
	SpotdlBinaryEnv           Env = "SPOTDL_BINARY"
	ConcurrencyEnv            Env = "CONCURRENCY"
	MaxHistoryEnv             Env = "MAX_HISTORY"
	MusicBrainzEnabledEnv     Env = "MB_LOOKUP_ENABLED"
	MusicBrainzUserAgentEnv   Env = "MB_USER_AGENT"
	MusicBrainzRateLimitMSEnv Env = "MB_RATE_LIMIT_MS"
)

type Config struct {
	ListenAddr  string
	DownloadDir string
	DataDir     string

	SpotdlFormat string
	SpotdlArgs   string
	SpotdlBinary string

	Concurrency int
	MaxHistory  int

	MusicBrainzEnabled     bool
	MusicBrainzUserAgent   string
	MusicBrainzRateLimitMS int
}

func Load() Config {
	return Config{
		ListenAddr:  getEnv(ListenAddrEnv, ":8080"),
		DownloadDir: getEnv(DownloadDirEnv, "./downloads"),
		DataDir:     getEnv(DataDirEnv, "./data"),

		SpotdlFormat: getEnv(SpotdlFormatEnv, "mp3"),
		SpotdlArgs:   getEnv(SpotdlArgsEnv, ""),
		SpotdlBinary: getEnv(SpotdlBinaryEnv, "spotdl"),

		Concurrency: getEnvInt(ConcurrencyEnv, 2),
		MaxHistory:  getEnvInt(MaxHistoryEnv, 200),

		MusicBrainzEnabled:     getEnvBool(MusicBrainzEnabledEnv, true),
		MusicBrainzUserAgent:   getEnv(MusicBrainzUserAgentEnv, "spotjspf/1.0"),
		MusicBrainzRateLimitMS: getEnvInt(MusicBrainzRateLimitMSEnv, 1000),
	}
}

func getEnv(key Env, fallback string) string {
	if v := os.Getenv(string(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key Env, fallback bool) bool {
	if v := os.Getenv(string(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		log.Printf("warning: invalid bool for %s=%q, using default %v", key, v, fallback)
	}
	return fallback
}

func getEnvInt(key Env, fallback int) int {
	if v := os.Getenv(string(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("warning: invalid int for %s=%q, using default %d", key, v, fallback)
	}
	return fallback
}
