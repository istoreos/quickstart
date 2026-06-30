# QuickStart API Summary

Base path through LuCI: `/cgi-bin/luci/istore`

Use relative paths with `scripts/qsctl.go`. Example: `/system/status/`.

Response envelope:

```json
{"success":0,"error":"","scope":"","result":{}}
```

`success == 0` means OK. Any non-zero `success` is an API failure.

## Read Endpoints

System:

- `GET /system/version/` - router model, kernel, firmware, QuickStart version
- `GET /system/time/` - uptime and local time
- `GET /system/status/` - overall system status
- `GET /system/cpu/status/` - CPU usage
- `GET /system/cpu/temperature/` - CPU temperature
- `GET /system/memery/status/` - memory status
- `GET /system/check-update/` - firmware/update check
- `GET /system/module-settings/` - module display settings

Network:

- `GET /network/statistics/` - traffic statistics
- `GET /network/status/` - network status
- `GET /network/interface/status/` - interface status
- `GET /network/interface/config/` - interface configuration
- `GET /network/device/list/` - connected devices
- `GET /network/port/list/` - physical port list

Wireless:

- `GET /wireless/list-iface/` - Wi-Fi interface list

NAS and storage:

- `GET /nas/disk/status/` - disk status
- `GET /nas/service/status/` - NAS service status
- `GET /nas/webdav/status` - WebDAV status
- `GET /nas/sandbox/` - sandbox status
- `GET /nas/sandbox/disks/` - sandbox disk list
- `GET /guide/docker/status/` - Docker status
- `GET /guide/docker/partition/list/` - Docker target partitions
- `GET /guide/download/partition/list/` - download-service target partitions
- `GET /guide/download-service/status/` - aria2/qBittorrent/Transmission status
- `GET /guide/global-folders/` - global folder list

RAID and SMART:

- `GET /raid/list/` - RAID arrays
- `GET /raid/create/list/` - candidate devices for RAID creation
- `POST /raid/detail/` - RAID detail for a request body

Sharing:

- `GET /share/user/list/` - share users
- `GET /share/service/list/` - share services
- `GET /share/protocol/webdav/` - WebDAV protocol config
- `GET /share/protocol/samba/` - Samba protocol config
- `GET /share/protocol/globals/` - global share protocol config

Guide and DDNS:

- `GET /guide/need/setup/` - setup requirement state
- `GET /guide/pppoe/` - PPPoE config/status
- `GET /guide/lan/` - LAN settings
- `GET /guide/client-mode/` - client mode settings
- `GET /guide/gateway-router/` - gateway/router mode settings
- `GET /guide/dns-config/` - DNS config
- `GET /guide/soft-source/` - software source config
- `GET /guide/soft-source/list` - software source list
- `GET /guide/ddns/` - DDNS config
- `GET /guide/ddnsto/config/` - DDNSTO config

Apps, files, LAN control:

- `POST /app/check/` - check app metadata/status
- `GET /app/install-list/` - installed app list
- `POST /file/basic/list/` - list basic files
- `GET /lanctrl/globalConfigs/` - LAN control global config
- `GET /lanctrl/listDevices/` - LAN devices
- `GET /lanctrl/listStaticDevices/` - static devices
- `GET /lanctrl/listSpeedLimitedDevices/` - speed limited devices
- `GET /lanctrl/speedsForDevices/` - device speeds
- `POST /lanctrl/speedsForOneDevice/` - one device speed

## Write Endpoints

Confirm before use:

- `POST /system/auto-check-update/`
- `POST /system/module-settings/`
- `POST /network/checkPublicNet/`
- `POST /network/homebox/enable`
- `POST /guide/docker/transfer/`
- `POST /guide/aria2/init/`
- `POST /guide/qbittorrent/init/`
- `POST /guide/transmission/init/`
- `POST /guide/pppoe/`
- `POST /guide/lan/`
- `POST /guide/client-mode/`
- `POST /guide/gateway-router/`
- `POST /guide/dns-config/`
- `POST /guide/soft-source/`
- `POST /guide/ddns/`
- `POST /guide/ddnsto/`
- `POST /guide/ddnsto/address/`
- `POST /nas/samba/create`
- `POST /nas/webdav/create`
- `POST /nas/linkease/enable`
- `POST /share/user/create/`
- `POST /share/user/delete/`
- `POST /share/user/update/`
- `POST /share/service/create/`
- `POST /share/service/delete/`
- `POST /share/service/update/`
- `POST /share/protocol/webdav/`
- `POST /share/protocol/samba/`
- `POST /share/protocol/globals/`
- `POST /wireless/enable-iface/`
- `POST /wireless/set-device-power/`
- `POST /wireless/edit-iface/`
- `POST /wireless/setup/`
- `POST /lanctrl/enableSpeedLimit/`
- `POST /lanctrl/enableFloatGateway/`
- `POST /lanctrl/staticDeviceConfig/`
- `POST /lanctrl/speedLimitConfig/`
- `POST /lanctrl/dhcpTagsConfig/`
- `POST /lanctrl/dhcpGatewayConfig/`

## Dangerous Endpoints

Require explicit user confirmation immediately before calling:

- `POST /system/reboot/` - reboots the router
- `POST /system/poweroff/` - powers off the router if supported
- `POST /system/setPassword/` - changes system password
- `POST /network/interface/config/` - can break network access
- `POST /nas/disk/init` - initializes disk
- `POST /nas/disk/initrest` - disk initialization follow-up
- `POST /nas/disk/partition/format` - formats partition
- `POST /nas/disk/partition/mount` - changes mount state
- `POST /nas/sandbox/commit/` - commits sandbox changes
- `POST /nas/sandbox/reset/` - resets sandbox
- `POST /raid/create/` - creates RAID array
- `POST /raid/delete/` - deletes RAID array
- `POST /raid/add/` - adds RAID member
- `POST /raid/remove/` - removes RAID member
- `POST /raid/recover/` - starts RAID recovery
- `POST /app/install/` - installs an app/package
