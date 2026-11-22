# Day/night and mute

The town follows local time, and `?hour=22` previews any hour as a visibly darker palette, not an alpha wash. `M` toggles mute; audio is created only after the first user gesture. **Blocked: plan Task 11.**

## Sub-features

- **D1 Hour override.** `town.hour()` returns the `?hour=` value when present, else local hours.
- **D2 Night palette.** The palette at hour 22 differs from hour 12, and the rendered ground is darker.
- **D3 Mute.** `M` toggles mute; a muted blip is not scheduled.
- **D4 No audio before gesture.** No `AudioContext` exists until the first click.

## How to get to it (user POV)

- Open `/?hour=22` and compare with `/`.
- Press `M` and listen, or read the mute state.

## Driving it with headless Chrome + curl

Preconditions: Task 11 landed.

- **D1 Hour.** `--url 'http://127.0.0.1:<port>/?hour=22' --eval 'town.hour()'`. Observable: `22`.
- **D2 Palette.** Compare `--eval 'JSON.stringify(audio.isMuted ? 1 : 0)'` and a ground pixel between `?hour=12` and `?hour=22`. Observable: different pixel values at the same coordinate.
- **D3 Mute.** `--eval 'audio.setMuted(true); audio.isMuted()'`. Observable: `true`, and no blip is scheduled.
- **D4 Gesture.** Load the page, read `--eval 'window.__audioCtxExisted || false'` or the equivalent debug surface, then click and read again. Observable: false, then true. If no debug surface exists, prove D4 by the absence of an `AudioContext` warning and the code path in `audio.unlock()`; say the check was indirect.

## Gotchas

- Night is a palette swap, not alpha. Comparing a composited screenshot can pass even when the map is tinted by an overlay; read the palette or a raw canvas pixel.
- Headless Chrome has no audio output; prove mute by the scheduled-node count or a state flag, never by ear.
- Audio autoplay policy blocks `AudioContext` before a gesture only in some Chrome flags. Assert the app's own unlock gate, not the browser's warning.
