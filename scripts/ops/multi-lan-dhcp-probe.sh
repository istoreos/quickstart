#!/bin/sh
# Run only on an explicitly authorized writable test router. All links live in netns.
set -eu
case "$(uci -q get network.wan.ipaddr)" in
  192.168.30.93|192.168.30.7) ;;
  *) ip -4 addr show dev eth0 | grep -Eq '192\.168\.30\.(93|7)/' || exit 70 ;;
esac
base=/tmp/quickstart-multi-lan/dhcp-probe-$$
srv=qsml-s-$$
a=qsml-a-$$
b=qsml-b-$$
mkdir -p "$base"
fingerprint() {
  sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall
  for pid in $(pidof dnsmasq); do
    [ "$(readlink "/proc/$pid/ns/net" 2>/dev/null)" = "$(readlink /proc/self/ns/net)" ] || continue
    printf 'dnsmasq=%s\n' "$pid"
  done
}
fingerprint > "$base/before"
cleanup() {
  for ns in "$srv" "$a" "$b"; do
    for pid in $(ip netns pids "$ns" 2>/dev/null); do kill "$pid" 2>/dev/null || true; done
    ip netns del "$ns" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM
for ns in "$srv" "$a" "$b"; do ip netns add "$ns"; ip -n "$ns" link set lo up; done
for side in a b; do
  client="$a"; prefix=192.0.2
  if [ "$side" = b ]; then client="$b"; prefix=198.51.100; fi
  ip -n "$srv" link add "lan$side" type veth peer name "client$side"
  ip -n "$srv" link set "client$side" netns "$client"
  ip -n "$srv" addr add "$prefix.1/24" dev "lan$side"
  ip -n "$srv" link set "lan$side" up
  ip -n "$client" link set "client$side" address 02:00:00:00:00:55
  ip -n "$client" link set "client$side" up
done
cat > "$base/dnsmasq.conf" <<EOF
port=0
user=root
no-hosts
no-resolv
bind-interfaces
interface=lana
interface=lanb
no-ping
dhcp-authoritative
log-dhcp
log-facility=$base/dnsmasq.log
pid-file=$base/dnsmasq.pid
dhcp-leasefile=$base/leases
dhcp-range=set:office,192.0.2.100,192.0.2.120,255.255.255.0,10m
dhcp-range=set:guest,198.51.100.100,198.51.100.120,255.255.255.0,10m
dhcp-host=tag:lana,02:00:00:00:00:55,set:route_a,192.0.2.55
dhcp-host=tag:lanb,02:00:00:00:00:55,set:route_b,198.51.100.55
dhcp-option=tag:route_a,3,192.0.2.2
dhcp-option=tag:route_a,6,192.0.2.2
dhcp-option=tag:route_b,3,198.51.100.2
dhcp-option=tag:route_b,6,198.51.100.2
EOF
if [ "$#" -ge 1 ]; then
  fixture="$1"
  [ -f "$fixture/dhcp" ] || exit 71
  # Source definitions only, never call start_service/reload. Render task-owned
  # UCI with the router's actual host/tag compiler into our isolated config.
  UCI_CONFIG_DIR="$fixture" CONFIGFILE_TMP="$base/generated.conf" HOSTFILE_TMP="$base/hosts" sh -c '
    . /lib/functions.sh
    . /etc/init.d/dnsmasq
    config_load dhcp
    config_foreach dhcp_host_add host
    config_foreach dhcp_tag_add tag
    config_get extra main extraconftext
    [ -z "$extra" ] || echo -e "$extra" >> "$CONFIGFILE_TMP"
  '
  # Replace only the handcrafted host/option lines of this test template.
  sed -i '/^dhcp-host=/d; /^dhcp-option=/d' "$base/dnsmasq.conf"
  cat "$base/generated.conf" >> "$base/dnsmasq.conf"
  sed -i '/^dhcp-range=/d' "$base/dnsmasq.conf"
  for side in a b; do
    prefix=192.0.2; network=office
    if [ "$side" = b ]; then prefix=198.51.100; network=guest; fi
    section="pool_$side"
    ignore=$(uci -c "$fixture" -q get "dhcp.$section.ignore" || true)
    [ "$ignore" != 1 ] || continue
    start=$(uci -c "$fixture" -q get "dhcp.$section.start")
    limit=$(uci -c "$fixture" -q get "dhcp.$section.limit")
    lease=$(uci -c "$fixture" -q get "dhcp.$section.leasetime" || true)
    end=$((start + limit - 1))
    printf 'dhcp-range=set:%s,%s.%s,%s.%s,255.255.255.0,%s\n' "$network" "$prefix" "$start" "$prefix" "$end" "${lease:-12h}" >> "$base/dnsmasq.conf"
  done
fi
cat > "$base/client-script" <<'EOF'
#!/bin/sh
if [ "$1" = bound ] || [ "$1" = renew ]; then
  printf 'ip=%s router=%s dns=%s\n' "$ip" "$router" "$dns" >> "$QSML_RESULT"
fi
EOF
chmod 700 "$base/client-script"
ip netns exec "$srv" dnsmasq --conf-file="$base/dnsmasq.conf"
for side in a b a b; do
  client="$a"; if [ "$side" = b ]; then client="$b"; fi
  if [ "${2:-static}" = disabled ] && [ "$side" = a ]; then
    if QSML_RESULT="$base/result-$side" ip netns exec "$client" udhcpc -i "client$side" -n -q -t 2 -T 1 -s "$base/client-script" >/dev/null 2>&1; then exit 72; fi
    continue
  fi
  QSML_RESULT="$base/result-$side" ip netns exec "$client" udhcpc -i "client$side" -n -q -t 3 -T 1 -s "$base/client-script" >/dev/null 2>&1
done
[ ! -f "$base/result-a" ] || cat "$base/result-a"
cat "$base/result-b"
cat "$base/leases"
if [ "${2:-static}" = static ]; then
grep -q 'ip=192.0.2.55 router=192.0.2.2 dns=192.0.2.2' "$base/result-a"
grep -q 'ip=198.51.100.55 router=198.51.100.2 dns=198.51.100.2' "$base/result-b"
elif [ "${2:-static}" = disabled ]; then
  [ ! -f "$base/result-a" ]
  grep -Eq 'ip=198.51.100.1[0-2][0-9] router=198.51.100.2 dns=198.51.100.2' "$base/result-b"
else
  grep -Eq 'ip=192.0.2.1[0-2][0-9] router=192.0.2.2 dns=192.0.2.2' "$base/result-a"
  grep -Eq 'ip=198.51.100.1[0-2][0-9] router=198.51.100.2 dns=198.51.100.2' "$base/result-b"
fi
fingerprint > "$base/after"
cmp "$base/before" "$base/after"
[ "$(ip netns pids "$srv" | wc -l)" -ge 1 ]
printf 'production_config_and_dnsmasq_unchanged=true\n'
printf 'evidence=%s\n' "$base"
