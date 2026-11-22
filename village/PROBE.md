# Prompt transport probe

What this machine's Claude Code session sockets actually do when a watcher
speaks to them. Run against the session that wrote this repo
(`/run/user/0/cc-socks/1960.sock`, Claude Code 2.1.327, `authRequired` false),
2026-11-22.

## Result

The socket takes the write and says nothing back.

| What was sent | Result |
| --- | --- |
| No bytes, connection held open | No reply, no close within 25 s |
| Partial line, no newline | No reply, no close within 25 s |
| A complete `{"type":"user",…}` line, no auth | The line appeared in the session |
| A complete line with a valid auth line first | The line appeared in the session |
| Garbage, or a bogus `{"type":"auth"}` token | No reply, no error, no close within 4 s |

So: **the socket accepts and injects; it never acknowledges.** A successful
write is silence, and a rejected one is the same silence. That is why `Send`
returns `nil` on a completed write rather than claiming delivery.

Timing is not observable from this end: each injected line showed up in the
session's transcript on the session's next turn, not as a socket reply. A line
without a trailing newline also reached the session. Read from the binary's own
strings, an inbox closes a connection that sends no complete line within
`firstLineDeadlineMs` (default 30 s), which is why the table caps its silent
rows at 25 s.

The inbox inspects nothing before it accepts a connection, so an
unauthenticated watcher can inject. Auth on this machine is optional
(`authRequired` false), and a valid token from the sibling `.key` file is
accepted too. Neither is needed, so `Send` does not read it.

Sockets that are not there fail to connect. An absent path, a socket file left
behind by a dead process, a regular file, and a directory all fail at `connect`
with `ENOENT` or `ECONNREFUSED`, which is what `Send` maps to `ErrUnreachable`.

## How it was probed

`prompt.Probe(path)` is the one-shot check: it sends one line and reports `nil`
for an accepted write. The table's no-bytes and partial-line rows came from raw
`unix(7)` connections that held the socket open without a complete line. The
line, bad-token and garbage rows came from the same writes by hand, and the
refusal row from `Send` against a dead socket path and an absent one.

## Limit

This says nothing about whether the session's own UI shows the message, only
that the protocol accepted it. Delivery is fire-and-forget by design, so a
write that completes is the strongest signal the socket offers.
