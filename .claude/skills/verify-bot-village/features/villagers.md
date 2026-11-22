# Live villagers on the map

Each running Claude Code session appears as a villager on the town ground: it walks to a work spot when its transcript grows, shows a work overlay for tool calls, idles with small activities, and sleeps in the open with a `Zzz` when quiet. **Blocked: plan Tasks 3-9.** Drivable once `GET /api/villagers` and `/ws` exist.

## Sub-features

- **V1 Spawn.** A session in `<data>/.home/.claude/sessions/` becomes one villager with a derived name.
- **V2 Activity.** Transcript growth walks the villager to its work spot and draws a work overlay.
- **V3 Gone.** A removed session file removes the villager; it is never left standing forever.
- **V4 Asleep.** 90s quiet leaves the villager asleep at the same position with a `Zzz`.
- **V5 No metrics.** No villager payload, nametag or bubble contains line, token or byte counts.

## How to get to it (user POV)

- Open the app; villagers are already on the map.
- Watch while a transcript is appended to; the matching villager walks and works.

## Driving it with headless Chrome + curl

Preconditions: Task 9 landed; a data dir is seeded with at least one session JSON and its transcript (see `scripts/serve.sh start` and the demo seeder in Task 14).

- **V1 Spawn.** `curl -sS http://127.0.0.1:<port>/api/villagers` then read frames from `/ws`. Observable: a roster containing one villager whose name derives from the seeded `cwd` folder.
- **V2 Activity.** Append a line to the seeded transcript, then read the next `/ws` frame. Observable: an `activity` frame, and a screenshot showing the villager away from its idle spot.
- **V3 Gone.** Delete the session JSON, wait one poll interval. Observable: a `gone` frame and the villager gone from the next roster.
- **V4 Asleep.** Leave the transcript quiet for 90s (or drive the clock). Observable: screenshot shows `Zzz` and the villager's position is unchanged from the quiet start.
- **V5 No metrics.** `curl -sS http://127.0.0.1:<port>/api/villagers | grep -Ei 'lines|bytes|tokens|snippet'`. Observable: no match.

## Gotchas

- Verification must never write into a real Claude Code data dir; seed a scratch one.
- Reading `/ws` needs a WebSocket client. Node has a global `WebSocket`; read exactly one frame per expected event and fail on timeout rather than looping forever.
- Liveness comes from the `sessions/` directory, not the transcript alone; deleting only the transcript does not remove a villager.
- Timings: 1s poll, 250ms burst window. Assert state after at least one full poll plus burst window, not immediately.
