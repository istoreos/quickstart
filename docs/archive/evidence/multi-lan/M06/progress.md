# M06 execution checkpoint

Status: passed after three optimization rounds. Final device and compatibility evidence is in result.json.

Implemented in Quickstart: local UCI engine registry; server-only endpoint selection; network/instance/interface/ifindex health validation; old main-engine process verification; scoped native provider copies; full stored-policy revision/source echo; explicit connection closure for original one-request HTTP servers.

Implemented in NetPolicy: network/instance CLI identity and epoch health; instance/request-bound pending plans (64); request-bound committed replay; recovery-required latch; exclusive data/interface locks and bridge-child overlap checks; loopback listener before BPF; SIGTERM shutdown; own named netlink-filter recovery; named procd configuration and resource validation; revision/source verification compatibility; correct HTTP Connection: close response.

Verified: Go full suite and targeted race tests; Rust 48 tests; pure production-init validation. Real ARM64 and x86 core two-instance lifecycle tests passed, including owner retention, foreign-plan rejection, same MAC distinct rates, separate BPF FDs, hard restart, single stop/start and interface recreation. x86 netlink plus actual Quickstart provider passed after round 1.

Round 1 fixed Go dropping required revision/source fields, adding strict legacy-compatible HTTP verification.
Round 2 fixed TCP half-close reuse (HTTP server served one request without close header), plus corrected the temporary old-Go source overlay generator to preserve string literals while renaming identifiers.

Completed: final ARM64+actual-provider and x86 netlink+actual-provider smoke, plus scripts/ops/multi-lan-netpolicy-compat.py using pinned old/new clients and engines; see result.json and corresponding JSON evidence. No production UCI configuration changed by these scripts; task services and empty bridges are removed in finally blocks. Test host .1 has not been mutated.

Build paths:
- NetPolicy: /var/cache/build-cache/users/911/projects/quickstart-netpolicy-15189b090927/cargo-target/{aarch64,x86_64}-unknown-linux-musl/release/quickstart-netpolicy
- Legacy engine (baseline 54b5338): /var/cache/build-cache/users/911/projects/quickstart-netpolicy-15189b090927/work/multi-lan/legacy-artifacts/quickstart-netpolicy
- Quickstart test executables and old-client overlay: /var/cache/build-cache/users/911/projects/quickstart-f0ca7c908f5f/work/multi-lan/M06/

The legacy engine uses a separate SSD target directory; sharing a target directory with an archived source tree produced an invalid stale binary and was discarded before compatibility testing. Assert old --help has no --network-id. The original provider comes from Quickstart 66589b22, injected as a test-only Go overlay; it is not copied into product source.

Final acceptance also fixed config_foreach IFS handling and passed a real VM reboot with production init/netifd bindings. The temporary bootstrap was restored; reboot.json confirms original network/DHCP/firewall hashes.
