#!/bin/sh
set -eu

mode="copy"
force="0"
target=""

usage() {
  cat >&2 <<'EOF'
Usage:
  sh install.sh [--target DIR] [--copy|--symlink] [--force]
  sh install.sh --codex [--copy|--symlink] [--force]
  sh install.sh --agents [--copy|--symlink] [--force]
  sh install.sh --opencode [--copy|--symlink] [--force]

Defaults:
  --codex --copy

Installs each child directory containing SKILL.md as an individual skill.
EOF
}

codex_target() {
  if [ -n "${CODEX_HOME:-}" ]; then
    printf '%s\n' "$CODEX_HOME/skills"
    return 0
  fi

  if command -v list-codex-skills >/dev/null 2>&1; then
    detected="$(list-codex-skills --roots 2>/dev/null | awk '/^  / { print $1; exit }')"
    if [ -n "${detected:-}" ]; then
      printf '%s\n' "$detected"
      return 0
    fi
  fi

  for base in /config/.codex "$HOME/.codex" /config/.agents "$HOME/.agents"; do
    if [ -d "$base" ]; then
      printf '%s\n' "$base/skills"
      return 0
    fi
  done

  printf '%s\n' "$HOME/.codex/skills"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --copy)
      mode="copy"
      ;;
    --symlink)
      mode="symlink"
      ;;
    --force)
      force="1"
      ;;
    --codex)
      target="$(codex_target)"
      ;;
    --agents)
      if [ -d /config/.agents ] && [ ! -d "$HOME/.agents" ]; then
        target="/config/.agents/skills"
      else
        target="$HOME/.agents/skills"
      fi
      ;;
    --opencode)
      target="${XDG_CONFIG_HOME:-$HOME/.config}/opencode/skills"
      ;;
    --target)
      shift
      [ "$#" -gt 0 ] || { usage; exit 2; }
      target="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
  shift
done

[ -n "$target" ] || target="$(codex_target)"

root="$(CDPATH= cd "$(dirname "$0")" && pwd -P)"
mkdir -p "$target"

install_one() {
  src="$1"
  name="$(basename "$src")"
  dst="$target/$name"

  if [ -e "$dst" ] || [ -L "$dst" ]; then
    if [ "$force" != "1" ]; then
      echo "skip: $name already exists at $dst (use --force to replace)" >&2
      return 0
    fi
    rm -rf "$dst"
  fi

  case "$mode" in
    copy)
      cp -R "$src" "$dst"
      ;;
    symlink)
      ln -s "$src" "$dst"
      ;;
    *)
      echo "invalid mode: $mode" >&2
      exit 2
      ;;
  esac
  echo "installed: $name -> $dst" >&2
}

count=0
for skill in "$root"/*; do
  [ -d "$skill" ] || continue
  [ -f "$skill/SKILL.md" ] || continue
  install_one "$skill"
  count=$((count + 1))
done

echo "done: installed $count skills into $target" >&2
