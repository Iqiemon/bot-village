package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type LiveSession struct {
	PID        int
	SessionID  string
	CWD        string
	Slug       string
	SocketPath string
	Status     string
	UpdatedAt  time.Time
}

type sessionFile struct {
	PID             int    `json:"pid"`
	SessionID       string `json:"sessionId"`
	CWD             string `json:"cwd"`
	Status          string `json:"status"`
	UpdatedAtMillis int64  `json:"updatedAt"`
	SocketPath      string `json:"messagingSocketPath"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// ScanSessions reads <dataDir>/sessions/*.json, skipping files it cannot parse.
// A missing sessions directory means no sessions, not an error.
func ScanSessions(dataDir string) ([]LiveSession, error) {
	entries, err := os.ReadDir(filepath.Join(dataDir, "sessions"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []LiveSession
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dataDir, "sessions", e.Name()))
		if err != nil {
			continue
		}
		var s sessionFile
		if json.Unmarshal(b, &s) != nil || s.SessionID == "" {
			continue
		}
		out = append(out, LiveSession{
			PID:        s.PID,
			SessionID:  s.SessionID,
			CWD:        s.CWD,
			Slug:       slugify(s.CWD),
			SocketPath: s.SocketPath,
			Status:     s.Status,
			UpdatedAt:  time.UnixMilli(s.UpdatedAtMillis),
		})
	}
	return out, nil
}

func slugify(cwd string) string {
	return strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(filepath.Base(cwd)), "-"), "-")
}
