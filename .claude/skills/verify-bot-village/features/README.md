# Bot Village verification map

The maintained source for proving Bot Village's user-facing behavior. Read this index first, then open the feature file for the recipe.

The plan (`docs/plan.md`) builds the app in 15 tasks. Features below are listed in the order they become drivable. A feature whose task is not implemented yet is marked **blocked**, with the task that unblocks it.

## Baseline preconditions

- Start from the repo root with a buildable checkout: `cd village && go test ./...` passes.
- Launch with `.claude/skills/verify-bot-village/scripts/serve.sh start <port> /tmp/bv-data-<run>` so the instance, its port, and its data dir belong to this run.
- Use port 8041 or higher, bound to `127.0.0.1`.
- Run the doctor before driving. Never drive an instance this run did not start.
- `CHROME_PATH=/usr/local/bin/dm-chrome`, headless, throwaway profile.

## Driving conventions

- Browser actions go through `node .claude/skills/verify-bot-village/scripts/drive.mjs`.
- HTTP actions go through `curl`.
- Prefer stable handles: `#stage`, `#card`, `#prompt`, `/api/villagers`, `/ws`. Do not click by pixel until a handle is missing, and then say so in the feature's Gotchas.
- Every command is literal. Keep ports and paths as written, substituting only the run's port and run id.
- Evidence goes to `/tmp/bv-proof/` and survives cleanup.

## Proof and skip reporting

- Capture the action and the resulting state. A final screenshot alone is not proof of an interaction.
- Canvas features also read back pixels with `getImageData`, because the DOM cannot show what was drawn.
- Record the feature ID and the entry point with each artifact.
- Report an unreachable path with the command tried and the unmet precondition. Never report it as verified through another path.

## Feature entry contract

Each file starts with an H1 and one paragraph on the user-visible behavior, then exactly four H2s: `Sub-features`, `How to get to it (user POV)`, `Driving it with headless Chrome + curl`, `Gotchas`.

## Features

- [Scaffold and embedded static serving](./scaffold.md) (Task 1, implemented) covers launch, the embedded index, the 480x270 stage, the title, the M hint and the letterboxed layout.
- [Live villagers on the map](./villagers.md) (Tasks 3-9, blocked) covers sessions becoming villagers that walk and work. Needs `GET /api/villagers` and `/ws`.
- [Villager card](./card.md) (Task 12, blocked) covers clicking a villager and reading its name and title.
- [Prompt a villager](./prompt.md) (Tasks 2, 13, blocked) covers selecting a villager, typing a message and reading the delivery status.
- [Day/night and mute](./day-night.md) (Task 11, blocked) covers `?hour=22` and the `M` key.
- [Demo mode](./demo.md) (Task 14, blocked) covers `--demo` generating fake sessions.
