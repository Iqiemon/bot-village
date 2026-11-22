#!/bin/sh
# Launch Bot Village for verification and wait until it answers.
#
#   scripts/serve.sh start <port> <data-dir> [--demo]
#   scripts/serve.sh doctor <port>
#   scripts/serve.sh stop <port>
#
# start prints the pid file path. stop kills only the process this script started.

set -eu

cmd=${1:?usage: serve.sh start|doctor|stop <port> [data-dir] [--demo]}

case "$cmd" in
  start)
    port=${2:?port}
    data=${3:?data directory}
    shift 3
    pidfile="/tmp/bot-village-verify-$port.pid"
    if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
      echo "already running on $port (pid $(cat "$pidfile"))" >&2
      exit 1
    fi
    mkdir -p "$data"
    root=$(cd "$(dirname "$0")/../../../.." && pwd)
    cd "$root/village"
    go build -o "/tmp/bot-village-verify-$port" .
    nohup "/tmp/bot-village-verify-$port" --listen "127.0.0.1:$port" --data "$data" "$@" \
      > "/tmp/bot-village-verify-$port.log" 2>&1 &
    echo $! > "$pidfile"
    i=0
    until curl -sf -o /dev/null "http://127.0.0.1:$port/"; do
      i=$((i + 1))
      if [ "$i" -gt 100 ]; then
        echo "server never answered on $port; log:" >&2
        cat "/tmp/bot-village-verify-$port.log" >&2
        exit 1
      fi
      sleep 0.1
    done
    echo "ready on http://127.0.0.1:$port (pid $(cat "$pidfile"), log /tmp/bot-village-verify-$port.log)"
    ;;
  doctor)
    port=${2:?port}
    pidfile="/tmp/bot-village-verify-$port.pid"
    [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null || {
      echo "no verification instance on $port" >&2
      exit 1
    }
    body=$(curl -sf "http://127.0.0.1:$port/") || {
      echo "port $port answered nothing" >&2
      exit 1
    }
    case "$body" in
      *"Bot Village"*) echo "ok: pid $(cat "$pidfile") serves Bot Village on $port" ;;
      *) echo "wrong app on $port" >&2; exit 1 ;;
    esac
    ;;
  stop)
    port=${2:?port}
    pidfile="/tmp/bot-village-verify-$port.pid"
    if [ -f "$pidfile" ]; then
      kill "$(cat "$pidfile")" 2>/dev/null || true
      rm -f "$pidfile"
    fi
    rm -f "/tmp/bot-village-verify-$port"
    echo "stopped $port"
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 2
    ;;
esac
