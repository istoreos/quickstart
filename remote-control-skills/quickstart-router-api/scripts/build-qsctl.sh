#!/bin/sh
set -eu

cd "$(dirname "$0")"
mkdir -p ../bin

build_one() {
  goos="$1"
  goarch="$2"
  suffix="$3"
  out="../bin/qsctl-${goos}-${goarch}${suffix}"
  echo "building ${out}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$out" qsctl.go
}

build_one linux amd64 ""
build_one linux arm64 ""
build_one darwin amd64 ""
build_one darwin arm64 ""
build_one windows amd64 ".exe"
