package playlist

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	Spotify         string = "open.spotify.com"
	SpotifyTrack    string = "spotify:track:"
	SpotifyAlbum    string = "spotify:album:"
	SpotifyPlaylist string = "spotify:playlist:"
)

const (
	MusicBrainzRegex string = `musicbrainz\.org/recording/([0-9a-fA-F-]{36})`
	PrefixHttp       string = "http"
	PrefixHttps      string = "https"
)

const (
	Empty string = ""
)

const (
	InvalidTrack  string = "track has no usable title, creator, identifier or location"
	UntitledTrack string = "(untitled track)"
)

func (t Track) DirectSpotifyRef() (string, bool) {
	for _, id := range t.Identifier {
		if isSpotifyRef(id) {
			return id, true
		}
	}
	for _, loc := range t.Location {
		if isSpotifyRef(loc) {
			return loc, true
		}
	}
	return "", false
}

var mbRecordingRe = regexp.MustCompile(MusicBrainzRegex)

func (t Track) MusicBrainzRecordingID() (string, bool) {
	for _, id := range t.Identifier {
		if mbid, ok := extractMBRecordingID(id); ok {
			return mbid, true
		}
	}
	for _, loc := range t.Location {
		if mbid, ok := extractMBRecordingID(loc); ok {
			return mbid, true
		}
	}
	return Empty, false
}

func (t Track) FallbackQuery() (string, error) {
	creator := strings.TrimSpace(t.Creator)
	title := strings.TrimSpace(t.Title)
	if creator != Empty && title != Empty {
		return fmt.Sprintf("%s - %s", creator, title), nil
	}
	if title != Empty {
		return title, nil
	}
	for _, loc := range t.Location {
		if strings.HasPrefix(loc, PrefixHttp) || strings.HasPrefix(loc, PrefixHttps) {
			return loc, nil
		}
	}
	return Empty, fmt.Errorf(InvalidTrack)
}

func (t Track) DisplayName() string {
	creator := strings.TrimSpace(t.Creator)
	title := strings.TrimSpace(t.Title)
	switch {
	case creator != Empty && title != Empty:
		return fmt.Sprintf("%s - %s", creator, title)
	case title != Empty:
		return title
	default:
		return UntitledTrack
	}
}

func isSpotifyRef(s string) bool {
	return strings.Contains(s, Spotify) ||
		strings.HasPrefix(s, SpotifyTrack) ||
		strings.HasPrefix(s, SpotifyAlbum) ||
		strings.HasPrefix(s, SpotifyPlaylist)
}

func extractMBRecordingID(s string) (string, bool) {
	m := mbRecordingRe.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}
