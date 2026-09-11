package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type JSPF struct {
	Playlist struct {
		Title   string  `json:"title"`
		Creator string  `json:"creator"`
		Track   []Track `json:"track"`
	} `json:"playlist"`
}

type Track struct {
	Title      string        `json:"title"`
	Creator    string        `json:"creator"`
	Album      string        `json:"album"`
	Location   StringOrSlice `json:"location"`
	Identifier StringOrSlice `json:"identifier"`
	Annotation string        `json:"annotation"`
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single != "" {
			*s = []string{single}
		}
		return nil
	}
	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*s = multi
		return nil
	}
	return fmt.Errorf("identifier/location must be a string or array of strings")
}

func ParseJSPF(data []byte) (*JSPF, error) {
	var p JSPF
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid JSPF: %w", err)
	}
	if len(p.Playlist.Track) == 0 {
		return nil, fmt.Errorf("JSPF file contains no tracks")
	}
	return &p, nil
}

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
	return "", false
}

var mbRecordingRe = regexp.MustCompile(`musicbrainz\.org/recording/([0-9a-fA-F-]{36})`)

func extractMBRecordingID(s string) (string, bool) {
	m := mbRecordingRe.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func (t Track) FallbackQuery() (string, error) {
	creator := strings.TrimSpace(t.Creator)
	title := strings.TrimSpace(t.Title)
	if creator != "" && title != "" {
		return fmt.Sprintf("%s - %s", creator, title), nil
	}
	if title != "" {
		return title, nil
	}
	for _, loc := range t.Location {
		if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
			return loc, nil
		}
	}
	return "", fmt.Errorf("track has no usable title, creator, identifier or location")
}

func isSpotifyRef(s string) bool {
	return strings.Contains(s, "open.spotify.com") || strings.HasPrefix(s, "spotify:track:") || strings.HasPrefix(s, "spotify:album:") || strings.HasPrefix(s, "spotify:playlist:")
}

func (t Track) DisplayName() string {
	creator := strings.TrimSpace(t.Creator)
	title := strings.TrimSpace(t.Title)
	switch {
	case creator != "" && title != "":
		return creator + " - " + title
	case title != "":
		return title
	default:
		return "(untitled track)"
	}
}
