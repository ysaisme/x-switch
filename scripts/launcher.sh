#!/bin/bash
set -e

CONTENTS_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
MSWITCH_BIN="$RESOURCES_DIR/mswitch"
PID_FILE="$HOME/.mswitch/mswitch.pid"
WEB_URL="http://127.0.0.1:9091"

if [ ! -x "$MSWITCH_BIN" ]; then
    osascript -e 'display dialog "mswitch binary not found.\n\nExpected: '"$MSWITCH_BIN"'" buttons {"OK"} default button 1 with title "mswitch Error" with icon stop'
    exit 1
fi

cleanup() {
    echo "[mswitch] shutting down..."
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE" 2>/dev/null)
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            kill "$PID" 2>/dev/null
        fi
        rm -f "$PID_FILE"
    fi
    exit 0
}

trap cleanup SIGTERM SIGINT

if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE" 2>/dev/null)
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
        echo "[mswitch] already running (PID: $OLD_PID), opening browser..."
        open "$WEB_URL"
        exit 0
    fi
    rm -f "$PID_FILE"
fi

CONFIG_DIR="$HOME/.mswitch"
mkdir -p "$CONFIG_DIR"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    "$MSWITCH_BIN" init --defaults 2>/dev/null || true
fi

echo "[mswitch] starting proxy server..."
"$MSWITCH_BIN" start &

sleep 2

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE" 2>/dev/null)
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
        echo "[mswitch] running (PID: $PID), opening browser..."
        open "$WEB_URL"
    else
        echo "[mswitch] warning: service may not have started correctly"
        open "$WEB_URL"
    fi
else
    echo "[mswitch] warning: pid file not found, opening browser anyway..."
    open "$WEB_URL"
fi

wait
