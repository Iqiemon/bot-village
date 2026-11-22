// Package roster derives villager identity from Claude Code sessions.
package roster

import (
	"fmt"
	"hash/fnv"
	"sort"

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

// Build names every session: the slug when it is unique, otherwise the slug
// plus a 4-hex suffix from the session id. The lowest session id keeps the
// plain slug, so names do not depend on scan order.
func Build(sessions []ingest.LiveSession, titles map[string]string) []Villager {
	bySlug := map[string][]int{}
	for i, s := range sessions {
		bySlug[s.Slug] = append(bySlug[s.Slug], i)
	}
	names := make([]string, len(sessions))
	for slug, idxs := range bySlug {
		if slug == "" {
			slug = "villager"
		}
		sort.Slice(idxs, func(a, b int) bool {
			return sessions[idxs[a]].SessionID < sessions[idxs[b]].SessionID
		})
		for n, i := range idxs {
			names[i] = slug
			if n > 0 {
				names[i] += "-" + shortID(sessions[i].SessionID)
			}
		}
	}

	out := make([]Villager, len(sessions))
	for i, s := range sessions {
		out[i] = Villager{
			SessionID: s.SessionID,
			Name:      names[i],
			Title:     titles[s.SessionID],
			Slug:      s.Slug,
			Seed:      seed(s.SessionID),
		}
	}
	return out
}

// shortID is 4 hex digits of the session id's hash; short enough for a nametag,
// wide enough that two sessions in one folder almost never share a name.
func shortID(sessionID string) string {
	h := fnv.New32a()
	h.Write([]byte(sessionID))
	return fmt.Sprintf("%08x", h.Sum32())[:4]
}

func seed(sessionID string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(sessionID))
	return h.Sum64()
}
