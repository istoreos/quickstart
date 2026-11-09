#!/bin/sh
# Called only by the authorized two-host packet harness; no root-netns links/config.
set -eu
base=$1
srv=$2
a=$3
b=$4
wan=$5
attachment=${6:-ethernet}
mkdir -p "$base/config" "$base/module" "$base/uci-state"
printf 'M07 isolated netns' > "$base/owner"
for ns in "$srv" "$a" "$b" "$wan"; do ip netns add "$ns"; ip -n "$ns" link set lo up; done
if [ "$attachment" = vlan ]; then
 ip -n "$srv" link add trunk type veth peer name swtrunk
 ip -n "$srv" link set swtrunk netns "$wan"
 ip -n "$srv" link set trunk up
 ip -n "$wan" link set swtrunk up
fi
for side in a b w; do
 client=$a; prefix=192.0.2; v6=fd71; iface=lana
 case "$side" in b) client=$b; prefix=198.51.100; v6=fd72; iface=lanb;; w) client=$wan; prefix=203.0.113; v6=fd73; iface=wan0;; esac
 if [ "$attachment" = vlan ] && [ "$side" != w ]; then
  vlan=10; [ "$side" = a ] || vlan=20
  ip -n "$srv" link add link trunk name "$iface" type vlan id "$vlan"
  ip -n "$wan" link add "switch$side" type bridge
  ip -n "$wan" link add link swtrunk name "vlan$side" type vlan id "$vlan"
  ip -n "$wan" link set "vlan$side" master "switch$side"
  ip -n "$wan" link set "vlan$side" up
  ip -n "$srv" link add "port$side" type veth peer name "client$side"
  ip -n "$srv" link set "port$side" netns "$wan"
  ip -n "$wan" link set "port$side" master "switch$side"
  ip -n "$wan" link set "port$side" up
  ip -n "$wan" link set "switch$side" up
 elif [ "$attachment" = bridge ] && [ "$side" != w ]; then
  ip -n "$srv" link add "$iface" type bridge
  ip -n "$srv" link add "port$side" type veth peer name "client$side"
  ip -n "$srv" link set "port$side" master "$iface"
  ip -n "$srv" link set "port$side" up
 else
  ip -n "$srv" link add "$iface" type veth peer name "client$side"
 fi
 ip -n "$srv" link set "client$side" netns "$client"
 ip -n "$srv" addr add "$prefix.1/24" dev "$iface"
 ip -n "$srv" -6 addr add "$v6::1/64" dev "$iface" nodad
 ip -n "$srv" link set "$iface" up
 host=55
 if [ "$side" = w ]; then host=2; else ip -n "$client" link set "client$side" address 02:00:00:00:00:55; fi
 ip -n "$client" addr add "$prefix.$host/24" dev "client$side"
 ip -n "$client" -6 addr add "$v6::$host/64" dev "client$side" nodad
 ip -n "$client" link set "client$side" up
 ip -n "$client" route add default via "$prefix.1"
 ip -n "$client" -6 route add default via "$v6::1"
done
ip netns exec "$srv" sysctl -qw net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1
cat > "$base/config/firewall" <<EOF
config defaults 'main'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
 option auto_includes '0'
config zone 'local'
 option name 'shared'
 list device 'lana'
 list device 'lanb'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
config zone 'outside'
 option name 'wan'
 list device 'wan0'
 option input 'ACCEPT'
 option output 'ACCEPT'
 option forward 'ACCEPT'
EOF
for side in a b; do
 iface=lana; port=19471; [ "$side" = a ] || { iface=lanb; port=19472; }
 cat >> "$base/config/quickstart-netpolicy" <<EOF
config quickstart-netpolicy '$side'
 option enabled '1'
 option network 'test_$side'
 option interface '$iface'
 option port '$port'
 option data_dir '$base/$side'
EOF
 # Root firmware's compiler, with only fixture/config/state paths redirected.
 # Ubus remains read-only; explicit zone devices are the namespace devices.
 ip netns exec "$srv" "$base/quickstart-netpolicy" --iface "$iface" --port "$port" --data-dir "$base/$side" --network-id "test_$side" --instance-id "$side" --enable-traffic --traffic-additional-subnets '192.0.2.0/24,198.51.100.0/24,fd71::/64,fd72::/64' > "$base/engine-$side.log" 2>&1 &
 echo $! > "$base/engine-$side.pid"
done
sed "s|uci.cursor()|uci.cursor('$base/config', '$base/uci-state')|;s|/var/run/fw4.state|$base/fw4.state|g" /usr/share/ucode/fw4.uc > "$base/module/fw4.uc"
cat > "$base/load-firewall" <<EOF
#!/bin/sh
set -eu
ACTION=print utpl -L '$base/module' -S /usr/share/firewall4/main.uc > '$base/rules.nft'
nft delete table inet fw4 2>/dev/null || true
nft -f '$base/rules.nft'
EOF
ip netns exec "$wan" iperf3 -s -D --pidfile "$base/iperf-wan.pid" --logfile "$base/iperf-wan.log"
ip netns exec "$b" iperf3 -s -D --pidfile "$base/iperf-b.pid" --logfile "$base/iperf-b.log"
printf 'namespace_setup_complete=true\n'
