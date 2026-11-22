# Bot Village Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A live village where every running Claude Code session is a villager that walks, works, idles and sleeps as its transcript changes.

**Architecture:** One Go binary serves an embedded 480×270 Canvas 2D scene over WebSocket. Ingest scans Claude Code's `sessions/` for liveness and tails `projects/*/*.jsonl` from EOF, emitting coalesced activity events; a small simulation turns those into villager behaviour.

**Tech Stack:** Go 1.24+ (stdlib + fsnotify + gorilla/websocket), vanilla JS/CSS/HTML, Canvas 2D, WebSocket.

**Spec:** `design.md`

## Global Constraints

- Go 1.24+, standard library plus fsnotify and `gorilla/websocket`. No other Go dependencies.
- Vanilla JS/CSS/HTML served by `//go:embed static`. No bundler, no framework, no build step.
- Internal resolution 480×270, integer nearest-neighbour upscale only. Type drawn at 1×, never sharpened, letter case preserved.
- 60fps with ~30 villagers.
- No audio files, no font files, no third-party sprite sheets. All sound synthesized in-browser.
- Never write into a Claude Code data directory.
- No metrics in the UI: no line counts, token counts, byte sizes, transcript snippets, or real tool names in bubbles.
- Listen on `0.0.0.0:8040`.
- Stage, `html` and `body` are `overflow: hidden`; overlays never move the map; focus uses `{ preventScroll: true }`.
- A quiet villager is never hidden. It sleeps in the open with a `Zzz`.

## Review Focus

1. A transcript is being appended to while read — a partial trailing line must not parse and must not advance the offset.
2. A transcript is replaced or truncated — the offset must reset rather than silently skipping new content.
3. A session file outlives its process — a villager must leave, not stand there forever.
4. Two sessions share a `cwd` — names must stay distinct and stable across restarts.
5. A prompt to a session with no reachable socket — the walk completes, the status says unreachable, nothing claims success.

---

### Task 1: Scaffold and embedded static serving

**Files:**
- Create: `village/go.mod`, `village/main.go`
- Create: `village/server/config.go`, `village/server/mux.go`
- Create: `village/server/mux_test.go`
- Create: `village/static/index.html`, `village/static/style.css`

**Interfaces:**
- Consumes: nothing.
- Produces: `server.Config{DataDirs []string; Listen string; Demo bool}`; `server.NewMux(cfg Config) *http.ServeMux`.
- Produces: `static.FS` (`embed.FS`) rooted at `static/`; `main` flags `--data` (repeatable, defaults to `~/.claude`), `--listen` (default `:8040`), `--demo`.

- [ ] **Step 1: Write the failing test**

```go
func TestRootServesEmbeddedIndex(t *testing.T) {
	srv := httptest.NewServer(NewMux(Config{Listen: ":8040"}))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("got %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !bytes.Contains(body, []byte("Bot Village")) {
		t.Fatalf("index missing title")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./...`
Expected: FAIL — `NewMux` undefined.

- [ ] **Step 3: Implement `NewMux`**

Serve `static.FS` under `/`. `index.html` contains the title `Bot Village`, a `<canvas id="stage" width="480" height="270">`, and the `M` mute hint. `style.css` letterboxes the stage on a dark backdrop and sets `overflow: hidden` on `html, body`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./...`
Expected: PASS. Then `go run .` and `GET http://localhost:8040/` → 200.

- [ ] **Step 5: Commit**

```bash
git add village/
git commit -m "feat: scaffold Go server with embedded static"
```

---

### Task 2: Prompt transport probe

**Files:**
- Create: `village/server/prompt/client.go`, `village/server/prompt/probe.go`
- Create: `village/server/prompt/client_test.go`
- Create: `village/PROBE.md`

**Interfaces:**
- Consumes: nothing.
- Produces: `prompt.Transport` interface `{ Send(ctx context.Context, sessionID, text string) error }`.
- Produces: `prompt.ErrUnreachable`, `prompt.NewSocketTransport(socketPath string) Transport`.
- Produces: `prompt.Probe(socketPath string) error` — the one-shot manual check.

