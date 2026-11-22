# Scaffold and embedded static serving

The binary serves a single embedded page: a 480x270 canvas stage, the title `Bot Village`, and an `M` mute hint, letterboxed on a dark backdrop with no page scroll. This is the base every later feature stands on.

## Sub-features

- **S1 Launch.** `go build .` produces one binary; `--listen`, `--data` (repeatable, defaults to `~/.claude`) and `--demo` are accepted.
- **S2 Index.** `GET /` returns 200 from the embedded filesystem, with `Bot Village` in the HTML title.
- **S3 Stylesheet.** `GET /style.css` returns 200 with `image-rendering: pixelated` and `overflow: hidden` on the stage and the page.
- **S4 Stage.** The document contains exactly one `<canvas id="stage">` at 480x270 backing resolution.
- **S5 Nothing scrolls.** The document is not taller or wider than the viewport.

## How to get to it (user POV)

- Open `http://127.0.0.1:<port>/` in a browser. There is no login, no route, no query parameter: `/` is the app.
- Open `/style.css` to inspect the served stylesheet.

## Driving it with headless Chrome + curl

Preconditions: `scripts/serve.sh start <port> /tmp/bv-data-<run>` printed `ready`; `scripts/serve.sh doctor <port>` printed `ok`.

- **S1 Launch.** Start as above. Observable: the log's first line is `Bot Village listening on 127.0.0.1:<port>` and the helper's readiness loop exited.
- **S2 Index.**
  ```sh
  curl -sS -o /tmp/bv-proof/index.html -w '%{http_code} %{content_type}\n' http://127.0.0.1:<port>/
  grep -c 'Bot Village' /tmp/bv-proof/index.html
  ```
  Observable: `200 text/html; charset=utf-8` and a non-zero grep count.
- **S3 Stylesheet.**
  ```sh
  curl -sS http://127.0.0.1:<port>/style.css | grep -E 'overflow|pixelated'
  ```
  Observable: an `overflow: hidden` line and an `image-rendering: pixelated` line.
- **S4 Stage.**
  ```sh
  node .claude/skills/verify-bot-village/scripts/drive.mjs \
    --url http://127.0.0.1:<port>/ --wait '#stage' \
    --eval '({n: document.querySelectorAll("#stage").length, w: stage.width, h: stage.height, title: document.title})' \
    --shot /tmp/bv-proof/scaffold-stage.png
  ```
  Observable: `n: 1`, `w: 480`, `h: 270`, `title: "Bot Village"`.
- **S5 Nothing scrolls and the stage fits.**
  ```sh
  node .claude/skills/verify-bot-village/scripts/drive.mjs \
    --url http://127.0.0.1:<port>/ --wait '#stage' \
    --eval '({scrollW: document.documentElement.scrollWidth, clientW: document.documentElement.clientWidth, scrollH: document.documentElement.scrollHeight, clientH: document.documentElement.clientHeight, overflow: getComputedStyle(document.body).overflow, box: stage.getBoundingClientRect().toJSON()})'
  ```
  Observable: `scrollW == clientW`, `scrollH == clientH`, `overflow: "hidden"`, and `box.top >= 0 && box.left >= 0 && box.bottom <= clientH && box.right <= clientW`. The second half matters: `overflow: hidden` hides a stage that overflows the viewport, so a fixed-size stage looks fine in a screenshot and is actually cropped. Run the check at a short viewport too (`--window-size=960,400`), not only the default.

## Gotchas

- The canvas backing size (`width`/`height` attributes) is 480x270. Its CSS size is larger and may change per task; assert the attributes, not the box, for S4.
- Serving is `http.FileServerFS`, so `/` resolves to `index.html` and no directory listing appears. A 404 here almost always means `//go:embed` lost a file (the pattern in `village/static/embed.go` must match the file name).
- The hint is `position: fixed`; if a change makes it participate in layout, S5 catches it as page height growth.
- An overflowing stage still reports equal scroll and client sizes because `overflow: hidden` clips it. Do not treat equal sizes as proof the stage is visible; check the box against the viewport as S5 says.
- Chrome in this image is `dm-chrome` with `--no-sandbox`; run as root, not as another user.
