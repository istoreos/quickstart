#!/bin/sh
uci set network.lan.ipaddr=192.168.9.118
uci set network.lan.netmask=255.255.255.0
uci set network.lan.gateway=192.168.9.1
uci commit network

ubus call service event '{"type":"config.change", "data":{"package":"network"}}'
