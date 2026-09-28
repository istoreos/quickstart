# M64 发布候选状态

日期：2026-09-28

M64 只有在最终候选完成连续 24 小时拓扑采样后才能关闭。采样必须同时验证：A/B Quickstart 未重启且 API 为 HTTP 200、浮动地址仍由 B 持有、C 经 B 上网、D 经 A 上网、两端 RSS/线程/FD 有界。

本目录的 `soak.tsv` 由 `lan-device-topology-soak.sh` 每 300 秒追加一次；完成后必须由 `analyze-lan-device-topology-soak.sh` 以默认 86400 秒门槛验收。未达到持续时间时分析器必须失败，不能以加速循环替代真实时间。

## 当前候选

- Quickstart 源码：`18bde6271d319cbcd61cc85b6a791feaf5336b78`
- NetPolicy 源码：`863ed0444b63316a5d5aa6fce25be62f92d297f6`
- 发布包：`quickstart-binary-0.14.0.tar.gz`
- 发布包 SHA-256：`9685fd810ee995155b0a6b5fd263311e2e39ffdc627d47ebf515189b9e05e9ea`（连续两次干净构建一致）
- x86_64：`b0c37f06f22f4cdb8c9e8f7b47b4b4252261dee451af46c919f0947a85f7369d`
- arm64：`0c518571cd9749b70594286c9828dff1ea81129d04eff3ed238787fba1467abc`
- armv7：`38b34cb343aedb8adb1f8611d7d49f467e858765f0326008fb78bd357c496cf9`
- B 灰度回滚目录：`/root/quickstart-backups/0.14.0-18bde6271d31-20260928024700`

## 即时门禁

- 后端 1,562 项、前端 65 项、NetPolicy 43 项、部署契约、55/55 五层矩阵通过。
- B 的二进制与发布包 x86_64 哈希一致；Web、英文词典和资源版本校验通过。
- 四机拓扑、业务策略、IPv4 限速、高级工具、登录态响应式 UI 通过。
- IPv6 以 20 Mbit/s UDP 恒定负载验证 2 Mbit/s 双向 policing：下载 `2.33 Mbit/s`、上传 `2.08 Mbit/s`，回滚后双向恢复约 `20 Mbit/s`。
- 主路由 A 未部署候选；network、dhcp、firewall、dnsmasq 和 Quickstart PID 前后不变。

## 24 小时门

- 状态：**运行中，M64 尚未完成**。
- 开始：`2026-09-28T02:49:40Z`；最早可验收：`2026-09-29T02:49:40Z`。
- 本地监测 PID：`548493`；输出：`docs/evidence/m64/soak.tsv`；日志：`/tmp/quickstart-m64-soak.log`。
- 首样本正常：A/B API 200，VIP 仅在 B，C 经 B、D 经 A，双方连通。
