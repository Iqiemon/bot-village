# Villager card

Clicking a villager opens a slim card showing its avatar, name, optional title, and current activity as one goal word plus a role chip. The card floats above the stage and never moves the map. **Blocked: plan Task 12.**

## Sub-features

- **C1 Open.** Clicking a villager opens the card with that villager's name and title.
- **C2 Title optional.** A villager with no title shows no title line.
- **C3 No metrics.** Card text matches none of `/\d+\s*(lines|bytes|tokens)/` and shows no transcript snippet.
- **C4 Map fixed.** `#stage.getBoundingClientRect()` is identical before and after the card opens.
- **C5 Close.** The close button closes the card and takes focus with `{preventScroll: true}`.

## How to get to it (user POV)

- Click a villager on the map.
- Press the card's close button to dismiss it.

## Driving it with headless Chrome + curl

Preconditions: Task 12 landed and at least one villager is on the map (see `villagers.md` V1).

- **C1 Open.** Drive a click at the villager's box (use `villagers.pick` coordinates from the page, or the villager's reported `x`,`y`). Observable: `#card` exists and contains the villager's name.
- **C2 Title optional.** Seed a `village.json` without a title for that session. Observable: no empty title element in `#card`.
- **C3 No metrics.** `--eval 'document.querySelector("#card").innerText'` and test it in the page with the plan's regex. Observable: no match.
- **C4 Map fixed.** Capture `document.querySelector("#stage").getBoundingClientRect()` to JSON before the click and after the card opens, then compare in the page. Observable: identical values.
- **C5 Close.** Click the close button, then read `document.activeElement`. Observable: `#card` gone and focus back on the stage or the close target per implementation.

## Gotchas

- Canvas click targets are pixel coordinates; assert `villagers.pick(x, y)` in the page before clicking, so a miss is reported as a miss.
- A card in normal flow passes C1 and fails C4. Check C4 explicitly.
- Screenshots alone do not prove focus; read `document.activeElement`.
