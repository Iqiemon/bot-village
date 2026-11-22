# Demo mode

`--demo` generates a living village from a temp directory so the app can be shown without a real Claude Code installation. Delivery is faked and the app says so. **Blocked: plan Task 14.**

## Sub-features

- **M1 Seed.** `--demo` creates N sessions and transcripts in a temp dir and the village shows N villagers.
- **M2 Living.** Transcripts grow every 2-6s, so villagers keep walking and working.
- **M3 Faked delivery stated.** The console/README says delivery is faked; no prompt claims real success.

## How to get to it (user POV)

- Run the binary with `--demo` and open the printed URL.

## Driving it with headless Chrome + curl

Preconditions: Task 14 landed.

- **M1 Seed.** `.claude/skills/verify-bot-village/scripts/serve.sh start <port> /tmp/bv-data-<run> --demo`, then `curl -sS http://127.0.0.1:<port>/api/villagers`. Observable: a non-empty roster of the documented demo count.
- **M2 Living.** Capture a screenshot, wait 10s, capture another. Observable: villager positions or overlays differ.
- **M3 Faked delivery.** Read the startup log and send a prompt. Observable: the log states delivery is faked and the status path is UI-only.

## Gotchas

- Demo mode must use only its temp dir; check the log's data path before trusting the roster, or a real `~/.claude` may be leaking in.
- Demo counts and intervals are documented in `village/README.md` (Task 15). Read that, not the plan, for the expected number.
