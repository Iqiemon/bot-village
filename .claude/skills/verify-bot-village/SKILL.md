---
name: verify-bot-village
description: "Prove Bot Village behavior the way a user sees it: launch the Go binary, open the 480x270 canvas page in headless Chrome, drive it, and capture a screenshot plus JSON evidence. Use after changing the server, the embedded static files, the canvas renderer, villagers, the card, prompt delivery or demo mode."
---

# Verify Bot Village

Bot Village is one Go binary that serves an embedded 480x270 Canvas 2D village page and, in later tasks, a `/ws` WebSocket and `/api/*` JSON routes. The primary surface is a web page; the secondary surface is HTTP (`curl`). Everything is stdlib Go plus `//go:embed static`, so there is no dependency install beyond `go build`.

Work from the repo root. Helpers live in `scripts/` next to this file and are invoked by path.

## Launch

```sh
# Build, start on a free port with a disposable data dir, wait until GET / answers.
.claude/skills/verify-bot-village/scripts/serve.sh start 8041 /tmp/bv-verify-$USER/data
```

- Ready means `curl -sf http://127.0.0.1:8041/` returns the index, which the helper waits for before printing `ready`.
- The helper builds to `/tmp/bot-village-verify-<port>`, writes `/tmp/bot-village-verify-<port>.pid` and logs to `/tmp/bot-village-verify-<port>.log`.
- Always pass an explicit `--listen 127.0.0.1:<port>` and a scratch `--data` directory. Never point `--data` at the real `~/.claude` during verification; the server only reads it, but a live village is not deterministic.
- Use a port nobody else holds. 8040 is the default and may be in use; pick 8041+.
- `go` must be on `PATH` (Go 1.24+; on this image `/usr/local/go/bin` may need adding). `serve.sh start` runs `go build`, so it fails loudly when Go is missing rather than serving a stale binary.

## Doctor

```sh
.claude/skills/verify-bot-village/scripts/serve.sh doctor 8041
```

Prints `ok: pid <n> serves Bot Village on 8041`. It fails when the pid file is missing, the process is dead, or something other than Bot Village owns the port. Run it first whenever a drive returns nothing useful. It is read-only.

## Drive

Headless Chrome through the DevTools protocol, no Playwright needed:

```sh
node .claude/skills/verify-bot-village/scripts/drive.mjs \
  --url http://127.0.0.1:8041/ \
  --wait '#stage' \
  --eval '({title: document.title, w: stage.width, h: stage.height})' \
  --shot /tmp/bv-proof/01-stage.png
```

It prints one JSON object: `{ok, url, title, result, screenshot}`. `--eval` is any JS expression evaluated in the page after load plus one animation frame; it may return a promise. `--wait <css>` blocks until the selector exists (10s). `--shot <png>` writes a PNG and echoes its path.

For routes, use `curl -sS -o body -w '%{http_code}'`. For a WebSocket, `node` has a global `WebSocket`, so a frame can be read in a few lines from inside `--eval` when `/ws` exists.

Set `CHROME_PATH` if it is unset (`/usr/local/bin/dm-chrome` here). The helper launches Chrome with `--no-sandbox` through that wrapper and a throwaway `--user-data-dir` it deletes on exit.

## Evidence

Keep every proof under `/tmp/bv-proof/` with numbered names: `01-stage.png`, `02-console.json`. A proof needs:

- The action and the state after it, not only the last screen. Screenshot the stage and also print the values the page reports for it.
- What the page actually rendered, read back through `--eval` (`canvas.getContext('2d').getImageData(...)` for pixels, `getBoundingClientRect()` for layout). Never trust the DOM alone: a canvas draws nothing unless a script runs.
- For HTTP, the status code, `Content-Type`, and the relevant body bytes.
- The feature ID and the entry point used in the file name, so the map and the artifact line up.

Do not use test-only endpoints or internal setters to make a proof pass. Mocks are allowed only where a production boundary already isolates an external system, and then the proof must say which part was faked: `--demo` fakes prompt delivery by design.

## Cleanup

```sh
.claude/skills/verify-bot-village/scripts/serve.sh stop 8041
rm -rf /tmp/bv-data-<run>
```

`stop` kills only the pid it recorded in `/tmp/bot-village-verify-<port>.pid` and removes the built binary. Never kill by process name. `/tmp/bv-proof/` is evidence and outlives cleanup; delete it only when the proof has been read and recorded elsewhere.

## Features

Read `features/README.md` for the index, the baseline preconditions, and the per-feature drives. Current build state: only the scaffold (plan Task 1) is implemented, so only `scaffold.md` is drivable end to end. The other files are seeded from `docs/plan.md` and name the task that unblocks them; a run against an unimplemented feature reports an unmet precondition, never a pass.
