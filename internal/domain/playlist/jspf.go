package playlist

import (
	"encoding/json"
	"fmt"
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

const (
	MessageInvalidLocation string = "identifier/location must be a string or array of strings"
	MessageInvalidJSPF     string = "invalid JSPF"
	MessageNoTracks        string = "JSPF file contains no tracks"
)

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
	return fmt.Errorf(MessageInvalidLocation)

}

func Parse(data []byte) (*JSPF, error) {
	var p JSPF
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%s: %w", MessageInvalidJSPF, err)
	}
	if len(p.Playlist.Track) == 0 {
		return nil, fmt.Errorf(MessageNoTracks)
	}
	return &p, nil
}
