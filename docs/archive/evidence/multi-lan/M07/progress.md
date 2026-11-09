# M07 execution checkpoint

Status: passed after three optimization rounds. Both final rebuilt business probes passed; see result.json and final-business-*.json.

Task tree: scope native/legacy rate providers and preferences; bind access rules to network/ingress/owner; sample each engine using WAN-only counters and runtime epoch; isolate history/quota documents and background contexts; verify real dual-stack packets, persistence and failure boundaries.

Acceptance: same MAC independent rates, exact shared-zone access, no internal WAN billing, no cross-LAN history/quota leaks, restart-safe exhausted quota, independent manual/quota ownership, period/cancel restoration, honest unsupported path/offload states, default LAN compatibility.

Implemented in Quickstart: immutable provider scope; scoped provider preferences; default-only eQoS/Bandix; network/ingress-bound UCI and prepend nft rules; independent quota rule ownership (also retained in deferred batches); scoped legacy reads; selected-scope settings versions and snapshots; engine-epoch WAN samples with stale/unknown/warmup states; aggregate row targets and per-network health; per-network history/quota documents with original default path; one quota worker resolving fresh scopes and loading saved networks at startup; shared lock ordering; critical usage checkpoint before firewall changes; immediate durable quota transitions; path/offload safety checks.

Implemented in NetPolicy: shared runtime epoch in health and traffic responses; last successful kernel sample timestamp; unsuccessful kernel reads do not publish a fresh sample; local IPv4/IPv6 topology for cross-LAN classification; bounded prefix capacity and fail-closed refresh. Device admission remains interface-scoped.

Completed evidence: Ethernet and bridge full windows on ARM64 and x86, each with four 100 Mbit/s baseline windows and eight rate windows (30 seconds plus 5 seconds warmup). All 32 limited windows fall within the fixed 25% tolerance; unlimited baselines exceed 3x each requested limit. Real shared-zone IPv4/IPv6 blocks, unaffected other LAN, internal WAN exclusion, hard engine restart and manual pause preservation passed. Bridge probes also pass injected next-period clock with real packet restoration; the host clock is untouched.

Optimization 1: quota setup checks sampling readiness and preserves a zero warm-up baseline so the first traffic burst is counted. Unusable sampling cannot be reported as enabled quota.
Optimization 2: do not claim forwarding enforcement for a one-arm/asymmetric or unproven default-route path; native offload risk/unknown never reports verified. Added config-based software/hardware offload and path tests.
Optimization 3: final ARM harness attempted a counter assertion while the device was still warming up. Wait for the actual MAC sample (maximum 30 seconds), exercising the client path; never treat missing data as measured zero. Final probes use rebuilt binaries and record artifact SHA256.

No production network, DHCP or firewall configuration was changed by packet tests; all packet links and loaded firewall tables were inside owned namespaces. Original production hashes matched after cleanup. Read-only host 192.168.30.1 was not mutated. M08/M09/M10, final upgrade/rollback matrix, capacities and 24-hour soak remain pending.
