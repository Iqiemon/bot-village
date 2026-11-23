package server

import (
	"sort"
	"sync"

	"botvillage/ingest"
	"botvillage/roster"
)

// State is the live roster.
type State struct {
	mu     sync.Mutex
	titles map[string]string
	byID   map[string]roster.Villager
}

func NewState(titles map[string]string) *State {
	if titles == nil {
		titles = map[string]string{}
	}
	return &State{titles: titles, byID: map[string]roster.Villager{}}
}

func (s *State) Apply(ev ingest.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch ev.Kind {
	case "spawn":
		v := roster.Build([]ingest.LiveSession{{SessionID: ev.SessionID, Slug: ev.Slug}}, s.titles)
		s.byID[ev.SessionID] = v[0]
	case "gone":
		delete(s.byID, ev.SessionID)
	}
}

// Snapshot returns villagers sorted by name so the payload is stable.
func (s *State) Snapshot() []roster.Villager {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]roster.Villager, 0, len(s.byID))
	for _, v := range s.byID {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
