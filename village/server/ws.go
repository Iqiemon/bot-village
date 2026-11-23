package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"botvillage/ingest"
	"botvillage/roster"
)

type rosterFrame struct {
	Type      string            `json:"type"`
	Villagers []roster.Villager `json:"villagers"`
}

type activityFrame struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Tool      string `json:"tool"`
}

// Village is the roster state plus the hub that fans it out to pages.
type Village struct {
	state *State
	hub   *hub
}

// NewVillage builds a roster and loads titles from village.json beside each
// data directory.
func NewVillage(dirs []string) *Village {
	titles := map[string]string{}
	for _, dir := range dirs {
		m, err := roster.LoadTitles(filepath.Join(filepath.Dir(dir), "village.json"))
		if err != nil {
			continue
		}
		for id, title := range m {
			titles[id] = title
		}
	}
	return &Village{state: NewState(titles), hub: newHub()}
}

// Run applies watcher events until the channel closes.
func (v *Village) Run(events <-chan ingest.Event) {
	for ev := range events {
		v.Apply(ev)
	}
}

func (v *Village) Apply(ev ingest.Event) {
	v.state.Apply(ev)
	switch ev.Kind {
	case "spawn", "gone":
		v.hub.broadcast(rosterFrame{Type: "roster", Villagers: v.state.Snapshot()})
	case "activity":
		v.hub.broadcast(activityFrame{Type: "activity", SessionID: ev.SessionID, Tool: ev.ToolName})
	}
}

// serveWS sends the current roster first, then every later frame. A client
// that cannot keep up is dropped by the hub.
func (v *Village) serveWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	ch := v.hub.add()
	defer v.hub.remove(ch)

	if err := c.WriteJSON(rosterFrame{Type: "roster", Villagers: v.state.Snapshot()}); err != nil {
		return
	}
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()
	for b := range ch {
		c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			return
		}
	}
}

var upgrader = websocket.Upgrader{}

// hub holds one buffered channel per client. A client that falls a full buffer
// behind is dropped rather than allowed to stall the others.
type hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newHub() *hub { return &hub{clients: map[chan []byte]struct{}{}} }

func (h *hub) add() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *hub) remove(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

func (h *hub) broadcast(msg any) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- b:
		default:
			delete(h.clients, ch)
			close(ch)
		}
	}
}
