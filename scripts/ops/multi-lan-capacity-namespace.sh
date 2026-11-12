#!/bin/sh
# All mutations are restricted to owned namespaces and a disposable config tree.
set -eu
base=$1
router=$2
wan=$3
prefix=$4
count=$5
full_application=${6:-0}
case "$base:$router:$wan:$prefix:$count" in /tmp/quickstart-multi-lan/m10-*:qsm10-*:qsm10-*:qc*:1|/tmp/quickstart-multi-lan/m10-*:qsm10-*:qsm10-*:qc*:2|/tmp/quickstart-multi-lan/m10-*:qsm10-*:qsm10-*:qc*:4) ;; *) exit 2;; esac
mkdir -p "$base/config" "$base/module" "$base/uci-state"
printf 'M10 isolated capacity' > "$base/owner"
ip netns add "$router"
ip netns add "$wan"
ip -n "$router" link set lo up
ip -n "$wan" link set lo up
ip -n "$router" link add "${prefix}w" type veth peer name "${prefix}x"
ip -n "$router" link set "${prefix}x" netns "$wan"
ip -n "$router" addr add 203.0.113.1/24 dev "${prefix}w"
ip -n "$wan" addr add 203.0.113.2/24 dev "${prefix}x"
ip -n "$router" -6 addr add fd90::1/64 dev "${prefix}w" nodad
ip -n "$wan" -6 addr add fd90::2/64 dev "${prefix}x" nodad
ip -n "$router" link set "${prefix}w" up
ip -n "$wan" link set "${prefix}x" up
ip -n "$wan" route add default via 203.0.113.1
ip -n "$wan" -6 route add default via fd90::1
ip netns exec "$router" sysctl -qw net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1
cat > "$base/config/firewall" <<EOF
config defaults 'main'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
 option auto_includes '0'
config zone 'locals'
 option name 'shared'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
EOF
printf '[' > "$base/scopes.json"
i=1
while [ "$i" -le "$count" ]; do
 iface="${prefix}$i"
 client="$router-c$i"
 subnet=$((200+i))
 v6=$((90+i))
 port=$((19500+i))
 ip netns add "$client"
 ip -n "$client" link set lo up
 ip -n "$router" link add "$iface" type bridge
 ip -n "$router" link add "${prefix}p$i" type veth peer name "${prefix}c$i"
 ip -n "$router" link set "${prefix}p$i" master "$iface"
 ip -n "$router" link set "${prefix}p$i" up
 ip -n "$router" link set "${prefix}c$i" netns "$client"
 ip -n "$router" addr add "10.$subnet.0.1/23" dev "$iface"
 ip -n "$router" -6 addr add "fd$v6::1/64" dev "$iface" nodad
 ip -n "$router" link set "$iface" up
 ip -n "$client" link set "${prefix}c$i" address 02:00:00:00:00:01
 ip -n "$client" addr add "10.$subnet.0.3/23" dev "${prefix}c$i"
 ip -n "$client" -6 addr add "fd$v6::3/64" dev "${prefix}c$i" nodad
 ip -n "$client" link set "${prefix}c$i" up
 ip -n "$client" route add default via "10.$subnet.0.1"
 ip -n "$client" -6 route add default via "fd$v6::1"
 j=1
 while [ "$j" -le 256 ]; do
  mac=$(printf '02:00:00:00:%02x:%02x' "$((j/256))" "$((j%256))")
  ip -n "$router" neigh replace "10.$subnet.$((j/250)).$((j%250+2))" lladdr "$mac" nud permanent dev "$iface"
  ip -n "$router" -6 neigh replace "fd$v6::$(printf '%x' "$((j+2))")" lladdr "$mac" nud permanent dev "$iface"
  j=$((j+1))
 done
 printf " list device '%s'\n" "$iface" >> "$base/config/firewall"
 ifindex=$(ip netns exec "$router" cat "/sys/class/net/$iface/ifindex")
 [ "$i" -eq 1 ] || printf ',' >> "$base/scopes.json"
 printf '{"networkId":"capacity%s","displayName":"capacity%s","device":"%s","state":"up","ipv4Prefixes":["10.%s.0.1/23"],"ipv6Prefixes":["fd%s::1/64"],"firewallZones":["shared"],"dhcpBindings":[],"version":"m10-%s"}' "$i" "$i" "$iface" "$subnet" "$v6" "$ifindex" >> "$base/scopes.json"
 cat >> "$base/config/quickstart-netpolicy" <<EOF
config quickstart-netpolicy 'capacity$i'
 option enabled '1'
 option network 'capacity$i'
 option interface '$iface'
 option port '$port'
 option data_dir '$base/engine-$i'
EOF
 ip netns exec "$router" "$base/quickstart-netpolicy" --iface "$iface" --port "$port" --data-dir "$base/engine-$i" --network-id "capacity$i" --instance-id "capacity$i" --enable-traffic --traffic-additional-subnets '10.201.0.0/23,10.202.0.0/23,10.203.0.0/23,10.204.0.0/23,fd91::/64,fd92::/64,fd93::/64,fd94::/64' > "$base/engine-$i.log" 2>&1 &
 echo $! > "$base/engine-$i.pid"
 router_mac=$(ip netns exec "$router" cat "/sys/class/net/$iface/address")
 ip netns exec "$client" env QUICKSTART_MULTI_LAN_CAPACITY_BASE="$base" QUICKSTART_MULTI_LAN_CAPACITY_IFACE="${prefix}c$i" QUICKSTART_MULTI_LAN_CAPACITY_ROUTER_MAC="$router_mac" QUICKSTART_MULTI_LAN_CAPACITY_SUBNET="$subnet" "$base/service.test" -test.run '^TestMultiLANCapacityPackets$' -test.timeout 26h > "$base/packets-$i.log" 2>&1 &
 echo $! > "$base/packets-$i.pid"
 i=$((i+1))
done
printf ']' >> "$base/scopes.json"
cat >> "$base/config/firewall" <<EOF
config zone 'outside'
 option name 'wan'
 list device '${prefix}w'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
EOF
sed "s|uci.cursor()|uci.cursor('$base/config', '$base/uci-state')|;s|/var/run/fw4.state|$base/fw4.state|g" /usr/share/ucode/fw4.uc > "$base/module/fw4.uc"
cat > "$base/load-firewall" <<EOF
#!/bin/sh
set -eu
ACTION=print utpl -L '$base/module' -S /usr/share/firewall4/main.uc > '$base/rules.nft'
nft delete table inet fw4 2>/dev/null || true
nft -f '$base/rules.nft'
EOF
ip netns exec "$router" sh "$base/load-firewall"
ip netns exec "$router" env QUICKSTART_MULTI_LAN_CAPACITY_BASE="$base" QUICKSTART_MULTI_LAN_FULL_APPLICATION="$full_application" "$base/service.test" -test.run '^TestMultiLANCapacityBackend$' -test.timeout 26h > "$base/backend.log" 2>&1 &
echo $! > "$base/backend.pid"

if [ "$full_application" = 1 ]; then
 attempts=0
 while [ ! -f "$base/core.ready" ]; do attempts=$((attempts+1)); [ "$attempts" -le 60 ] || exit 1; sleep 1; done
 mv "$base/backend.pid" "$base/core.pid"
 printf 'production executable' > "$base/full-application"
 ip netns exec "$router" env QUICKSTART_MULTI_LAN_CAPACITY_BASE="$base" "$base/service.test" -test.run '^TestMultiLANFullApplication$' -test.timeout 26h > "$base/application.log" 2>&1 &
 echo $! > "$base/application-launcher.pid"
fi
