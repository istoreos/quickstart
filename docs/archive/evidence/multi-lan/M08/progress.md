# M08 execution checkpoint

Status: passed after one optimization round. Both final-auxiliary and quota-regression passed on both hosts; see result.json.

Task tree: scoped group members/defaults/exceptions/references; fresh background scheduling and immutable per-network execution state; atomic native batches and bounded reloads; scoped ledger cleanup and adoption receipts; audit/webhook identity; source-bound management probes; versioned scoped imports and recovery; installation versus per-network connection; complete endpoint classification.

Initial real tests passed on ARM64 and x86: IPv4/IPv6/link-local HTTP HEAD probes with recorded source addresses; foreign target rejection; selected network connection with other engine PID unchanged; real minute-based background block and restore after page traffic stops; original network/DHCP/firewall hashes unchanged.

Implemented: one scheduler with scoped runtime children and fresh bindings; scoped group versions/member/target validation; per-owner native batch replacement with one plan/apply/verify; scoped native/manual/quota ledger with protected quota-owned rows; same-MAC DHCP cleanup isolation; scoped migration receipts; process diagnostics explicitly labeled shared; network-qualified audit IDs and per-network restart dedup; source/device-bound management clients; schema 2 additional-LAN bundles rejected by old readers; target-bound import previews; quota and firewall import recovery; component installation metadata, per-network activation, versioned previews and true shared-service impact; older service upgrade requirement; activation rollback without stopping other LANs. Production init exposes its scope contract marker.

Optimization 1: fault injection showed the import/ordinary quota save paths needed the same transactional quota restoration as batches. Single-quota writes now use the batch implementation. Failed imports restore removed quota blocks, durable usage, group documents and configuration snapshots. The test lock seam uses the actual configured directory, matching the production directory invariant.

Local verification: Go full suite and targeted race checks passed; production init contract tests passed. API coverage classifies all 67 current routes (including 66 baseline routes) as scoped, aggregate read, shared component dependency or explicitly default-only compatibility paths. Webhook tests use a controlled localhost HTTP sink only.

Final rebuilt packet regressions passed and formal M08 result is recorded. M09 production UI integration and M10 upgrade/capacity/24-hour acceptance remain incomplete.
