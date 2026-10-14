# Device Management

This context describes how Quickstart identifies LAN devices, presents their state, and applies per-device network policy.

## Language

**Device**:
A LAN client represented as one product entity even when it has multiple observations or addresses.
_Avoid_: Host, client row, IP device

**Device Identity**:
The stable reference used to recognize the same Device across observations; an address or display name alone is not an identity.
_Avoid_: IP identity, hostname identity

**Observation**:
Evidence that a Device or address was seen from a local network source at a particular time.
_Avoid_: Device record

**Current Address**:
An IPv4 or IPv6 address presently owned by a Device according to current observations.
_Avoid_: Active device

**Historical Address**:
An address previously owned by a Device during the retained history window but not currently owned by it.
_Avoid_: Old device

**Online Device**:
A Device with sufficiently recent direct presence evidence on the LAN.
_Avoid_: Active device

**Offline Device**:
A previously observed Device without sufficiently recent direct presence evidence on the LAN.
_Avoid_: Disabled device, blocked device

**Capability**:
A device-management function that the current router installation and configuration can actually provide.
_Avoid_: Feature flag, plugin state

**Device Policy**:
The desired static addressing, gateway, speed, or network-access rules associated with a Device.
_Avoid_: Device state, device config

**Telemetry**:
Time-sensitive traffic measurements attributed to a Device, distinct from its identity and Device Policy.
_Avoid_: Device status

**Manufacturer**:
The raw organization name associated with a Device observation, usually from an OUI registry; it is evidence and may not be the consumer-facing maker.
_Avoid_: Brand, device type

**Brand**:
A normalized, consumer-facing maker name that can be shown as factual text when supported by Manufacturer or model evidence.
_Avoid_: Manufacturer, logo

**Device Category**:
A generic product form such as computer, phone, television, camera, or network device used to select the Device's visual shape; it is not an exact model claim.
_Avoid_: Brand, model, scene logo

**Device Classification**:
The current Brand and Device Category conclusion for a Device, together with its evidence source and confidence.
_Avoid_: Device Identity, Manufacturer

**Classification Source**:
The strongest evidence that produced a Device Classification: manual correction, model, hostname, manufacturer default, or fallback.
_Avoid_: Data source health

**Manual Classification**:
A user-confirmed Device Category that takes precedence over automatic classification until the user resets it.
_Avoid_: Device name, Device Policy

**Device Alias**:
A user-owned display name for a Device that remains independent of DHCP hostname and Address Reservation.
_Avoid_: Hostname, DHCP name

**DHCP Hostname**:
An optional protocol-compatible ASCII host label advertised or assigned through DHCP configuration; it may affect network name resolution and is never the user's Unicode-capable Device Alias.
_Avoid_: Device Alias, display name

**Device Profile**:
The durable, user-owned metadata for a Device Identity: Device Alias, explicit Brand or Device Category corrections, and Icon Preference. It contains neither Device Group membership nor network policy.
_Avoid_: Device Group, Device Policy, DHCP host

**Icon Preference**:
The user's choice to follow automatic Brand-and-Category visual resolution or pin a specific approved icon for a Device.
_Avoid_: Manufacturer logo, Device Category

**Desired Policy**:
The network outcome the user has asked Quickstart to maintain for a Device or Device Group.
_Avoid_: Applied Configuration, Observed Effect

**Applied Configuration**:
The router configuration that Quickstart has successfully written and verified as representing a Desired Policy.
_Avoid_: Desired Policy, Observed Effect

**Observed Effect**:
Bounded evidence about what happened after configuration was applied; it may be pending or unverifiable when the endpoint cannot be observed.
_Avoid_: Save result, Applied Configuration, Online Device

**Address Reservation**:
A DHCP rule that reserves an IPv4 or IPv6 address for a Device Identity; it is distinct from the Device's currently observed address.
_Avoid_: Current Address, static device

**Internet Path**:
The user-facing choice of which Gateway Target a Device should receive, with DNS following the same path unless a future product decision explicitly changes that rule.
_Avoid_: DHCP tag, route label

**Gateway Target**:
A selectable next-hop destination such as this router, an upstream router, a bypass router, or a Floating Gateway.
_Avoid_: Device, node role

**DHCP Policy Tag**:
An internal dnsmasq tag that binds a DHCP host rule to DHCP options such as gateway and DNS server; it is an implementation mechanism, not a user organization label.
_Avoid_: Device tag, Device Group

**Floating Gateway**:
A virtual IPv4 address that can move between a preferred node and a fallback node according to health checks; DHCP must separately advertise it as a Gateway Target.
_Avoid_: Floating route, DHCP gateway

**Gateway Node**:
A router participating in a Floating Gateway pair, described to users by its real role (preferred service node or takeover node) rather than the plugin's internal role value.
_Avoid_: Device, client

**Device Group**:
A user-facing collection used to organize Devices and assign shared policy; it must not reuse the DHCP Policy Tag name or storage semantics.
_Avoid_: DHCP Policy Tag

**Internet Access Policy**:
The Desired Policy that determines whether a Device may access the network, independently of any speed-limiting capability.
_Avoid_: Online Device, Speed Policy

**Speed Policy**:
The optional Desired Policy that limits upload or download throughput for a Device or Device Group.
_Avoid_: Internet Access Policy, Traffic Quota

**Usage Policy**:
The schedules, quotas, Internet Access Policy, and Speed Policy associated with a Device or Device Group.
_Avoid_: Telemetry, online state

**Router Context**:
The router's evidence-backed role and DHCP allocation authority for one selected LAN, used to explain whether Internet Path changes are locally editable.
_Avoid_: Global bypass-router flag, Capability

**Policy Precedence**:
The deterministic rule that explains which global, group, and per-device policy wins when more than one policy applies.
_Avoid_: Rule order, UCI order

**Traffic Insights**:
Bounded, locally persisted hourly/daily/monthly usage derived from Device Telemetry; it is separate from real-time speed and optional connection/DNS adapters.
_Avoid_: Live speed, Bandix data

**Traffic Quota**:
A daily, weekly, or monthly byte allowance that can notify or temporarily block a Device; an enabled quota is an explicit background telemetry consumer.
_Avoid_: Speed limit, schedule

**Audit Event**:
A bounded, privacy-safe explanation that a device lifecycle, policy, conflict, quota, probe, or observed gateway-holder change occurred.
_Avoid_: Debug log, DNS history

**Policy Bundle**:
A versioned export of Device Groups, per-device group exceptions, and Traffic Quotas that must be previewed and version-checked before restoration.
_Avoid_: Full router backup, UCI archive
