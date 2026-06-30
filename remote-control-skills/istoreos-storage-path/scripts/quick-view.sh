#!/bin/sh
set -eu

echo "## df -h"
df -h 2>&1 || true
echo ""
echo "## df -i"
df -i 2>&1 || true
echo ""
echo "## mount (head)"
mount 2>&1 | head -n 80 || true

