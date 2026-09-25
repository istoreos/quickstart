# Use task-scoped network transactions

Status: accepted

Quickstart treats Device Profile, Network & Internet, and Usage Restrictions as three independent write transactions. A single cross-system transaction spanning profile storage, dnsmasq, firewall, eqos, schedules, and quotas is not honestly atomic on OpenWrt; each task therefore plans, validates, snapshots, applies, verifies, and either compensates or reports that recovery is required.
