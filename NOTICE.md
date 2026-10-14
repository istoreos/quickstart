# Quickstart notices

Quickstart is distributed under the MIT License in [`LICENSE`](LICENSE).

## Device scene artwork

The 30 bitmap files in
`web/public/luci-static/quickstart/device-icons/` are project-original,
AI-assisted generic device illustrations. They were generated with OpenAI's
image generation tool on 2026-09-24 and 2026-09-25. The common prompt required
an original rounded geometric device pictogram suitable for 32–48 px use, a
lavender/indigo palette with a small mint accent, a transparent background,
and no words, letters, logos, watermarks, trademarked elements, or
brand-specific product silhouettes. Each device shape used its own scene
description.

The artwork represents generic categories only. Manufacturer names such as
ASUS are rendered as text by the application and are not embedded in the
artwork. No artwork or logo from luci-app-oui, luci-app-bandix, a device
manufacturer, or another third party is included.

The machine-readable asset list and review gate are in
`web/public/luci-static/quickstart/device-icons/manifest.json`; the complete
generation and review record is in `docs/device-scene-icons.md`. Generative
creation does not guarantee non-infringement, so the manifest intentionally
requires product/legal approval before a public release.

## Reference implementations

The LAN device-management design was informed by observable behavior in
luci-app-oui, luci-app-bandix, and floatip. Quickstart's implementation and
artwork were produced independently; code and branded assets from those
projects were not copied into this release. Relevant source-license findings
and integration boundaries are recorded in
`docs/lan-device-management-reference-research.md`.

