package ingest

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event is a change the village cares about. Kind is "spawn", "activity" or
// "gone".
type Event struct {
	Kind      string
	SessionID string
	Slug      string
	ToolName  string
	Role      string
	At        time.Time
}

type WatcherConfig struct {
	DataDirs    []string
	Interval    time.Duration
	BurstWindow time.Duration
}

type watched struct {
	dataDir string
	tail    *Tailer
}

// Watcher turns session files and transcript growth into events. One goroutine
// owns all state, so no lock is needed. fsnotify makes a write visible at once;
// the poll is the fallback when a notification is missed.
type Watcher struct {
	cfg      WatcherConfig
	events   chan Event
	notify   *fsnotify.Watcher
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	sessions map[string]*watched
	pending  map[string]Event
	expiry   map[string]time.Time
	burst    *time.Timer
	watched  map[string]bool
}

func NewWatcher(cfg WatcherConfig) *Watcher {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.BurstWindow <= 0 {
		cfg.BurstWindow = 250 * time.Millisecond
	}
	return &Watcher{
		cfg:      cfg,
		events:   make(chan Event, 64),
		done:     make(chan struct{}),
		sessions: map[string]*watched{},
		pending:  map[string]Event{},
		expiry:   map[string]time.Time{},
		watched:  map[string]bool{},
	}
}

// Events is closed when the watcher stops.
func (w *Watcher) Events() <-chan Event { return w.events }

func (w *Watcher) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)
	if nw, err := fsnotify.NewWatcher(); err == nil {
		w.notify = nw
	}
	go w.run()
}

// Stop ends event delivery and waits for the watcher goroutine. It is safe to
// call before Start and more than once.
func (w *Watcher) Stop() {
	if w.cancel == nil {
		return
	}
	w.cancel()
	<-w.done
}

func (w *Watcher) run() {
	defer close(w.done)
	defer close(w.events)
	if w.notify != nil {
		defer w.notify.Close()
	}
	defer func() {
		if w.burst != nil {
			w.burst.Stop()
		}
		for _, s := range w.sessions {
			if s.tail != nil {
				s.tail.Close()
			}
		}
	}()

	var notifyEvents <-chan fsnotify.Event
	var notifyErrors <-chan error
	if w.notify != nil {
		notifyEvents, notifyErrors = w.notify.Events, w.notify.Errors
	}

	w.scan()
	w.readAll()
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		var burstC <-chan time.Time
		if w.burst != nil {
			burstC = w.burst.C
		}
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.scan()
			w.readAll()
		case <-burstC:
			w.flush()
		case _, ok := <-notifyEvents:
			if !ok {
				notifyEvents = nil
				continue
			}
			w.readAll()
		case <-notifyErrors:
		}
	}
}

func (w *Watcher) scan() {
	live := make(map[string]*watched, len(w.sessions))
	for _, dir := range w.cfg.DataDirs {
		sessions, err := ScanSessions(dir)
		if err != nil {
			// A transient read error must not remove live villagers.
			for id, sess := range w.sessions {
				if sess.dataDir == dir {
					live[id] = sess
				}
			}
			continue
		}
		for _, s := range sessions {
			if _, ok := live[s.SessionID]; ok {
				continue
			}
			sess, ok := w.sessions[s.SessionID]
			if !ok {
				sess = &watched{dataDir: dir, tail: w.openTranscript(s.SessionID, dir)}
				w.sessions[s.SessionID] = sess
				if !w.emit(Event{Kind: "spawn", SessionID: s.SessionID, Slug: s.Slug, At: time.Now()}) {
					return
				}
			}
			live[s.SessionID] = sess
		}
		w.watchProjects(dir)
	}
	for id, sess := range w.sessions {
		if _, ok := live[id]; ok {
			continue
		}
		if sess.tail != nil {
			sess.tail.Close()
		}
		delete(w.sessions, id)
		delete(w.expiry, id)
		if ev, ok := w.pending[id]; ok {
			delete(w.pending, id)
			if !w.emit(ev) {
				return
			}
		}
		if !w.emit(Event{Kind: "gone", SessionID: id, At: time.Now()}) {
			return
		}
	}
}

func (w *Watcher) openTranscript(id, dataDir string) *Tailer {
	for _, p := range matches(filepath.Join(dataDir, "projects", "*", id+".jsonl")) {
		if t, err := NewTailer(p, func(l Line) { w.record(id, l) }); err == nil {
			return t
		}
	}
	return nil
}

func (w *Watcher) readAll() {
	for id, sess := range w.sessions {
		if sess.tail == nil {
			if sess.tail = w.openTranscript(id, sess.dataDir); sess.tail == nil {
				continue
			}
		}
		if _, err := sess.tail.Read(); err != nil {
			sess.tail.Close()
			sess.tail = nil
		}
	}
}

func (w *Watcher) record(id string, l Line) {
	w.pending[id] = Event{Kind: "activity", SessionID: id, ToolName: l.ToolName, Role: l.Role, At: time.Now()}
	if _, ok := w.expiry[id]; !ok {
		w.expiry[id] = time.Now().Add(w.cfg.BurstWindow)
		w.arm()
	}
}

// arm points the single timer at the earliest per-session expiry.
func (w *Watcher) arm() {
	var next time.Time
	for _, at := range w.expiry {
		if next.IsZero() || at.Before(next) {
			next = at
		}
	}
	if w.burst != nil {
		w.burst.Stop()
		w.burst = nil
	}
	if !next.IsZero() {
		w.burst = time.NewTimer(time.Until(next))
	}
}

// flush emits one activity per session whose own burst window has closed.
func (w *Watcher) flush() {
	now := time.Now()
	for id, at := range w.expiry {
		if at.After(now) {
			continue
		}
		delete(w.expiry, id)
		if ev, ok := w.pending[id]; ok {
			delete(w.pending, id)
			if !w.emit(ev) {
				return
			}
		}
	}
	w.arm()
}

func (w *Watcher) emit(ev Event) bool {
	select {
	case w.events <- ev:
		return true
	case <-w.ctx.Done():
		return false
	}
}

// watchProjects watches the transcript root and its project folders, since
// fsnotify is not recursive and a folder can appear after startup.
func (w *Watcher) watchProjects(dataDir string) {
	root := filepath.Join(dataDir, "projects")
	w.addWatch(root)
	for _, sub := range matches(filepath.Join(root, "*")) {
		w.addWatch(sub)
	}
}

func (w *Watcher) addWatch(path string) {
	if w.notify == nil || w.watched[path] {
		return
	}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		return
	}
	if err := w.notify.Add(path); err == nil {
		w.watched[path] = true
	}
}

func matches(pattern string) []string {
	got, _ := filepath.Glob(pattern)
	return got
}
