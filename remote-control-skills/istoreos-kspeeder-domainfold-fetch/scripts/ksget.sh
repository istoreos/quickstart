#!/bin/sh
set -eu

# ksget: unified download entry for iStoreOS/OpenWrt sessions.
# Behavior: origin -> accel -> origin
#
# - Phase 1: try origin URL (A). If it doesn't finish within a timeout, kill it and treat as failure.
# - Phase 2: call KSpeeder DomainFold remap API (POST JSON) to get accelerated candidates (B).
# - Phase 3: try origin URL once more.
#
# Usage:
#   ksget.sh <URL>
#   ksget.sh -o <file> <URL>
#   ksget.sh -O <file> <URL>   # save to explicit path (wget-style)
#   ksget.sh -O <URL>          # save as basename of URL (curl-style -O)
#
# Env:
#   KSGET_TOOL=curl|wget|uclient-fetch (default: curl if present, else wget/uclient-fetch)
#   KSGET_RETRY=3
#   KSGET_CONNECT_TIMEOUT=10
#   KSGET_ORIGIN_MAX_TIME=40   (seconds; kill origin attempt after this time)
#   KSPEEDER_ADMIN_PORT=5003

usage() { echo "usage: $0 [-o <file>|-O <file>|-O] <URL>" >&2; }

out_file=""
save_basename=0
while [ $# -gt 0 ]; do
  case "$1" in
    -o)
      shift
      [ $# -gt 0 ] || { usage; exit 2; }
      out_file="$1"
      shift
      ;;
    -O)
      shift
      if [ $# -gt 0 ]; then
        case "$1" in
          -*|*://*)
            save_basename=1
            ;;
          *)
            out_file="$1"
            shift
            ;;
        esac
      else
        save_basename=1
      fi
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      echo "unsupported option: $1" >&2
      usage
      exit 2
      ;;
    *)
      break
      ;;
  esac
done

url="${1:-}"
[ -n "$url" ] || { usage; exit 2; }

admin_port="${KSPEEDER_ADMIN_PORT:-5003}"
retry="${KSGET_RETRY:-3}"
connect_timeout="${KSGET_CONNECT_TIMEOUT:-10}"
origin_max_time="${KSGET_ORIGIN_MAX_TIME:-40}"

basename_from_url() {
  u="$1"
  b="$(printf '%s' "$u" | sed 's/[?#].*$//' | awk -F/ '{print $NF}')"
  if [ -z "$b" ] || [ "$b" = "/" ]; then
    echo "download.bin"
  else
    echo "$b"
  fi
}

if [ "$save_basename" -eq 1 ] && [ -z "$out_file" ]; then
  out_file="$(basename_from_url "$url")"
fi

pick_tool() {
  if [ -n "${KSGET_TOOL:-}" ]; then
    echo "$KSGET_TOOL"
    return 0
  fi
  if command -v curl >/dev/null 2>&1; then
    echo "curl"
    return 0
  fi
  if command -v wget >/dev/null 2>&1; then
    echo "wget"
    return 0
  fi
  if command -v uclient-fetch >/dev/null 2>&1; then
    echo "uclient-fetch"
    return 0
  fi
  echo ""
}

tool="$(pick_tool)"
[ -n "$tool" ] || { echo "need: curl/wget/uclient-fetch not found." >&2; exit 2; }

now_epoch() { date +%s 2>/dev/null || echo 0; }

url_host() {
  u="$1"
  case "$u" in
    *://*)
      h="${u#*://}"
      h="${h%%/*}"
      h="${h##*@}"
      h="${h%%:*}"
      printf '%s' "$h"
      ;;
    *)
      printf '%s' ""
      ;;
  esac
}

print_attempt() { echo "attempt[$1]: url=$2" >&2; }
print_result() {
  idx="$1"; ok="$2"; elapsed="$3"
  if [ "$ok" = "1" ]; then
    echo "attempt[$idx]: ok elapsed_s=$elapsed" >&2
  else
    echo "attempt[$idx]: fail elapsed_s=$elapsed" >&2
  fi
}

download_to() {
  u="$1"
  dest="$2"
  case "$tool" in
    curl)
      curl -fL --retry "$retry" --connect-timeout "$connect_timeout" -o "$dest" "$u"
      ;;
    wget)
      wget -O "$dest" "$u"
      ;;
    uclient-fetch)
      uclient-fetch -O "$dest" "$u"
      ;;
    *)
      echo "unsupported tool: $tool" >&2
      exit 2
      ;;
  esac
}

kill_after() {
  secs="$1"
  pid="$2"
  [ -n "${secs:-}" ] || return 0
  [ -n "${pid:-}" ] || return 0
  if [ "$secs" = "0" ] 2>/dev/null; then
    return 0
  fi
  (
    sleep "$secs" 2>/dev/null || sleep 1
    kill "$pid" >/dev/null 2>&1 || true
    sleep 1 2>/dev/null || true
    kill -9 "$pid" >/dev/null 2>&1 || true
  ) &
  echo $!
}

