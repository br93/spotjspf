package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
	userAgent  string

	rateMu   sync.Mutex
	interval time.Duration
	lastCall time.Time

	cacheMu sync.Mutex
	cache   map[string]lookupResult
}

type lookupResult struct {
	spotifyURL string
	found      bool
}

type mbRecordingResponse struct {
	Relations []struct {
		Type       string `json:"type"`
		TargetType string `json:"target-type"`
		URL        struct {
			Resource string `json:"resource"`
		} `json:"url"`
	} `json:"relations"`
}

func NewClient(userAgent string, minInterval time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		userAgent:  userAgent,
		interval:   minInterval,
		cache:      make(map[string]lookupResult),
	}
}

func (c *Client) ResolveSpotifyURL(ctx context.Context, mbid string) (spotifyURL string, found bool, err error) {
	c.cacheMu.Lock()
	if r, ok := c.cache[mbid]; ok {
		c.cacheMu.Unlock()
		return r.spotifyURL, r.found, nil
	}
	c.cacheMu.Unlock()

	url, found, err := c.fetchWithRetry(ctx, mbid)
	if err != nil {
		return "", false, err
	}

	c.cacheMu.Lock()
	c.cache[mbid] = lookupResult{spotifyURL: url, found: found}
	c.cacheMu.Unlock()

	return url, found, nil
}

func (c *Client) fetchWithRetry(ctx context.Context, mbid string) (string, bool, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		c.waitForRateLimit()

		reqURL := fmt.Sprintf("https://musicbrainz.org/ws/2/recording/%s?fmt=json&inc=url-rels", mbid)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return "", false, err
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("musicbrainz rate-limited us (status %d)", resp.StatusCode)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return "", false, nil
		}

		if resp.StatusCode != http.StatusOK {
			body := make([]byte, 512)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			return "", false, fmt.Errorf("musicbrainz returned status %d: %s", resp.StatusCode, string(body[:n]))
		}

		var parsed mbRecordingResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if decodeErr != nil {
			return "", false, fmt.Errorf("could not parse musicbrainz response: %w", decodeErr)
		}

		for _, rel := range parsed.Relations {
			if rel.TargetType != "url" {
				continue
			}
			if strings.Contains(rel.URL.Resource, "open.spotify.com/track/") {
				return rel.URL.Resource, true, nil
			}
		}
		return "", false, nil
	}
	return "", false, fmt.Errorf("musicbrainz lookup failed after retries: %w", lastErr)
}

func (c *Client) waitForRateLimit() {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	if elapsed := time.Since(c.lastCall); elapsed < c.interval {
		time.Sleep(c.interval - elapsed)
	}
	c.lastCall = time.Now()
}
