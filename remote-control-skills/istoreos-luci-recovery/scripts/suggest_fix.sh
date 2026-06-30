#!/bin/sh
set -eu

path="${1:-}"

have() { command -v "$1" >/dev/null 2>&1; }

need() {
  echo "need: $*" >&2
  exit 2
}

section() {
  echo ""
  echo "## $*"
}

pm() {
  if have is-opkg; then
    echo "is-opkg"
    return 0
  fi
  if have opkg; then
    echo "opkg"
    return 0
  fi
  echo ""
}

if [ -z "$path" ]; then
  echo "usage: $0 /usr/lib/lua/luci/<path>.lua" >&2
  echo "example: $0 /usr/lib/lua/luci/dispatcher.lua" >&2
  exit 2
fi

case "$path" in
  /usr/lib/lua/luci/*) : ;;
  /*) echo "warn: path not under /usr/lib/lua/luci; still attempting owner lookup" >&2 ;;
  *) need "path must be absolute: $path" ;;
esac

pmm="$(pm)"
if [ -z "$pmm" ]; then
  need "neither is-opkg nor opkg found"
fi

rom="/rom$path"

section "context"
echo "file=$path"
echo "rom=$rom"
echo "pm=$pmm"

section "owner candidates (opkg search)"
owners=""
if have opkg; then
  opkg search "$path" 2>/dev/null | head -n 80 || true
  owners="$(opkg search "$path" 2>/dev/null | sed -n 's/^[[:space:]]*\\([^[:space:]]\\+\\)[[:space:]]\\+-[[:space:]].*$/\\1/p' | head -n 6 | tr '\n' ' ' || true)"
else
  echo "opkg=missing (cannot search owner reliably)"
fi

section "recommendation"
echo "priority: do not edit /usr/lib/lua/luci first; prefer soft recovery then reinstall the owning package."
echo "note: any reinstall/remove/edit is a risky action; recommend full system backup first (istoreos-backup-restore)."

section "step 1: soft recovery (recommended first)"
cat <<EOF
CONFIRM_LUCI_SOFT_RECOVERY=YES sh "$(dirname "$0")/soft_recover.sh"
EOF

section "step 2: reinstall owner package (recommended)"
if [ -n "${owners:-}" ]; then
  for p in $owners; do
    cat <<EOF
# Do not run until the user confirms package list update and reinstall risk.
$pmm update
$pmm install --force-reinstall $p
EOF
  done
else
  echo "no_owner_found=1"
  cat <<EOF
# Do not run until the user confirms package list update and reinstall risk.
$pmm update
$pmm install --force-reinstall luci-base
EOF
fi

section "step 3: restart services (manual; only after confirmation)"
cat <<'EOF'
# Prefer CONFIRM_LUCI_SOFT_RECOVERY=YES sh soft_recover.sh.
# If running manually, confirm the brief web UI interruption first.
/etc/init.d/rpcd restart || /etc/init.d/rpcd start || true
/etc/init.d/uhttpd restart || /etc/init.d/uhttpd start || true
EOF

section "step 4: rollback options (dangerous; use only if needed)"
echo "Option A) If the last installed plugin is suspected, remove it (example):"
echo "  $pmm remove <luci-app-xxx>    # risky; may remove dependencies"
echo ""
echo "Option B) If ROM baseline exists and you need a quick fallback, removing the overlay file may reveal /rom version:"
if [ -f "$rom" ]; then
  echo "  rm -f '$path'    # risky; only if you understand the impact; prefer package reinstall"
else
  echo "  rm -f '$path'    # rom baseline missing for this file; fallback may not work"
fi

section "next"
echo "1) Run diag first if you haven't: sh \"$(dirname "$0")/diag.sh\""
echo "2) If you have the exact lua traceback path, run trace_owner.sh to confirm ownership before reinstall."