`Send` returns `ErrUnreachable` when the socket is missing, refuses, or times out after 2s. Blank text is rejected with a non-`ErrUnreachable` error.

- [ ] **Step 1: Write the failing test**

```go
func TestSendMissingSocketIsUnreachable(t *testing.T) {
	tr := NewSocketTransport(filepath.Join(t.TempDir(), "absent.sock"))
	err := tr.Send(context.Background(), "abc", "hello")
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("want ErrUnreachable, got %v", err)
	}
}

func TestSendBlankTextIsRejected(t *testing.T) {
	tr := NewSocketTransport("")
	if err := tr.Send(context.Background(), "abc", "   "); err == nil {
		t.Fatal("want rejection")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./server/prompt/`
Expected: FAIL — `NewSocketTransport` undefined.

- [ ] **Step 3: Implement the transport**

`Send` opens the socket under a 2s `context.WithTimeout`, writes the message, and maps every failure mode to `ErrUnreachable`. Record in `PROBE.md` what this machine's `messagingSocketPath` actually does when spoken to — this is the design's one unverified claim.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./server/prompt/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add village/server/prompt village/PROBE.md
git commit -m "feat: session prompt transport with unreachable reporting"
```

---

### Task 3: Session liveness scan

**Files:**
- Create: `village/ingest/sessions.go`, `village/ingest/sessions_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `ingest.LiveSession{PID int; SessionID, CWD, Slug, SocketPath, Status string; UpdatedAt time.Time}`.
- Produces: `ingest.ScanSessions(dataDir string) ([]LiveSession, error)`.

Reads `<dataDir>/.home/.claude/sessions/*.json`, skipping unparseable files and returning empty (no error) when the directory is absent.

- [ ] **Step 1: Write the failing test**

