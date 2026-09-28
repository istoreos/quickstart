# M64 发布候选状态

日期：2026-09-28

M64 只有在最终候选完成连续 24 小时拓扑采样后才能关闭。采样必须同时验证：A/B Quickstart 未重启且 API 为 HTTP 200、浮动地址仍由 B 持有、C 经 B 上网、D 经 A 上网、两端 RSS/线程/FD 有界。

本目录的 `soak.tsv` 由 `lan-device-topology-soak.sh` 每 300 秒追加一次；完成后必须由 `analyze-lan-device-topology-soak.sh` 以默认 86400 秒门槛验收。未达到持续时间时分析器必须失败，不能以加速循环替代真实时间。

当前状态与最终候选哈希将在干净构建、灰度部署和 24 小时采样启动后更新。
