package roster

import (
	"encoding/json"
	"os"
)

// LoadTitles reads an optional village.json. A missing or malformed file means
// no titles, never an error, so a bad file can never take the village down.
func LoadTitles(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f struct {
		Sessions map[string]struct {
			Title string `json:"title"`
		} `json:"sessions"`
	}
	if json.Unmarshal(b, &f) != nil {
		return map[string]string{}, nil
	}
	titles := make(map[string]string, len(f.Sessions))
	for id, s := range f.Sessions {
		if s.Title != "" {
			titles[id] = s.Title
		}
	}
	return titles, nil
}
