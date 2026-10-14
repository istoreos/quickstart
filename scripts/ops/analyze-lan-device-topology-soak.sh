#!/bin/sh
set -eu

input_path="${1:-}"
minimum_seconds="${M64_MIN_SECONDS:-86400}"
[ -n "$input_path" ] && [ -r "$input_path" ] || { echo 'usage: analyze-lan-device-topology-soak.sh <soak.tsv>' >&2; exit 2; }
case "$minimum_seconds" in ''|*[!0-9]*) echo 'M64_MIN_SECONDS must be an integer' >&2; exit 2 ;; esac

awk -F '\t' -v minimum="$minimum_seconds" '
NR == 1 {
    if (NF != 24 || $1 != "utc" || $2 != "epoch") { print "invalid soak header" > "/dev/stderr"; exit 2 }
    next
}
NR == 2 {
    first_epoch=$2; a_pid=$3; a_start=$4; a_rss_first=$5; b_pid=$10; b_start=$11; b_rss_first=$12
    a_rss_min=a_rss_max=$5; b_rss_min=b_rss_max=$12; a_fd_max=$7; b_fd_max=$14; samples=0
}
NR > 1 {
    samples++; last_epoch=$2; a_rss_last=$5; b_rss_last=$12
    if ($3 != a_pid || $4 != a_start || $10 != b_pid || $11 != b_start) fail("quickstart restarted")
    if ($8 != "200" || $15 != "200") fail("API was not HTTP 200")
    if ($17 != "absent" || $18 != "present") fail("floating gateway holder changed")
    if ($19 != "192.168.30.244" || $21 != "ok") fail("C route or connectivity failed")
    if ($22 != "192.168.30.1" || $24 != "ok") fail("D route or connectivity failed")
    if ($5 !~ /^[0-9]+$/ || $12 !~ /^[0-9]+$/ || $7 !~ /^[0-9]+$/ || $14 !~ /^[0-9]+$/) fail("invalid resource sample")
    if ($5 < a_rss_min) a_rss_min=$5; if ($5 > a_rss_max) a_rss_max=$5
    if ($12 < b_rss_min) b_rss_min=$12; if ($12 > b_rss_max) b_rss_max=$12
    if ($7 > a_fd_max) a_fd_max=$7; if ($14 > b_fd_max) b_fd_max=$14
    if ($6 > 64 || $13 > 64 || $7 > 128 || $14 > 128) fail("thread or fd budget exceeded")
}
function fail(message) { errors++; if (errors <= 10) print "sample " NR ": " message > "/dev/stderr" }
END {
    if (NR < 3) { print "soak needs at least two samples" > "/dev/stderr"; exit 1 }
    duration=last_epoch-first_epoch
    if (duration < minimum) { print "soak duration " duration "s is below " minimum "s" > "/dev/stderr"; errors++ }
    if (a_rss_last > a_rss_first * 1.25 + 8192 || b_rss_last > b_rss_first * 1.25 + 8192) { print "terminal RSS growth budget exceeded" > "/dev/stderr"; errors++ }
    if (a_rss_max > a_rss_min * 2 + 32768 || b_rss_max > b_rss_min * 2 + 32768) { print "RSS range budget exceeded" > "/dev/stderr"; errors++ }
    if (errors) exit 1
    printf "SUMMARY result=pass duration_seconds=%d samples=%d a_rss_first=%d a_rss_last=%d a_rss_max=%d b_rss_first=%d b_rss_last=%d b_rss_max=%d a_fd_max=%d b_fd_max=%d\n", duration, samples, a_rss_first, a_rss_last, a_rss_max, b_rss_first, b_rss_last, b_rss_max, a_fd_max, b_fd_max
}
' "$input_path"
