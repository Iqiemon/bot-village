#!/usr/bin/env node
// Drives the Bot Village page in headless Chrome over the DevTools protocol.
//
//   node scripts/drive.mjs --url http://127.0.0.1:8040/ [--eval "expr"] [--wait selector] [--shot out.png]
//
// Prints one JSON object on stdout: {ok, url, title, result, screenshot}.
// Exits non-zero when the page failed to load or the expression threw.

import { spawn } from "node:child_process";
import { writeFileSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const CHROME = process.env.CHROME_PATH || "/usr/local/bin/dm-chrome";

function arg(name, fallback = undefined) {
  const i = process.argv.indexOf(`--${name}`);
  return i === -1 ? fallback : process.argv[i + 1];
}

const url = arg("url");
if (!url) {
  console.error("usage: drive.mjs --url <url> [--eval <expr>] [--wait <css>] [--shot <png>]");
  process.exit(2);
}
const expression = arg("eval", "document.title");
const waitFor = arg("wait");
const shot = arg("shot");

const profile = mkdtempSync(join(tmpdir(), "bv-drive-"));
const chrome = spawn(CHROME, [
  "--headless=new",
  "--disable-gpu",
  "--hide-scrollbars",
  "--window-size=960,540",
  "--remote-debugging-port=0",
  `--user-data-dir=${profile}`,
  "about:blank",
], { stdio: ["ignore", "ignore", "pipe"] });

const wsURL = await new Promise((resolve, reject) => {
  let buf = "";
  const timer = setTimeout(() => reject(new Error(`chrome did not report a devtools port: ${buf}`)), 15000);
  chrome.stderr.on("data", (d) => {
    buf += d;
    const m = buf.match(/DevTools listening on (ws:\/\/\S+)/);
    if (m) {
      clearTimeout(timer);
      resolve(m[1]);
    }
  });
  chrome.on("exit", (code) => {
    clearTimeout(timer);
    reject(new Error(`chrome exited early with ${code}: ${buf.slice(-500)}`));
  });
});

const ws = new WebSocket(wsURL);
await new Promise((resolve, reject) => {
  ws.addEventListener("open", resolve, { once: true });
  ws.addEventListener("error", reject, { once: true });
});

let nextID = 0;
function send(method, params = {}, sessionId = undefined) {
  const id = ++nextID;
  ws.send(JSON.stringify({ id, method, params, ...(sessionId ? { sessionId } : {}) }));
  return new Promise((resolve, reject) => {
    const onMessage = (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.id !== id) return;
      ws.removeEventListener("message", onMessage);
      msg.error ? reject(new Error(`${method}: ${msg.error.message}`)) : resolve(msg.result);
    };
    ws.addEventListener("message", onMessage);
  });
}

function once(method) {
  return new Promise((resolve) => {
    const onMessage = (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.method !== method) return;
      ws.removeEventListener("message", onMessage);
      resolve(msg.params);
    };
    ws.addEventListener("message", onMessage);
  });
}

const cleanup = () => {
  try { ws.close(); } catch {}
  chrome.kill("SIGKILL");
  try { rmSync(profile, { recursive: true, force: true }); } catch {}
};
process.on("exit", cleanup);

const { targetId } = await send("Target.createTarget", { url: "about:blank" });
const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });

await send("Page.enable", {}, sessionId);
await send("Runtime.enable", {}, sessionId);
const loaded = once("Page.loadEventFired");
await send("Page.navigate", { url }, sessionId);
await loaded;

if (waitFor) {
  const deadline = Date.now() + 10000;
  for (;;) {
    const { result } = await send("Runtime.evaluate", {
      expression: `!!document.querySelector(${JSON.stringify(waitFor)})`,
      returnByValue: true,
    }, sessionId);
    if (result.value) break;
    if (Date.now() > deadline) throw new Error(`selector never appeared: ${waitFor}`);
    await new Promise((r) => setTimeout(r, 100));
  }
}

// Give canvas scripts one animation frame before observing.
await send("Runtime.evaluate", { expression: "new Promise(r => requestAnimationFrame(() => setTimeout(r, 50)))", awaitPromise: true }, sessionId);

const { result, exceptionDetails } = await send("Runtime.evaluate", {
  expression, returnByValue: true, awaitPromise: true,
}, sessionId);
if (exceptionDetails) throw new Error(exceptionDetails.exception?.description || "expression threw");

const page = await send("Runtime.evaluate", {
  expression: "({title: document.title, url: location.href})", returnByValue: true,
}, sessionId);

let screenshot = null;
if (shot) {
  const { data } = await send("Page.captureScreenshot", { format: "png" }, sessionId);
  writeFileSync(shot, Buffer.from(data, "base64"));
  screenshot = shot;
}

console.log(JSON.stringify({
  ok: true,
  url: page.result.value.url,
  title: page.result.value.title,
  result: result.value,
  screenshot,
}, null, 2));
cleanup();
process.exit(0);
