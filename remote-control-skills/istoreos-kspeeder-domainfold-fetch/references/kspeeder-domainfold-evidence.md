# DomainFold / gh.linkease.net: evidence in `kspeeder` repo

## 1) Origin URL → entry host alias URL (github.com → gh.linkease.net)

- Default route mapping includes GitHub:
  - `/gh` has `OriginURL: "https://github.com"`.
  - Evidence: `kspeeder/domainfold/routes.go` (`DefaultRoutes`).
- Remap algorithm:
  - Map `origin host + base path` → route prefix (longest base path wins).
  - Build alias host from prefix: `/gh` + `AliasSuffix` → `gh.<suffix>`.
  - Output forces `https` and sets `Host` to `aliasHost:tlsPort`.
  - Evidence: `kspeeder/domainfold/url_mapper.go` (`remapOriginURL`) and `kspeeder/docs/domainfold-originurl-remap.md`.

## 2) Why gh.linkease.net requests are handled by the KSpeeder TLS port

`cmd/multi` TLS port uses host-based dispatch:

- If host matches `*.linkease.net` (excluding `registry.linkease.net`) then route into DomainFold handler.
- Evidence: `kspeeder/cmd/multi/proxy_handlers.go` (`buildTLSHandler`).

## 3) How to avoid DNS for gh.linkease.net (admin proxy /gh/...)

`cmd/multi` admin port registers per-prefix routes like `/gh/` and proxies them to the local TLS listener:

- Route registration for every `cfg.Routes` prefix:
  - Evidence: `kspeeder/cmd/multi/domainfold_admin_proxy_routes.go`.
- Handler behavior:
  - Takes `/gh/...` as PathHub-style path.
  - Uses mapper to compute desired alias host.
  - Reverse-proxies to `https://127.0.0.1:<tlsPort>` with `ServerName=<desiredHost>` and `req.Host=<desiredHost>`.
  - Evidence: `kspeeder/cmd/multi/domainfold_admin_proxy.go`.

This is the most reliable way to “curl/wget GitHub through KSpeeder” on the same box (or inside LAN), because it does not require `gh.linkease.net` DNS.

## 4) Remap API endpoint (for skills/scripts)

- `POST /api/domainfold/remap` with JSON body `{"url":"<origin>"}` returns JSON with:
  - `output`: entry URL like `https://gh.linkease.net:<tlsPort>/...`
  - `admin_path`: PathHub-style `/gh/...` path usable on admin port
  - Evidence: `kspeeder/cmd/multi/domainfold_remap_api.go`.
