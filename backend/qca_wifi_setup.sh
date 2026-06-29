#!/bin/sh

wireType=`uci -q get wireless.wifi0.type`
if [ "${wireType}" != "qcawificfg80211" ]; then
  echo "not qca wireless"
  exit 0
fi

guestDev=`uci -q get network.guest_dev.name`
if [ "${guestDev}" != "br-guest" ]; then
  uci -q batch <<-EOF >/dev/null
        delete network.guest_dev
        set network.guest_dev="device"
        set network.guest_dev.type="bridge"
        set network.guest_dev.name="br-guest"
        delete network.guest
        set network.guest="interface"
        set network.guest.proto="static"
        set network.guest.device="br-guest"
        set network.guest.ipaddr="192.168.102.1/24"
        commit network
EOF

  #echo "network guest_dev ok"
fi

guestDhcp=`uci -q get dhcp.guest`
if [ "${guestDhcp}" != "dhcp" ]; then
  uci -q batch <<-EOF >/dev/null
        delete dhcp.guest
        set dhcp.guest="dhcp"
        set dhcp.guest.interface="guest"
        set dhcp.guest.start="100"
        set dhcp.guest.limit="150"
        set dhcp.guest.leasetime="1h"
        commit dhcp
EOF

  #echo "dhcp guest ok"
fi

guestFirewall=`uci -q get firewall.guest`
if [ "${guestFirewall}" != "zone" ]; then
  uci -q batch <<-EOF >/dev/null
    delete firewall.guest
    set firewall.guest="zone"
    set firewall.guest.name="guest"
    set firewall.guest.network="guest"
    set firewall.guest.input="REJECT"
    set firewall.guest.output="ACCEPT"
    set firewall.guest.forward="REJECT"
    delete firewall.guest_wan
    set firewall.guest_wan="forwarding"
    set firewall.guest_wan.src="guest"
    set firewall.guest_wan.dest="wan"
    delete firewall.guest_dns
    set firewall.guest_dns="rule"
    set firewall.guest_dns.name="Allow-DNS-Guest"
    set firewall.guest_dns.src="guest"
    set firewall.guest_dns.dest_port="53"
    set firewall.guest_dns.proto="tcp udp"
    set firewall.guest_dns.target="ACCEPT"
    delete firewall.guest_dhcp
    set firewall.guest_dhcp="rule"
    set firewall.guest_dhcp.name="Allow-DHCP-Guest"
    set firewall.guest_dhcp.src="guest"
    set firewall.guest_dhcp.dest_port="67"
    set firewall.guest_dhcp.proto="udp"
    set firewall.guest_dhcp.family="ipv4"
    set firewall.guest_dhcp.target="ACCEPT"
    commit firewall
EOF

  #echo "firewall guest ok"
fi

devDisabled=`uci -q get wireless.wifi0.disabled`
if [ "$devDisabled" = "1" ]; then
uci -q batch <<-EOF >/dev/null
       set wireless.wifi0.channel='auto'
       set wireless.wifi0.hwmode='11beg'
       set wireless.wifi0.disabled='0'
       set wireless.wifi0.country='CN'
       set wireless.wifi0.txpower='100'
       set wireless.wifi0.random_bssid='1'
       set wireless.wifi0.band='2g'
       set wireless.wifi0.htmode='HT40'
       set wireless.wifi0.legacy_rates='0'
       set wireless.wifi2g.disabled='0'
       set wireless.wifi2g.mode='ap'
       set wireless.wifi2g.ssid='iStoreOS-2G'
       set wireless.wifi2g.encryption='none'
       set wireless.wifi2g.key=''
       set wireless.wifi2g.wds=1
       set wireless.wifi2g.isolate=0
       set wireless.wifi2g.hidden=0
       set wireless.wifi2g.ieee80211k=1
       set wireless.wifi2g.bss_transition='1'
       set wireless.wifi2g.sae='0'
EOF

fi

devDisabled=`uci -q get wireless.wifi1.disabled`
if [ "$devDisabled" = "1" ]; then
uci -q batch <<-EOF >/dev/null
       set wireless.wifi1.channel='auto'
       set wireless.wifi1.hwmode='11bea'
       set wireless.wifi1.disabled='0'
       set wireless.wifi1.country='CN'
       set wireless.wifi1.txpower='100'
       set wireless.wifi1.random_bssid='1'
       set wireless.wifi1.band='5g'
       set wireless.wifi1.htmode='HT80'
       set wireless.wifi1.channels='36,40,44,48,149,153,157,161'
       set wireless.wifi1.legacy_rates='0'
       set wireless.wifi5g.disabled='0'
       set wireless.wifi5g.mode='ap'
       set wireless.wifi5g.ssid='iStoreOS-5G'
       set wireless.wifi5g.encryption='none'
       set wireless.wifi5g.key=''
       set wireless.wifi5g.wds=1
       set wireless.wifi5g.isolate=0
       set wireless.wifi5g.hidden=0
       set wireless.wifi5g.ieee80211k=1
       set wireless.wifi5g.bss_transition='1'
       set wireless.wifi5g.sae='0'
EOF

fi

WIFI_DEV=`uci -q get wireless.@wifi-iface[0].device`
uci -q batch <<-EOF >/dev/null
  delete wireless.guest2g
  set wireless.guest2g="wifi-iface"
  set wireless.guest2g.device="${WIFI_DEV}"
  set wireless.guest2g.ifname="wlan01"
  set wireless.guest2g.mode="ap"
  set wireless.guest2g.network="guest"
  set wireless.guest2g.guest="1"
  set wireless.guest2g.ssid="iStoreOS-Guest"
  set wireless.guest2g.encryption="psk2-ccmp"
  set wireless.guest2g.key="goodlife"
  set wireless.guest2g.disabled=1
  set wireless.guest2g.isolate=1
  set wireless.guest2g.wds=1
EOF

#echo "wireless guest2g ok"

WIFI_DEV=`uci -q get wireless.@wifi-iface[1].device`
uci -q batch <<-EOF >/dev/null
  delete wireless.guest5g
  set wireless.guest5g="wifi-iface"
  set wireless.guest5g.device="${WIFI_DEV}"
  set wireless.guest5g.ifname="wlan11"
  set wireless.guest5g.mode="ap"
  set wireless.guest5g.network="guest"
  set wireless.guest5g.guest="1"
  set wireless.guest5g.ssid="iStoreOS-5G-Guest"
  set wireless.guest5g.encryption="psk2+ccmp"
  set wireless.guest5g.key="goodlife"
  set wireless.guest5g.disabled=1
  set wireless.guest5g.isolate=1
  set wireless.guest5g.wds=1
  commit wireless
EOF

#echo "wireless guest5g ok"
