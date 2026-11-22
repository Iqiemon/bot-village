# Bot Village — design

A live little town, drawn in a cute PS1-Animal-Crossing style at 480×270 and
upscaled chunky, where every running Claude Code session is a villager you can
watch and walk up to.

Reference: [@poteto's botvillage](https://x.com/poteto/status/2093023536558555531) —
same idea, adapted from Grok Bots to Claude Code.

## Why

Watching a dozen agent sessions used to mean reading a wall of JSONL. A village
makes the same information legible at a glance: who is working, who is stuck,
who has gone quiet, and what they're working on — without exposing any of the
transcript itself. You can also say one thing to one of them without opening a
terminal.

It is a window, not a control panel. That distinction drives most of the
decisions below.

## Goals

- A villager appears for each live Claude Code session, walks and works as its
  transcript grows, and reacts when you message it.
- The whole thing runs from one Go binary with `//go:embed static`: no bundle, no
  build step, no assets to install.
- It reads Claude Code's own data directory. No new instrumentation, no daemon
  inside Claude Code, nothing written back into a session.
- It is pleasant to leave open on a second monitor for hours.

## Non-goals

- Not a transcript viewer, log search, or session manager.
- No historical replay of finished sessions.
- No database. State lives in memory and is rebuilt from the data directory.
- Not multi-user. Tailscale is the security model, or localhost.
- No editing, interrupting or killing sessions from the village.

## Anti-goals

Things that would make this a failure even if it ran:

- A dashboard with metrics. Line counts, token counts and byte sizes must never
  reach the UI.
- A villager that hides inside a building when it goes quiet. Quiet villagers
  stay on the map and fall asleep in the open.
- Real transcript text or tool names in a bot's speech bubble. Bubbles are
  generic chatter only.
- A map that shifts when the card or prompt opens. Overlays float above a fixed
  stage.

## What Claude Code gives us

Claude Code's own folder on this machine (`<data>`, `~/.claude` by default) holds:

```
<data>/sessions/<pid>.json
  { pid, sessionId, cwd, startedAt, updatedAt, status: "busy" | "idle", ... }
<data>/projects/<slug>/<sessionId>.jsonl
  append-only session transcript, typed lines (user, assistant, tool results…)
```

Sessions also carry a `messagingSocketPath` and a peer protocol. Untested.

That maps onto the reference cleanly:

| reference            | here                              |
| -------------------- | --------------------------------- |
| `agents/<uuid>/`     | `sessions/<pid>.json` (liveness)  |
| `agent-transcripts/` | `projects/<slug>/<id>.jsonl`      |
| `profile.json`       | none — derived or configured      |
| Grok Bot webhook     | session messaging socket          |

**Identity is the gap.** A session has an id, a cwd, a pid. No name, title or
avatar. The reference gets these from `profile.json`; we invent them:

- **Name** — the project folder name plus a short id suffix, always
  (`my-app-3f2a91cd`). The name depends only on the session, never on which
  siblings are live, so it survives a sibling joining or leaving.
- **Title** — optional, from a `village.json` beside the data directory.
  Unset is fine; the card simply omits the line.
- **Avatar** — generated deterministically from the session id, never random
  per run, so a villager looks the same forever.

## Decisions

**Fixed tilted camera, drawn in Canvas 2D.** The AC look comes from a ground
plane seen at a tilt, villagers as billboards sorted by depth, and a limited
dithered palette — not from a 3D engine. A hand-drawn perspective ground with
three depth bands (back, mid, front) gives houses and villagers real occlusion
at a fraction of the cost of raw WebGL2, and keeps the vanilla-JS, no-bundler
constraint intact. If the town reads flat, a WebGL2 pass is a later swap behind
the same renderer interface.

**Rendering at 480×270, integer upscale only.** Nearest-neighbour, letterboxed
on a soft backdrop. The chunky upscale is the art direction, not a compromise.
All type is drawn at 1× and never sharpened.

**Prompting: messaging socket first, honest failure second.** `/api/prompt`
attempts the session's messaging socket. If the socket is absent or refuses, the
walk still completes and the player's status line reads that they couldn't be
reached. No silent fallback to `claude --resume`: spawning a process to fake a
message is a different action than speaking to a running session, and doing it
unasked would be a lie in the UI. Demo mode fakes delivery only, and says so in
the README.

**Everything is derived, nothing is stored.** Restart the process and the
village rebuilds from disk. This is what makes it safe to run anywhere and to
kill at any time.

## Behaviour

Each villager has a home, a work spot, and a nametag.

- Transcript growth → walk to the work spot and work.
- Tool call → work overlay (hammer, wrench, spark), style-appropriate.
- "Walked up to talk" → letter or bang plus a speech bubble.
- Idle → sit, fish, lean, kick a stone.
- Long quiet → asleep in the open with a `Zzz`, still on the map.
- Click → slim card: avatar, name, title, and current activity as a short goal
  word plus a role chip. No metrics, no snippets.
- Day/night follows local time; `?hour=22` previews any hour.
- `M` mutes; audio starts on first click. All sound synthesized in-browser.

Villagers spawn when a session appears and walk off when it ends.

## The player

You appear in the same art style, nametagged `you`, idling near the centre. No
house, not in the roster, not click-to-prompt, no WASD.

A single-line prompt sits middle-bottom as a HUD overlay under the map. It is
disabled until a villager is selected — "click a villager first…" — then
"message <name>…". Enter sends.

Sending is optimistic: clear the input, show "on my way", walk to the target
chasing their position, wake them if asleep, then show a bubble with roughly a
22-character clip of what you said and the status "said it". The request is in
flight in the background; failure changes the status, never the walk.

The card and prompt are fixed overlays. They must not move the map. Stage, html
and body are `overflow: hidden`; focus uses `{ preventScroll: true }`.

## Architecture

One Go module, `village/`.

```
village/
  main.go            flags, wiring, embed
  ingest/            session liveness scan, JSONL tailer, burst coalescing
  roster/            villager identity: naming, village.json, avatars
  server/            HTTP routes, websocket hub, prompt transport
  demo/              fake sessions and transcripts for --demo
  static/            index.html, app.js, town.js, art.js, villagers.js,
                     behaviour.js, card.js, player.js, audio.js,
                     style.css, card.css, composer.css, smoke.html
```

The simulation lives in the browser, not in Go: the server emits events and the
page turns them into walking, working and sleeping. Go holds only what has to be
watched and forwarded.

Routes:

- `GET /` — the SPA
- `GET /ws` — activity events, roster diffs, clock ticks
- `GET /api/villagers` — current roster
- `GET /avatars/<sessionId>` — generated critter face
- `POST /api/prompt` — `{ sessionId, text }`

Ingest tails each transcript from EOF, using fsnotify plus a 1-second
mtime/size poll, and coalesces bursts (~250ms) into one `activity` event.
Liveness comes from re-scanning `sessions/`; a session file that disappears
produces a `gone` event.

The server listens on `0.0.0.0:8040` so a Tailscale node can reach it.

`--data` may be passed more than once; it defaults to `~/.claude`, the Claude
Code folder of the machine the village runs on. `--demo` generates ten to twelve
fake sessions in a temp directory that append real JSONL, so the village is
watchable with nothing running.

## Constraints

- Go 1.24+, standard library plus fsnotify and `gorilla/websocket` (the stdlib
  has no server-side WebSocket). Vanilla JS/CSS/HTML, no bundler.
- 60fps with ~30 villagers at 480×270.
- No audio files. No font files. No sprite sheets shipped from elsewhere.
- Never write into a Claude Code data directory.

## Testing

`go test ./...` covers: tail offsets and resumption, partial trailing lines,
oversized line truncation, unknown JSON shapes, burst coalescing, roster
spawn/gone, name derivation and collision suffixes, avatar determinism,
`village.json` parsing, and prompt routing including the unreachable path.

## Left to the builder

- The exact art direction *within* PS1 Animal Crossing: palette, ground tiles,
  house shapes, walk cycle. The design fixes the medium and the camera, not the
  specific drawing.
- Whether the messaging socket can actually be spoken to by a watcher. This is
  the one unverified claim; the task order checks it before the walking is built
  on top of it, and the fallback is already specified.
- Sprite generation strategy once the look is drawn: hand-authored pixel art
  versus procedural critters from the avatar seed.
