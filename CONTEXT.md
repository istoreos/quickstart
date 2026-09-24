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