```go
func TestScanReadsSessionsAndSkipsGarbage(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ".home/.claude/sessions/14676.json"),
		`{"pid":14676,"sessionId":"d91c","cwd":"E:\\Projects\\DawnMerge","status":"busy","updatedAt":1795357608275,"messagingSocketPath":"\\\\.\\pipe\\x"}`)
	write(t, filepath.Join(dir, ".home/.claude/sessions/9.json"), `{oops`)
	got, err := ScanSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SessionID != "d91c" || got[0].Status != "busy" {
		t.Fatalf("%+v", got)
	}
	if got[0].Slug == "" {
		t.Fatal("slug not derived")
	}
}

func TestScanMissingDirIsEmptyNotError(t *testing.T) {
	got, err := ScanSessions(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./ingest/`
Expected: FAIL — `ScanSessions` undefined.

- [ ] **Step 3: Implement `ScanSessions`**

Slug is the final element of `cwd` with non-alphanumerics collapsed and lowercased (`DawnMerge` → `dawnmerge`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./ingest/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add village/ingest/sessions.go village/ingest/sessions_test.go
git commit -m "feat: scan live Claude Code sessions"
```

---

### Task 4: JSONL tailer

**Files:**
- Create: `village/ingest/tail.go`, `village/ingest/tail_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `ingest.Line{Kind, ToolName, Role string}` where `Kind` is `message`, `tool`, or `unknown`.
- Produces: `ingest.NewTailer(path string, onLine func(Line)) (*Tailer, error)`; `(*Tailer).Read() (int, error)` returning new bytes consumed.
- Produces: `ingest.StartOffset(path string) (int64, error)` — current size, so tailing begins at EOF.

Invariants: a line without a trailing newline is never emitted and never advances the offset; when file size drops below the offset, the offset resets to zero.

- [ ] **Step 1: Write the failing test**

```go
func TestPartialTrailingLineIsNotEmitted(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(p, []byte(`{"type":"user"}`+"\n"+`{"type":"assis`), 0o644)
	var lines []Line
	tr, _ := NewTailer(p, func(l Line) { lines = append(lines, l) })
	tr.Read()
	if len(lines) != 1 {
		t.Fatalf("got %d lines", len(lines))
	}
}

func TestCompletingTheLineEmitsItOnce(t *testing.T) {
	// append `tant"}` + newline, Read once: exactly 2 lines total, second Kind == "message"
}

func TestTruncationResetsOffset(t *testing.T) {
	// read 3 lines, truncate the file to 10 bytes, Read: offset resets, no panic
}

func TestUnknownJSONIsKindUnknown(t *testing.T) {
	// `{"nope":1}\n` yields one Line with Kind == "unknown", no error
}

func TestLongLineIsTruncatedNotFatal(t *testing.T) {
	// a 2MB single line yields one Line, no error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./ingest/ -run TestPartialTrailingLine`
Expected: FAIL — `NewTailer` undefined.

- [ ] **Step 3: Implement the tailer**

Buffered reads on a held `*os.File`, one `bufio.Scanner` per chunk with a 1MB max line, `Kind` derived from the JSON `type` field, offset reset when size < offset.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./ingest/`
Expected: PASS, all five tests.

- [ ] **Step 5: Commit**

```bash
git add village/ingest/tail.go village/ingest/tail_test.go
git commit -m "feat: incremental JSONL tailer with partial-line safety"
```

---

### Task 5: Burst coalescing into activity events

**Files:**
- Create: `village/ingest/watcher.go`, `village/ingest/watcher_test.go`

**Interfaces:**
- Consumes: `Tailer`, `Line`, `ScanSessions`.
- Produces: `ingest.Event{Kind, SessionID, ToolName, Role string; At time.Time}` where `Kind` ∈ `spawn|activity|gone`.
- Produces: `ingest.WatcherConfig{DataDirs []string; Interval time.Duration; BurstWindow time.Duration}` — defaults 1s poll, 250ms burst.
- Produces: `ingest.NewWatcher(cfg WatcherConfig) *Watcher`; `(*Watcher).Events() <-chan Event`; `(*Watcher).Start(ctx)`; `(*Watcher).Stop()`.

- [ ] **Step 1: Write the failing test**

```go
func TestBurstInsideWindowEmitsOneActivity(t *testing.T) {
	// write 5 lines within 50ms, burst window 250ms: exactly 1 Event{Kind:"activity"}
}

func TestSpawnThenGoneAcrossScans(t *testing.T) {
	// sessions/ has one file: spawn emitted once, not on every scan
}

func TestGoneEmitsForDeletedSession(t *testing.T) {
	// delete the session file: exactly one Event{Kind:"gone"} for that id
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./ingest/ -run TestBurst`
Expected: FAIL — `NewWatcher` undefined.

- [ ] **Step 3: Implement the watcher**

An fsnotify watch on each transcript directory triggers an immediate read; a 1s mtime/size poll runs alongside it so a missed event never stalls the village. A per-session 250ms timer flushes at most one `activity` carrying the last line's tool and role.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./ingest/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add village/ingest/watcher.go village/ingest/watcher_test.go
git commit -m "feat: coalesce transcript bursts into activity events"
```

---

### Task 6: Villager identity

**Files:**
- Create: `village/roster/name.go`, `village/roster/village.go`, `village/roster/avatar.go`
- Create: `village/roster/name_test.go`, `village/roster/village_test.go`, `village/roster/avatar_test.go`

**Interfaces:**
- Consumes: `ingest.LiveSession`.
- Produces: `roster.Villager{SessionID, Name, Title, Slug string; Seed uint64}`.
- Produces: `roster.Build(sessions []ingest.LiveSession, titles map[string]string) []roster.Villager`.
- Produces: `roster.LoadTitles(path string) (map[string]string, error)`.
- Produces: `roster.AvatarPNG(sessionID string) []byte` — deterministic 32×32 PNG.

- [ ] **Step 1: Write the failing test**

```go
func TestSameSlugGetsDistinctStableNames(t *testing.T) {
	// two sessions, same cwd: names "dawnmerge" and "dawnmerge-<4 hex>", identical on repeat calls
}

func TestAvatarIsDeterministic(t *testing.T) {
	if !bytes.Equal(AvatarPNG("abc"), AvatarPNG("abc")) {
		t.Fatal("not stable")
	}
	if bytes.Equal(AvatarPNG("abc"), AvatarPNG("abd")) {
		t.Fatal("collides")
	}
}

func TestTitleFromVillageJSON(t *testing.T) {
	// {"sessions":{"abc":{"title":"Architect"}}} gives Title == "Architect"
}

func TestMalformedVillageJSONIsIgnored(t *testing.T) {
	// returns an empty map and nil error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./roster/`
Expected: FAIL — `Build` undefined.

- [ ] **Step 3: Implement naming, titles and avatars**

Names: slug, or `slug-<first 4 hex of session id>` on collision. Avatars: a fixed-palette critter face derived from FNV-1a of the session id, encoded with `image/png`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./roster/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add village/roster/
git commit -m "feat: derive villager names, titles and avatars"
```

---

### Task 7: Roster API and WebSocket hub

**Files:**
- Create: `village/server/state.go`, `village/server/ws.go`, `village/server/api.go`
- Create: `village/server/state_test.go`, `village/server/ws_test.go`, `village/server/api_test.go`

**Interfaces:**
- Consumes: `Watcher.Events()`, `roster.Build`, `roster.AvatarPNG`.
- Produces: `server.State` with `Apply(ingest.Event)` and `Snapshot() []roster.Villager`.
- Produces: routes `GET /api/villagers`, `GET /avatars/{sessionID}`, `GET /ws`.
- Produces: WS frames `{"type":"roster","villagers":[…]}` and `{"type":"activity","sessionId":…,"tool":…}`.

- [ ] **Step 1: Write the failing test**

```go
func TestVillagersEndpointReflectsSpawnAndGone(t *testing.T) {
	// spawn: 1 villager with name and seed; gone: 0
}

func TestAvatarRouteIsDeterministicPNG(t *testing.T) {
	// two GETs return equal bytes, Content-Type image/png
}

func TestWSReceivesRosterThenActivity(t *testing.T) {
	// dial: first frame type == "roster"; after spawn+activity, a frame type == "activity"
}

func TestVillagersPayloadHasNoMetrics(t *testing.T) {
	// marshal the snapshot; assert the JSON contains none of "lines", "bytes", "tokens", "snippet"
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./server/`
Expected: FAIL — `State` undefined.

- [ ] **Step 3: Implement the hub and routes**

Per-client buffered send channel; a slow client is dropped rather than blocking the hub. Import `github.com/gorilla/websocket`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./server/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add village/server/
git commit -m "feat: roster API and websocket event hub"
```

---

### Task 8: Renderer skeleton and town ground

**Files:**
- Create: `village/static/art.js`, `village/static/town.js`, `village/static/app.js`
- Create: `village/static/smoke.html`
- Modify: `village/static/index.html`

**Interfaces:**
- Consumes: `index.html`'s `<canvas id="stage">`.
- Produces: `art.PALETTE` (named 24-colour ramp); `art.dither(ctx, x, y, w, h, c1, c2, ratio)`.
- Produces: `town.create(canvas) → { draw(t), W: 480, H: 270 }`, with `town.bandY(i) int` for i ∈ {0 back, 1 mid, 2 front}.

- [ ] **Step 1: Write the failing check**

In `smoke.html`, assert `town.bandY(0) < town.bandY(1) && town.bandY(1) < town.bandY(2)` and that the canvas is 480×270.

- [ ] **Step 2: Verify it fails**

Open `village/static/smoke.html`; console shows the assertion failure.

- [ ] **Step 3: Implement the stage and ground**

Integer scale `floor(min(w/480, h/270))`, nearest-neighbour via `image-rendering: pixelated`, dithered sky, three banded ground planes.

- [ ] **Step 4: Verify it passes**

Smoke check passes; ground is sharp at 1× and 3×.

- [ ] **Step 5: Commit**

```bash
git add village/static/
git commit -m "feat: 480x270 canvas stage with banded town ground"
```

---

### Task 9: Villagers on the map

**Files:**
- Create: `village/static/villagers.js`
- Modify: `village/static/town.js`, `village/static/app.js`

**Interfaces:**
- Consumes: `/ws` frames; `town.bandY`.
- Produces: `villagers.applyRoster(list)`, `villagers.onActivity(ev)`, `villagers.pick(x, y) → SessionID|null`, `villagers.draw(ctx, hour)`.
- Villager shape: `{id, name, seed, band, x, y, state, targetX, targetY}` with `state` ∈ `idle|walking|working|asleep`.

- [ ] **Step 1: Write the failing check**

Smoke asserts that spawning two villagers yields two draw calls, and that `pick()` inside a villager's box returns that id and `null` outside.

- [ ] **Step 2: Verify it fails**

Smoke console shows the assertion failure.

- [ ] **Step 3: Implement**

Spawn walk-on, depth sort by `y`, billboard sprites, bold 7px monospace nametag on a rounded pill, walk to a work spot on `activity`.

- [ ] **Step 4: Verify it passes**

Smoke passes; 30 villagers hold 60fps in the devtools FPS meter.

- [ ] **Step 5: Commit**

```bash
git add village/static/
git commit -m "feat: villagers spawn, walk and draw in depth order"
```

---

### Task 10: Behaviour states

**Files:**
- Create: `village/static/behaviour.js`
- Modify: `village/static/villagers.js`

**Interfaces:**
- Consumes: `villagers` state objects.
- Produces: `behaviour.step(v, now, rng)`; `behaviour.overlayFor(tool) → "hammer"|"wrench"|"spark"`; idle activities cycling `sit|fish|lean|kick`.

Invariant: after 90s with no activity the villager is `asleep` and its position is unchanged.

- [ ] **Step 1: Write the failing check**

Smoke asserts: `activity` → `working`; 90s quiet → `asleep` with identical position; three known tools map to their overlays and an unknown tool maps to `spark`.

- [ ] **Step 2: Verify it fails**

Smoke console shows the assertion failure.

- [ ] **Step 3: Implement**

The state machine, work overlay sprites, and the `Zzz` tell.

- [ ] **Step 4: Verify it passes**

Smoke passes.

- [ ] **Step 5: Commit**

```bash
git add village/static/
git commit -m "feat: villager behaviour, work overlays and sleep tell"
```

---

### Task 11: Day/night and audio

**Files:**
- Create: `village/static/audio.js`
- Modify: `village/static/town.js`, `village/static/app.js`

**Interfaces:**
- Produces: `town.hour() → number` (0–23, from `?hour=` else local time).
- Produces: `audio.unlock()`, `audio.setMuted(bool)`, `audio.isMuted()`, `audio.blip(kind)`.

Invariant: `AudioContext` is created only after the first user gesture; `M` toggles mute.

- [ ] **Step 1: Write the failing check**

Smoke asserts `hour()` honours `?hour=22`, the palette at 22 differs from at 12, and `setMuted(true)` prevents a blip from being scheduled.

- [ ] **Step 2: Verify it fails**

Smoke console shows the assertion failure.

- [ ] **Step 3: Implement**

Night tint by palette swap (not alpha), blips synthesized with oscillator plus envelope.

- [ ] **Step 4: Verify it passes**

Smoke passes; `?hour=22` is visibly darker; no audio before a click.

- [ ] **Step 5: Commit**

```bash
git add village/static/
git commit -m "feat: local-time day/night and synthesized audio"
```

---

### Task 12: Villager card

**Files:**
- Create: `village/static/card.js`, `village/static/card.css`
- Modify: `village/static/app.js`, `village/static/index.html`

**Interfaces:**
- Produces: `card.show(villager)`, `card.hide()`, `card.selected() → string|null`.

The card shows avatar, name, title (omitted when empty), and activity as one goal word plus a role chip.

- [ ] **Step 1: Write the failing check**

Smoke asserts the card opens on click with the right name and title, that its text matches none of `/\d+\s*(lines|bytes|tokens)/`, and that `stage.getBoundingClientRect()` is identical before and after opening.

- [ ] **Step 2: Verify it fails**

Smoke console shows the assertion failure, or the rect check fails if the card sits in flow.

- [ ] **Step 3: Implement**

The card is a `position: fixed` overlay outside the stage; its close button takes focus with `{ preventScroll: true }`.

- [ ] **Step 4: Verify it passes**

Smoke passes; opening and closing shifts nothing.

- [ ] **Step 5: Commit**

```bash
git add village/static/
git commit -m "feat: slim villager card that never moves the map"
```

---

### Task 13: Player, prompt bar and optimistic send

**Files:**
- Create: `village/static/player.js`, `village/static/composer.css`
- Modify: `village/static/app.js`, `village/server/api.go`, `village/main.go`

**Interfaces:**
- Consumes: `prompt.Transport`, `card.selected()`.
- Produces: `POST /api/prompt` with body `{"sessionId":string,"text":string}` → `200 {"status":"delivered"}`, `502 {"status":"unreachable"}`, `400` for blank text, `404` for an unknown session.
- Produces: `player.send(sessionID, text)` — walks to the target, wakes them, shows a bubble clipped to 22 characters, status `on my way` → `said it` or `couldn't reach them`.

- [ ] **Step 1: Write the failing test**

```go
func TestPromptRouteReportsUnreachable(t *testing.T) {
	// POST for a known session with no socket: 502, body status == "unreachable"
}

func TestPromptRouteRejectsBlankText(t *testing.T) {
	// "   " gives 400
}

func TestPromptRouteUnknownSession(t *testing.T) {
	// unknown id gives 404
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./server/ -run TestPromptRoute`
Expected: FAIL — route returns 404 for everything.

- [ ] **Step 3: Implement the route and the client-side walk**

The input is disabled with placeholder `click a villager first…` until a selection exists, then `message <name>…`. Enter sends.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./...`
Expected: PASS. Smoke asserts the walk completes and the status reads `couldn't reach them` when the route returns 502.

- [ ] **Step 5: Commit**

```bash
git add village/
git commit -m "feat: player walks to a villager and delivers a prompt"
```

---

### Task 14: Demo mode

**Files:**
- Create: `village/demo/demo.go`, `village/demo/demo_test.go`
- Modify: `village/main.go`

**Interfaces:**
- Consumes: `ingest.ScanSessions`.
- Produces: `demo.Seed(dir string, n int) (func(), error)` — creates `n` sessions and transcripts in `dir`, appending lines every 2–6s; the returned func stops it.

- [ ] **Step 1: Write the failing test**

```go
func TestSeedCreatesSessionsAndAppends(t *testing.T) {
	dir := t.TempDir()
	stop, err := Seed(dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	s, err := ingest.ScanSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 {
		t.Fatalf("got %d", len(s))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd village && go test ./demo/`
Expected: FAIL — `Seed` undefined.

- [ ] **Step 3: Implement demo wiring**

`--demo` uses only the temp dir, prints the URL, and states that delivery is faked.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd village && go test ./demo/`
Expected: PASS, then `go run . --demo` shows a living village.

- [ ] **Step 5: Commit**

```bash
git add village/demo/ village/main.go
git commit -m "feat: demo mode with twelve fake villagers"
```

---

### Task 15: README

**Files:**
- Create: `village/README.md`

**Interfaces:** none.

- [ ] **Step 1: Write it**

Demo, live mode (`--data` pointed at `~/.claude` and at a DawnMerge project home), `0.0.0.0:8040`, Tailscale access, the `village.json` format, `?hour=`, `M`, and an explicit line that demo delivery is faked and that live delivery depends on the session socket.

- [ ] **Step 2: Verify**

Every command in the README runs as written from a clean checkout.

- [ ] **Step 3: Commit**

```bash
git add village/README.md
git commit -m "docs: village README"
```
