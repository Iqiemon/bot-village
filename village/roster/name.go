// Package roster derives villager identity from Claude Code sessions.
package roster

import (
	"fmt"
	"hash/fnv"

	"botvillage/ingest"
)

// Villager is one session's presence in the village.
type Villager struct {
	SessionID string
	Name      string
	Title     string
	Slug      string
	Seed      uint64
}

// Build names each session from its own slug and session id. A name never
// depends on which other sessions are live, so it survives a sibling joining,
// leaving, or the scan arriving in a different order.
func Build(sessions []ingest.LiveSession, titles map[string]string) []Villager {
	out := make([]Villager, len(sessions))
	for i, s := range sessions {
		slug := s.Slug
		if slug == "" {
			slug = "villager"
		}
		out[i] = Villager{
			SessionID: s.SessionID,
			Name:      slug + "-" + shortID(s.SessionID),
			Title:     titles[s.SessionID],
			Slug:      s.Slug,
			Seed:      seed(s.SessionID),
		}
	}
	return out
}

func shortID(sessionID string) string {
	h := fnv.New32a()
	h.Write([]byte(sessionID))
	return fmt.Sprintf("%08x", h.Sum32())
}

func seed(sessionID string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(sessionID))
	return h.Sum64()
}
