#!/bin/sh
set -eu

root="$(CDPATH= cd "$(dirname "$0")" && pwd -P)"
parent="$(dirname "$root")"
base="$(basename "$root")"
format="tar.gz"
out=""

usage() {
  cat >&2 <<'EOF'
Usage:
  sh package.sh [--tar.gz|--zip|--all] [OUT]

Defaults:
  --tar.gz

Examples:
  sh package.sh --zip
  sh package.sh --zip /tmp/remote-control-skills.zip
  sh package.sh --all
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --tar.gz|--tgz)
      format="tar.gz"
      ;;
    --zip)
      format="zip"
      ;;
    --all)
      format="all"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      usage
      exit 2
      ;;
    *)
      [ -z "$out" ] || { usage; exit 2; }
      out="$1"
      ;;
  esac
  shift
done

abs_out() {
  case "$1" in
    /*) printf '%s\n' "$1" ;;
    *) printf '%s\n' "$(pwd)/$1" ;;
  esac
}

write_checksum() {
  file="$1"
  rm -f "$file.sha256"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" >"$file.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" >"$file.sha256"
  else
    echo "warning: sha256sum/shasum not found; checksum not written" >&2
  fi
}

make_targz() {
  file="$1"
  rm -f "$file" "$file.sha256"
  tar \
    --exclude "$base/*/bin" \
    --exclude "$base/.git" \
    --exclude "$base/.DS_Store" \
    --exclude "$base/remote-control-skills.tar.gz" \
    --exclude "$base/remote-control-skills.tar.gz.sha256" \
    --exclude "$base/remote-control-skills.zip" \
    --exclude "$base/remote-control-skills.zip.sha256" \
    -C "$parent" \
    -czf "$file" \
    "$base"
  write_checksum "$file"
}

make_zip() {
  file="$1"
  if ! command -v zip >/dev/null 2>&1; then
    echo "need: zip command not found" >&2
    exit 2
  fi

  rm -f "$file" "$file.sha256"
  (
    cd "$parent"
    zip -qr "$file" "$base" \
      -x "$base/*/bin/*" \
      -x "$base/.git/*" \
      -x "$base/.DS_Store" \
      -x "$base/remote-control-skills.tar.gz" \
      -x "$base/remote-control-skills.tar.gz.sha256" \
      -x "$base/remote-control-skills.zip" \
      -x "$base/remote-control-skills.zip.sha256"
  )
  write_checksum "$file"
}

case "$format" in
  tar.gz)
    out="$(abs_out "${out:-$parent/remote-control-skills.tar.gz}")"
    make_targz "$out"
    echo "archive: $out" >&2
    [ -f "$out.sha256" ] && echo "checksum: $out.sha256" >&2
    ;;
  zip)
    out="$(abs_out "${out:-$parent/remote-control-skills.zip}")"
    make_zip "$out"
    echo "archive: $out" >&2
    [ -f "$out.sha256" ] && echo "checksum: $out.sha256" >&2
    ;;
  all)
    [ -z "$out" ] || { echo "--all does not accept OUT" >&2; exit 2; }
    targz="$parent/remote-control-skills.tar.gz"
    zipfile="$parent/remote-control-skills.zip"
    make_targz "$targz"
    make_zip "$zipfile"
    echo "archive: $targz" >&2
    [ -f "$targz.sha256" ] && echo "checksum: $targz.sha256" >&2
    echo "archive: $zipfile" >&2
    [ -f "$zipfile.sha256" ] && echo "checksum: $zipfile.sha256" >&2
    ;;
esac
