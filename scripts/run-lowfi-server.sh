#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DOC_ROOT="$PROJECT_DIR/docs/prototypes"
PAGE_FILE="lan-device-management/index.html"
PAGE_URL_PATH="lan-device-management/"

PORT=${PORT:-5173}
BIND_ADDRESS=${BIND_ADDRESS:-0.0.0.0}
RUNTIME_DIR=${RUNTIME_DIR:-/tmp/quickstart-lowfi-server}
PID_FILE="$RUNTIME_DIR/server.pid"
LOG_FILE="$RUNTIME_DIR/server.log"

mkdir -p "$RUNTIME_DIR"

read_pid() {
    if [ ! -f "$PID_FILE" ]; then
        return 1
    fi

    pid=$(sed -n '1p' "$PID_FILE")
    case "$pid" in
        ''|*[!0-9]*) return 1 ;;
    esac

    printf '%s\n' "$pid"
}

is_our_process() {
    pid=$1
    [ -r "/proc/$pid/cmdline" ] || return 1
    command_line=$(tr '\000' ' ' < "/proc/$pid/cmdline")
    case "$command_line" in
        *"python3 -m http.server $PORT"*"$DOC_ROOT"*) return 0 ;;
        *) return 1 ;;
    esac
}

lan_address() {
    hostname -I 2>/dev/null | awk '{ print $1 }'
}

print_url() {
    address=$(lan_address)
    if [ -n "$address" ]; then
        printf 'http://%s:%s/%s\n' "$address" "$PORT" "$PAGE_URL_PATH"
    else
        printf 'http://<LAN-IP>:%s/%s\n' "$PORT" "$PAGE_URL_PATH"
    fi
}

start_server() {
    if pid=$(read_pid 2>/dev/null) && kill -0 "$pid" 2>/dev/null && is_our_process "$pid"; then
        printf 'Low-fi server is already running (pid %s).\n' "$pid"
        print_url
        return 0
    fi

    rm -f "$PID_FILE"

    if ss -ltn "sport = :$PORT" 2>/dev/null | grep -q LISTEN; then
        printf 'Port %s is already in use.\n' "$PORT" >&2
        return 1
    fi

    if [ ! -f "$DOC_ROOT/$PAGE_FILE" ]; then
        printf 'Prototype page not found: %s\n' "$DOC_ROOT/$PAGE_FILE" >&2
        return 1
    fi

    nohup setsid python3 -m http.server "$PORT" \
        --bind "$BIND_ADDRESS" \
        --directory "$DOC_ROOT" \
        >"$LOG_FILE" 2>&1 </dev/null &
    pid=$!
    printf '%s\n' "$pid" > "$PID_FILE"

    attempts=0
    while [ "$attempts" -lt 50 ]; do
        if kill -0 "$pid" 2>/dev/null && curl -fsS "http://127.0.0.1:$PORT/$PAGE_FILE" >/dev/null 2>&1; then
            printf 'Low-fi server started (pid %s).\n' "$pid"
            print_url
            printf 'Log: %s\n' "$LOG_FILE"
            return 0
        fi
        attempts=$((attempts + 1))
        sleep 0.1
    done

    printf 'Low-fi server failed to start. See %s\n' "$LOG_FILE" >&2
    return 1
}

stop_server() {
    if ! pid=$(read_pid 2>/dev/null); then
        printf 'Low-fi server is not running.\n'
        return 0
    fi

    if ! kill -0 "$pid" 2>/dev/null; then
        rm -f "$PID_FILE"
        printf 'Removed stale PID file.\n'
        return 0
    fi

    if ! is_our_process "$pid"; then
        printf 'PID %s does not belong to this low-fi server; refusing to stop it.\n' "$pid" >&2
        return 1
    fi

    kill "$pid"
    attempts=0
    while kill -0 "$pid" 2>/dev/null && [ "$attempts" -lt 50 ]; do
        attempts=$((attempts + 1))
        sleep 0.1
    done
    rm -f "$PID_FILE"
    printf 'Low-fi server stopped.\n'
}

show_status() {
    if pid=$(read_pid 2>/dev/null) && kill -0 "$pid" 2>/dev/null && is_our_process "$pid"; then
        printf 'Low-fi server is running (pid %s).\n' "$pid"
        print_url
        return 0
    fi

    printf 'Low-fi server is not running.\n'
    return 1
}

case "${1:-start}" in
    start) start_server ;;
    stop) stop_server ;;
    restart)
        stop_server
        start_server
        ;;
    status) show_status ;;
    *)
        printf 'Usage: %s {start|stop|restart|status}\n' "$0" >&2
        exit 2
        ;;
esac
