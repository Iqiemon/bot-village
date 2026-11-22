# Prompt a villager

Selecting a villager enables the single-line prompt; Enter sends. The player walks to the target, the status reads `on my way`, then `said it` or `couldn't reach them`, and a bubble shows about 22 characters of what you said. **Blocked: plan Tasks 2 and 13.**

## Sub-features

- **P1 Disabled until selected.** With nothing selected the input is disabled with placeholder `click a villager first…`; after selecting it reads `message <name>…`.
- **P2 Delivery status.** `POST /api/prompt` returns `200 {"status":"delivered"}`, `502 {"status":"unreachable"}` for a reachable session with no socket, `400` for blank text, `404` for an unknown session.
- **P3 Optimistic walk.** The walk completes even when delivery fails; only the status changes.
- **P4 Bubble clip.** The bubble shows about 22 characters of the sent text, never transcript content.

## How to get to it (user POV)

- Click a villager, type into the prompt, press Enter.
- Read the status line under the map.

## Driving it with headless Chrome + curl

Preconditions: Tasks 2 and 13 landed; a villager is selected.

- **P1 Disabled.** `--eval '({disabled: document.querySelector("#prompt input").disabled, ph: document.querySelector("#prompt input").placeholder})'`. Observable: `disabled: true`, placeholder `click a villager first…` before selection; after selecting a villager, `disabled: false` and `message <name>…`.
- **P2 Routes.** `curl -sS -o /tmp/bv-proof/prompt.json -w '%{http_code}\n' -X POST -d '{"sessionId":"<id>","text":"hello"}' http://127.0.0.1:<port>/api/prompt`. Observable: `502` and status `unreachable` when no socket exists; `400` for `"   "`; `404` for an unknown id.
- **P3 Walk completes.** Send to an unreachable session. Observable: villager reaches the target and the status reads `couldn't reach them`; the villager is not stuck mid-walk.
- **P4 Bubble.** Send text longer than 22 chars. Observable: the bubble shows a clipped version at roughly 22 characters and none of the transcript.

## Gotchas

- `--demo` fakes delivery by design. A `delivered` status under `--demo` proves the UI path only; say so in the evidence.
- The 2s socket timeout bounds P2. Do not raise the verify timeout above the route's own timeout or the status will be read too early.
- The input is inside a fixed overlay; the map must not move when it focuses. Pair P1 with `card.md` C4 when editing CSS.