try_one() {
  idx="$1"
  u="$2"
  max_time="${3:-0}"

  start="$(now_epoch)"
  print_attempt "$idx" "$u"

  if [ -n "$out_file" ]; then
    part="${out_file}.part.$$.$(now_epoch)"
    tmpe="/tmp/ksget.err.$$.$(now_epoch)"
    ( download_to "$u" "$part" ) > /dev/null 2>"$tmpe" &
    pid=$!
    killer=""
    if [ "${max_time}" != "0" ] 2>/dev/null; then
      killer="$(kill_after "$max_time" "$pid" || true)"
    fi
    ok=0
    if wait "$pid"; then
      if [ -s "$part" ]; then
        cat "$part" >"$out_file" 2>/dev/null || mv -f "$part" "$out_file"
        ok=1
      fi
    fi
    [ -n "$killer" ] && kill "$killer" >/dev/null 2>&1 || true
    rm -f "$part" >/dev/null 2>&1 || true
    if [ "$ok" -ne 1 ] && [ -s "$tmpe" ]; then
      echo "curl error (tail):" >&2
      tail -n 5 "$tmpe" >&2 || true
    fi
    rm -f "$tmpe" >/dev/null 2>&1 || true
    end="$(now_epoch)"
    elapsed=$((end - start))
    if [ "$ok" -eq 1 ]; then
      print_result "$idx" 1 "$elapsed"
      return 0
    fi
    print_result "$idx" 0 "$elapsed"
    return 1
  fi

  if download_to "$u" /dev/stdout >/dev/null 2>&1; then
    end="$(now_epoch)"; elapsed=$((end - start))
    print_result "$idx" 1 "$elapsed"
    return 0
  fi
  end="$(now_epoch)"; elapsed=$((end - start))
  print_result "$idx" 0 "$elapsed"
  return 1
}

json_escape() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'; }

remap_domainfold_json() {
  command -v curl >/dev/null 2>&1 || return 1
  payload="{\"url\":\"$(json_escape "$url")\"}"
  curl -fsS -H "Content-Type: application/json" -d "$payload" "http://127.0.0.1:${admin_port}/api/domainfold/remap"
}

json_get() {
  resp="$1"
  key="$2"
  printf '%s' "$resp" | tr -d '\n' | sed -n "s/.*\"$key\":\"\\([^\"]*\\)\".*/\\1/p" | head -n 1
}

accel_candidates=""
add_accel() {
  u="$1"
  [ -n "$u" ] || return 0
  accel_candidates="${accel_candidates}${u}
"
}

print_candidates() {
  echo "origin: $url" >&2
  if [ -n "$accel_candidates" ]; then
    echo "accel_candidates:" >&2
    tmpc="/tmp/ksget.candidates.$$.$(now_epoch)"
    printf '%s' "$accel_candidates" >"$tmpc"
    while IFS= read -r u; do
      [ -n "$u" ] || continue
      echo "  - $u" >&2
    done <"$tmpc"
    rm -f "$tmpc" >/dev/null 2>&1 || true
  else
    echo "accel_candidates: (none)" >&2
  fi
}

ensure_istoreenhance_running() {
  [ "${KSGET_AUTO_START:-0}" = "1" ] || return 0
  if command -v pidof >/dev/null 2>&1; then
    pidof iStoreEnhance >/dev/null 2>&1 && return 0
  elif command -v pgrep >/dev/null 2>&1; then
    pgrep -x iStoreEnhance >/dev/null 2>&1 && return 0
  fi
  if [ -x /etc/init.d/istoreenhance ]; then
    if [ "${CONFIRM_KSPEEDER_SERVICE_APPLY:-}" != "YES" ]; then
      echo "need-confirmation: set CONFIRM_KSPEEDER_SERVICE_APPLY=YES only after the user confirms enabling/starting /etc/init.d/istoreenhance." >&2
      return 0
    fi
    echo "action: start istoreenhance (best-effort) for domainfold" >&2
    /etc/init.d/istoreenhance enable >/dev/null 2>&1 || true
    /etc/init.d/istoreenhance start >/dev/null 2>&1 || /etc/init.d/istoreenhance restart >/dev/null 2>&1 || true
  fi
  return 0
}

host="$(url_host "$url")"
echo "policy: host=${host:-unknown} order=origin-accel-origin connect_timeout_s=$connect_timeout origin_max_time_s=$origin_max_time tool=$tool" >&2

idx=1
if try_one "$idx" "$url" "$origin_max_time"; then
  exit 0
fi

ensure_istoreenhance_running

remap="$(remap_domainfold_json 2>/dev/null || true)"
admin_path=""
entry_url=""
remap_err=""
if [ -n "$remap" ]; then
  admin_path="$(json_get "$remap" admin_path)"
  entry_url="$(json_get "$remap" output)"
  remap_err="$(json_get "$remap" error)"
fi

if [ -n "$admin_path" ]; then
  add_accel "http://127.0.0.1:${admin_port}${admin_path}"
fi
if [ -n "$entry_url" ]; then
  add_accel "$entry_url"
fi

if [ -n "$remap_err" ] && [ -z "$admin_path" ] && [ -z "$entry_url" ]; then
  echo "warn: domainfold remap error: $remap_err" >&2
fi

print_candidates

if [ -n "$accel_candidates" ]; then
  tmpa="/tmp/ksget.accel.$$.$(now_epoch)"
  printf '%s' "$accel_candidates" >"$tmpa"
  while IFS= read -r u; do
    [ -n "$u" ] || continue
    idx=$((idx + 1))
    if try_one "$idx" "$u" 0; then
      rm -f "$tmpa" >/dev/null 2>&1 || true
      exit 0
    fi
  done <"$tmpa"
  rm -f "$tmpa" >/dev/null 2>&1 || true
else
  echo "info: no accel candidates available (domainfold remap unsupported), skip accel phase" >&2
fi

idx=$((idx + 1))
if try_one "$idx" "$url" "$origin_max_time"; then
  exit 0
fi

echo "failed: origin -> accel -> origin all failed." >&2
exit 1
