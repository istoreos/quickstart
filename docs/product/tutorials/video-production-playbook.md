# 软件教程视频制作手册：FastNet 最终 v5 基线

本手册用于下一期浏览器功能介绍和教程。基线是用户已经回复“已经 OK”的
`voice-b-v5` 成片；不是早期 A 风格、v3 慢速试听样本或带字幕版本。
本次只沉淀文档与确认状态，没有重新生成语音、测速、安装或渲染视频。

## 1. 最终成果与固定约定

| 项目 | 本期采用、下期默认沿用 |
|---|---|
| 已选发布标题 | iStoreOS 首页新玩法：FastNet 并发测速，安装、实测、添加卡片 |
| 内容结构 | 真实效果预览 → 安装 → 操作实测 → 必要原理 → 首页配置 → 结果复看 → 固定片尾 |
| 画面 | 视频服务器内 Playwright 操作、录制真实软件；确定性脚本执行 |
| 声音 | 原始 `speech_studio_tts.py --style b`，`gange-natural`，原生 `speed=1.0` |
| 品牌读法 | 画面写 `iStoreOS`，TTS 输入写 `I Store OS`，短停顿连贯念 |
| 字幕 | `subtitles: false`；不烧录旁白字幕，不交付字幕轨、SRT、ASS |
| 保留文字 | 软件自身文字、章节名、安装命令、必要提示、片尾署名 |
| 规格 | 1920×1080，固定 30fps，H.264 / yuv420p，AAC 48kHz / 128k |
| 本期时长 | 完整 99.867 秒，开头预览 21.967 秒，片尾预览 15.967 秒 |
| 封面 | 1200×900，4:3；主文案 FastNet / 并发测速，辅助文案安装·实测·添加卡片 |
| 检查方式 | 自动检查 + 看关键帧/动态片段 + 人工听感；每个制作任务最多三轮 |

本期第一轮 197 项技术检查与画面抽查通过；随后用户确认成片可用。
完整 MP4 的 SHA256：
`1dd1a99a84fcda7937163d92291a3040b68724251f5b4ee88dae0ad51cce7aa4`。
不把 197 当作下期固定检查数量，应根据新功能断言扩展检查项。

可信入口：

- [最终教程规范](home-fastnet-card-voice-b-v5.json)：旁白、B 参数、发音、字幕和预览设置。
- [最终验收记录](voice-b-v5-acceptance.md)、[执行状态](voice-b-v5-execution.json)。
- [原始 B 参数](gange-natural-style-b.json)。
- [脚本说明](../../../scripts/tutorial-video/README.md)。
- 私有环境附录：仓库根目录下 `scripts/tutorial-video/.local/video-production-environment.md`。
  保存服务器、现有解释器和原始脚本的实际路径；它受 Git 忽略，交接时单独备份。

## 2. 制作系统与服务器分工

```text
开发机：读代码、定目标/分镜、维护教程 JSON、SSH 编排、收集交付
  ↓ sync / remote.py
Windows 视频服务器：原始 B 风格 TTS → 测时 → 本地 ASR 对齐
  + Playwright 录真实页面 / Trace / 事件
  ↓ compile-voice.mjs + voice-timeline.mjs
edit-plan.json：镜头、取景、点击、连续旁白、转场和实际时间
  ↓ render.py + FFmpeg
无声章节 → 连续旁白混音 → 视频合成 → AAC 封装
  ↓ review-voice.py + 人工看/听
MP4 / 开头与片尾预览 / 旁白 WAV / manifest / review-history
```

浏览器、TTS 和渲染均在视频服务器执行。开发机用 Python 标准库和 SSH 编排；
访问待演示设备由现有设备脚本建立会话，账号和会话不写入教程规范。
TTS 地址 `127.0.0.1` 指**视频服务器自身**，不要从开发机误调自己的 localhost。

本期已有 Windows 11、RTX 5080、Python 3.13.18、Node 26.8.1、Playwright
1.65.0 / Chromium 158.0.8127.18、FFmpeg 和 Speech Studio 0.5.0 的运行记录。
机器还提供 mise、Scoop、kspeeder、ks-cli；不要求为了复用本期流程全部升级或重装。
这些是本次环境快照，下期先检查现状，不自动追新版本。

## 3. 声音：必须复用原始 B 风格

### 3.1 参数和实际调用链

以下内容来自已核对的 [B 预设](gange-natural-style-b.json)，不要缩写 instructions：

```json
{
  "model": "higgs",
  "voice": "gange-natural",
  "response_format": "wav",
  "speed": 1.0,
  "seed": 76421,
  "temperature": 0.88,
  "top_p": 0.95,
  "top_k": 80,
  "loudness": "video",
  "instructions": "带一点笑意和生活气的中文口语，亲切随意，像刚泡好茶后和熟人聊天；停顿自然，句尾不要刻意上扬；不要播音腔。"
}
```

业务配置还包含 `endpointEnv: TUTORIAL_TTS_ENDPOINT`。服务地址、原始脚本路径、
解释器和音色参考位置通过私有配置传入，不硬编码到业务脚本。

当前调用链是 `cli.py tts → tts.py → 原始 speech_studio_tts.py --style b`。
每段先执行原始脚本的 `--dry-run`，逐字段核对实际 request 与上述参数完全一致，
再生成；回执要求 `voice_state=existing`。仅设置 JSON 中的 `voiceStyle: b`
不能替代这条实际调用链。配置缺少 `speechDriver` 时 `tts.py` 会走直接 HTTP 分支，
不符合本期“复用原始脚本”的约定，应在生成前检查配置。

TTS 输入源 WAV 为单声道、24kHz、16-bit；混音交付 WAV / AAC 为 48kHz。
声学自然度不能由 seed、参数相等或 ASR 正确来证明，最终要试听。

### 3.2 参考声音与恢复

音色来源是本人授权的录音；参考区间 86.32–97.84 秒，共 11.52 秒。
参考逐字稿：

> 智能家居等各类应用，提供安装指南和问题解答。硬件深度适配，专注华硕路由器、软路由以及主流硬件设备。

现有参考 WAV 为 44.1kHz / 单声道 / PCM16；它与生成 WAV 的采样率不同，属于正常情况。
本期参考 WAV 指纹：
`f10af4483eaf69f1fad558ba9854039466a135e3173e7beb58deac1eaca7068e`。
缓存还覆盖 `.txt`、`.json`、`.words.json`，避免逐字稿改动后误命中旧音频。

优先复用现有 `gange-natural`，不重新清理、裁剪或克隆。只有音色丢失时，
才按原始脚本的 `--ensure-voice` 恢复；导入 `clean:false / transcribe:false`，
使用上述区间和准确逐字稿。恢复前先 dry-run。个人原始录音和音色不进入 Git。

### 3.3 自然节奏的文案规则

- 一段一个自然句群，通常 1–3 句、4–15 秒；按意义断段，不按每次点击断段。
- 用短句、逗号、句号和口语连接词。避免为了口播连续堆产品名、功能名。
- 旁白可以跨镜头连续播放；切镜头不强行切断声音，也不补入机械的章间静音。
- 先生成语音、测真实时长，再安排画面；不把声音压进预设秒数。
- 保持 `speed=1.0`；不用 `atempo`、变调或 `<|prosody:speed_slow|>` 修节奏。
- 某句不自然，先改文案和标点，再只重生成这一段；未经新反馈不调整整套 B 参数。
- 固定 seed 帮助复现，不能保证引擎升级后的 WAV 字节一致；认可样本和参数一起保存。

本期曾出现两个反例：碎成很多 1–2 秒句子，导致停顿多、机械；为了摆脱拖慢，
又把开头压到 7.6 秒，用户觉得太赶。最终解决方式是原始 B 风格 + 自然句群，
让画面适应声音。不要把“少停顿”理解成“全程抢着念”。

## 4. iStoreOS 发音词典与单词内停顿

标准文案、标题和界面始终写 `iStoreOS`；送入 TTS 的字符串使用：

```json
{
  "pronunciations": {"iStoreOS": "I Store OS"},
  "pronunciationControl": {
    "word": "iStoreOS",
    "spoken": "I Store OS",
    "maxInternalPauseSeconds": 0.12,
    "targetInternalPauseSeconds": 0.06
  }
}
```

读法是 I / Store / OS，但作为同一个品牌连贯说，中间不加逗号或句号。
早期使用连续小写 `istoreos` 的方案未满足听感，不能作为默认词典。
实际输入保存在 `spokenText` 和 `.input.txt`，原文保存在 `text`。

如果 TTS 把 Store 和 OS 拉开：

1. 保留原生 WAV；用同一音频生成实际 ASR 词级时间。
2. `brand_pronunciation.py` 仅在英语词间的候选范围内检查低电平片段。
3. 只有低于 -45dB、超过 120ms 的静音才收短至约 60ms，不按 ASR 间隙直接切发音。
4. 保存 `.native.wav` 与 `.brand-edit.json`；检查 PCM 重建、原/新哈希，保证其余样本一致。
5. 再运行 `tts` 更新音频清单的哈希/真实时长：应全部缓存命中，不重新合成；再 `align`。
6. 试听开头，确认发音、短停顿和中文衔接，再渲染。

本期实际只删除 145ms 静音，开头 WAV 从 10.240 秒变为 10.095 秒。
这是本期测量结果，**不是每期都删除 145ms**。ASR 间隔里可能仍包含发声，
不能自动删整个间隔。此工具目前固定处理 `opening-intro.wav` 和开头英语词组，
其他品牌或句中位置需要适配工具和断言，不能假设是通用发音修复器。

## 5. 固定结尾：署名、链接和关注

下期默认单独保留 `closing` 旁白段，原文逐字为：

> 最后，视频下方有开源地址，教程等链接，记得我叫酷友社小宝，关注我。感谢大家的支持。

使用同一 B 预设。本期该 WAV 为 7.4 秒，配约 8 秒的片尾镜头。
画面继续展示本期真实成果，叠加“酷友社小宝”和“开源地址 · 教程链接，都在视频下方”。
不用空黑屏拖时长，也不把整段口播做字幕。

前一段 `outro` 先总结产品价值、邀请体验，随后 `closing` 再署名。下一期根据
真实功能替换 outro，固定 closing 可在音色/参数/内容未变时缓存复用。
发布时视频简介必须实际填入核实过的开源、教程和安装链接，不只说“在下方”。
本手册不推断尚未给出的开源仓库地址。

## 6. 开头技巧：先建立认知，再展示结果

本期最重要的经验：观众第一次看到 `921.4 Mbps`，不知道它来自什么软件、
什么链路，也不知道与首页卡片的关系。单独放大数字不足以传达价值。

开头的阅读顺序固定为：

**这是 iStoreOS → 首页有测速卡片 → 点开的是新版 FastNet → 真实测试在动
→ 本次外网下载结果 → 回首页保存这个结果 → 接下来教你实现。**

### 6.1 最终 v5 的真实分镜时间

下表来自交付 `edit-plan.json`，镜头间有 0.1 秒重叠，所以区间有交叠。
它与早期 v2 的时间不同，下期不要套旧时间码。

| 镜头 ID | v5 时间 | 画面与运镜 | 旁白/信息任务 | 检查重点 |
|---|---|---|---|---|
| opening-home | 0.000–4.000s | 先露出首页品牌、导航、完整卡片；再平滑推近；真实流量图在变化 | “iStoreOS 首页，现在也能看测速成绩了” | 一眼能识别产品，卡片在首页，画面有真实动态 |
| opening-enter | 3.900–7.433s | 提示测速入口，实际点击，看到 FastNet 加载完成 | “用的就是刚更新的 FastNet” | 工具名与点击来源建立关联，不停在空白 iframe |
| opening-test | 7.333–10.933s | 同一场原始实测的开始按钮、运行状态；剪短等待 | “点一下，测测路由器的外网下载” | 保留真实运行变化，标明等待已剪辑 |
| opening-result | 10.833–15.033s | 从完整结果推进到“外网下载 921.4 Mbps”；读数停留 | “这次实测，外网下载跑到了九百多兆” | 链路标签与数字一起看见，推进后可读停留 ≥2秒 |
| opening-return | 14.933–19.333s | 回到首页同场结果 921 Mbps，再拉远交代位置 | “更新 Quickstart BETA，就能把成绩留在首页” | 首页取整与结果小数一致，明确更新前提 |
| opening-next | 19.233–22.067s | 稍作收束，提示安装·测速·添加卡片，衔接教程 | “下面带你装好，再跑一次” | 知道接下来会学什么，声音连续 |

前三镜共用同一条 10.095 秒 `opening-intro`，不是每个镜头重新开一句声音。
开头不塞并发原理、BETA 命令、安装详情；先让观众想体验，细节放到后面。

### 6.2 动态画面怎么选

- 优先取真实操作、页面运行状态、速率变化、流量图、弹窗展开/关闭；最终停在同场真实结果。
- 不用静止截图承担完整的前 20 秒；推拉镜头只能引导视线，不能代替真实运行素材。
- 本期复用已有同一场实测录像，不为改配音重跑测速，不拼接不同场次的运行和结果。
- 不让 AI 生成滚动数字、仪表盘或“更好看”的软件界面；未成功的 NAT / IPv6 状态也保留。
- 原片有必要等待可以剪掉，保留“等待已剪辑”提示，不暗示实际测速只花几秒。
- 首页显示 921 Mbps、完整结果显示 921.4 Mbps，来自实际 921.366451 Mbps 的不同精度。
  断言按各自格式检查，避免把首页强改成小数来配合文案。

### 6.3 可迁移到其他功能的开头模板

| 认知阶段 | 建议内容 |
|---|---|
| 识别场景 | 先给产品全景和当前页面，1句话告诉用户在哪儿 |
| 发现变化 | 指出本期唯一主卖点，并出现真实入口 |
| 看见操作 | 点击 → 真实变化 → 可读结果，优先选有动态反馈的操作 |
| 理解价值 | 用用户语言解释结果，不堆术语；必要前提只点一次 |
| 愿意继续 | 告诉用户接下来能照着做出来，进入安装/配置 |

每镜只有一个信息任务，避免同时放新产品名、版本号、原理和数字。
20秒左右是本期实际长度；下期以自然语音和信息清楚为准，不硬凑同秒数。

## 7. 正文分镜与操作断言

| 章节 | v5 时间 | 真实内容与断言 |
|---|---|---|
| install | 21.967–30.633s | 软件中心安装/更新 FastNet；同时给出 `is-opkg install app-meta-fastnet`；安装完成并核对版本 |
| run | 30.533–40.600s | 打开 FastNet，开始全链路体检；断言运行态、完成态和外/内网结果 |
| parallel-note | 40.500–49.067s | 解释多来源并发下载减少单节点限制；不承诺任何线路必然更快或绝对准确 |
| results | 48.967–58.533s | 外网/内网分别解释，NAT 和 IPv6 真实状态保留；断言数据对应同次测试 |
| beta | 58.433–69.200s | 官方 Quickstart BETA 入口、备份、检查与更新说明；已经装好本版本时不重复升级 |
| add | 69.100–84.000s | 自定义首页 → 添加模块 → 外网测速 → 确认放置 → 完成 → 按需确认整理 → 刷新后仍存在 |
| outro | 83.900–91.967s | 首页成果复看，邀请用户自己跑一次、分享成绩 |
| closing | 91.867–99.867s | 固定署名、链接提示、感谢关注 |

本期公开 BETA 目录见 [v5 规范](home-fastnet-card-voice-b-v5.json) 的 `betaUrl`。
它是本期版本证据，下期需要重新核对适用版本和架构，不能默认是届时最新版本。
软件安装和设备升级不应作为配音、渲染失败后的自动重试操作。

## 8. 运镜、转场、像素风格

本期真正使用的是 **FFmpeg 运镜/xfade + Playwright 渲染叠加图**。
没有引入 Remotion，也没有接入 Motion Primitives / 专用 MP 动画素材包；不要将
最初对 MP 动画的设想写成已完成能力。后续有明确素材再作为独立增强。

| 处理 | 已实现方式 | 使用原则 |
|---|---|---|
| 推近/拉远 | `timeline.mjs` 的 `cameraFilter`，smoothstep 缓动，FFmpeg zoompan | 全景先交代位置，再聚焦入口或结果；结束回到全景 |
| 连续镜头 | v5 前六个边界 `fade`，每次 0.1 秒 | 保持首页、FastNet、结果、回首页的空间关系 |
| 后续章节 | v5 其余边界 `pixelize`，每次 0.1 秒 | 轻量区分步骤；早期 0.3 秒不再是最终默认 |
| 局部强调 | 实际鼠标/点击记录、按钮高亮、卡片边框、提示标签 | 先提示再点击；一次只强调一个目标 |
| 章节/片尾 | Playwright 页面生成透明叠加 PNG，FFmpeg overlay | 统一字号、颜色和小像素角标，界面主体保持原样 |

结果读数要留时间阅读；本期结果推进后至少 2 秒。避免频繁晃动、持续放大、
过长转场或每次点击都全屏闪烁。画面动态来自功能反馈，装饰动画只负责导视。
转场重叠必须计入总时长：不是直接把全部章节秒数相加。

## 9. 语音与操作如何对齐

1. 以步骤 ID 保存文案，TTS 生成每段 WAV 和实际时长。
2. `speech-align.py` 调用视频服务器既有本地 ASR，保存词级时间和音频 SHA。
3. `compile-voice.mjs` 为实际词尾设置点击锚点，本期默认词尾后约 150ms。
4. `voice-timeline.mjs` 只调整视频片段时间。点击附近保留最多前后各约 220ms
   的正常速度窗口；等待区间可以压缩、结果可以多留。
5. 连续 `narrationTracks` 跨镜头播放，语音不变速；画面按 30fps 对齐。
6. 输出事件、实际时间、cue 和误差；目标误差 ≤300ms，失败时定位步骤 ID。
7. ASR 有误识别时核对原声、修正该功能的词尾候选，再重新编译，不默默跳过断言。

ASR JSON 仅供内部剪辑，**不等于字幕**。改动音频后必须重做受影响的对齐，
旧词级 JSON 的 audioSha256 不相等时编译器应拒绝。当前误差检查主要证明
计划中词尾与点击的关系，机器检查不能代替人工观看最终画面的同步感受。

## 10. 可复用脚本与边界

以下均位于仓库根目录 `scripts/tutorial-video/`：

| 文件 | 用途 | 下期注意 |
|---|---|---|
| `cli.py` | sync / prepare / preflight / tts / align / record / render / review / collect | 每次明确 `--spec`；默认值仍是早期首版，不会自动选择 v5 |
| `settings.py`、`remote.py` | 私有配置、环境映射、严格 SSH、UTF-16LE PowerShell 传输 | 不用手工拼接复杂中英文 SSH shell 命令 |
| `config.example.json` | 最小配置模板 | B 模式还需补 `speechPython/speechDriver/speechStyle/parentOutput` |
| `tts.py` | 原始 B 驱动、dry-run 核对、分段缓存与音频清单 | 缓存由请求和音色文件指纹决定；最终缓存也要备份 |
| `speech-align.py` | 本地 ASR 词级时间 | 旧音频哈希不匹配时重新对齐 |
| `brand_pronunciation.py` | 开头 I Store OS 静音修整 | 固定开头结构；不把它当任意单词修复器 |
| `device.py`、`backup-device.py` | 设备会话、安装前备份 | 会话与备份仅私有保存 |
| `capture.mjs`、`probe.mjs` | 浏览器录制、定位、事件、Trace 与页面检查 | 模式和选择器仍与 FastNet / Quickstart 绑定 |
| `prepare-voice.ps1` | 创建独立配音修订，验证父成片与复制回执 | 不复制全部素材/音频缓存；拒绝重复初始化 |
| `compile-voice.mjs` | 本期实际旁白、镜头、ASR cue 编译 | ID、词尾候选、片尾布局、素材链均含 FastNet 定制 |
| `voice-timeline.mjs`、`timeline.mjs` | 视频重排、帧舍入、运镜、时长验证 | 原始模板与已经重排的剪辑计划不可混用 |
| `render.py` | FFmpeg 合成、响度、无字幕、开头/片尾预览 | 保留 AAC fast 参数；可只重渲染某个章节 |
| `review-voice.py` | 本期版本、声音、字幕、发音修整、保留哈希检查 | 本期特有 921Mbps 等检查下期需要替换 |
| `test/` | 时间轴、无字幕、静音裁剪回归 | 修改这些算法后运行；只改文档不需要编码/测试整条视频 |

**新功能视频不能只改 JSON 标题就执行全部流程。** 现有框架提供可复用的底层工具，
不是任意项目一键生成器。需要先为新功能补齐选择器、初始化/恢复、状态断言、
镜头模板、词尾 cue 和审片规则，再录制彩排通过的脚本。

尤其 `visualSourceFromParent: parent-edit-plan.json` 是 v5→v4→v2 原始模板的
特定链路。若下期父版本是 v5，不能继续照抄此值：必须显式找到原始片段模板，
防止把已 retime 的片段再次当原始时间表重排。`prepare-voice.ps1` 本身也不是
从零制作新功能的初始化器。

## 11. 下次实际开工顺序与命令

### 11.1 先做分镜和任务树

每期先创建 `docs/product/tutorials/<功能>-plan.md` 与对应 JSON，写清目标用户、
唯一卖点、片长范围、版本/commit、真实结果、安装前提、旁白、镜头和断言。
推荐任务树：

```text
V01 核对代码、版本、安装入口、设备与公开资料
V02 定义分镜：识别场景→效果→安装→操作→结果→固定片尾
V03 新开头试听：原始B、品牌词典、自然速度；通过后生成其余段
V04 演示准备/备份→确定性彩排→浏览器实录→断言→恢复
V05 音频测时/ASR→时间轴→单章预览→完整渲染
V06 最多三轮验收：自动、看画面、听声音、检查隐私与链接
V07 交付：完整片、开头/片尾、旁白、清单、封面与发布文案
```

每步记录输入、产物、断言、结果；相同任务修正沿用同一审片历史，最多三轮。
三轮仍不通过则说明具体问题和已完成内容，不删除历史或换目录绕过限制。
用户提出新的修订范围，才建立独立修订并保留父版本。

### 11.2 命令骨架：新修订适配完成后使用

以下是**操作模板，不是在本次文档任务中执行的命令**。先创建并核对新 spec 和
私有配置；将示例文件名替换为实际名称。不要对已验收 v5 目录重新 `prepare`。
独立终端/工具执行不会继承上次 export，需每次传入 `TUTORIAL_CONFIG` 或在同一 shell 设置。

```sh
export TUTORIAL_CONFIG=scripts/tutorial-video/.local/NEXT.config.json

rtk proxy python3 scripts/tutorial-video/cli.py sync --spec NEXT.json
# 仅在该修订采用已适配的 prepareScript 时初始化一次：
rtk proxy python3 scripts/tutorial-video/cli.py prepare --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py preflight --spec NEXT.json

# 先完成需要的确定性录制与素材准备，再按实际旁白编译。
rtk proxy python3 scripts/tutorial-video/cli.py tts --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py align --spec NEXT.json
# 若仍使用同一开头结构且确有过长单词内静音，再运行品牌修整；见私有附录。
rtk proxy python3 scripts/tutorial-video/cli.py render --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py review --round 1 --spec NEXT.json
# 实际看过画面、听过声音之后才记录 visual-pass，不能自动代填：
rtk proxy python3 scripts/tutorial-video/cli.py review --round 1 --visual-pass --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py collect --delivery delivery-NEXT --spec NEXT.json
```

`preflight` 的 `--spec` 目前只是 CLI 接受的参数，实际检查仍执行固定
`preflight.ps1`；新功能如有额外就绪条件，应适配预检，不能假设已按新 JSON 检查。
首次 `bootstrap` 会安装依赖；已有可用环境不重复执行。

仅修订本期素材时可参考这些既有录制入口，**不是配音修订的必跑清单**：

```sh
rtk proxy python3 scripts/tutorial-video/cli.py record --mode opening --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py record --mode home --spec NEXT.json
# 下列会真实安装/测速，仅有需要且范围明确时执行：
rtk proxy python3 scripts/tutorial-video/cli.py record --mode install --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py record --mode measure --real-test --spec NEXT.json
```

当前录制器按 `--mode` 工作，不能因为传了新 `--spec` 就期待它自动识别新功能。
一段声音变更时，重新 `tts → align → render`；未改段落命中缓存。
单章/装配入口：

```sh
rtk proxy python3 scripts/tutorial-video/cli.py render --scene closing --spec NEXT.json
rtk proxy python3 scripts/tutorial-video/cli.py render --assemble-only --spec NEXT.json
```

前提是其他章节和剪辑计划仍有效；改变前面镜头长度后应重新编译整条时间轴。

## 12. 编码和稳定性经验

- 本机 FFmpeg 开发版曾在 AAC 192k / 默认 coder 卡住。沿用 AAC 128k、
  `-aac_coder fast`，先视频、再旁白混音、最后 mux，日志分开。
- 当前用 `-filter_complex_threads 1`、输入 `-threads 1` 避免同一渲染过度并发；
  先保留已验收参数，不以改配音为由重写编码方案。
- 声音后期目标约 -16 LUFS、真峰值约 -1.5dBTP；v5 实测 -16.16 LUFS / -1.48dBTP。
- 点击附近正常速度，等待可剪短；保留源事件、色块校准和实际偏差，不能仅按脚本调用时间推断录屏帧。
- 多个 LuCI 弹窗可能同时存在，按准确可访问名称定位，不用模糊的第一个 dialog。
- 首页布局改动前保存版本，恢复前检查并发写入，冲突时不覆盖他人改动。
- 无字幕修订不能复用带烧录字幕的旧章节；复用必须确认 `sourceHasSubtitles:false`。
- 本期 v5 复用 11 段音频、10 个无字幕章节并逐个核对哈希；认可内容能不重做就不重做。

## 13. 验收与交付

每期至少检查：

- 全部页面状态断言通过；安装完成、实际结果、持久保存等成功状态有证据。
- 先有目标提示再点击；关键操作与旁白 cue 目标误差 ≤300ms。
- 开头能辨认产品、功能、入口、真实运行和结果；无长时间静止截图。
- 数值和链路说明匹配，未把内网速度当外网宽带速度；等待剪辑已标明。
- 原始 B request、实际驱动回执、音色指纹、WAV 时长及 ASR 音频哈希一致。
- 人工听英语品牌、整体语气、句间节奏、结尾姓名；声音自然度单独确认。
- 无字幕轨、无烧录旁白字幕、无 SRT/ASS；可保留章节/命令/操作文字。
- 1080p30、编码、解码、实际时长、响度、转场重叠和结果停留通过。
- 用户名、密钥、地址等个人/私密信息已避开或遮挡；临时布局和会话已处理。
- 有单独开头/片尾预览，单段可重生成，失败可定位 scene/step ID。
- manifest 能关联产品版本、源素材、请求与音色哈希、脚本快照、父成片和验收历史。
- 简介链接已实际填入并核对；封面和标题准确，不把本期实测当普遍承诺。

交付目录保存：完整 MP4、开头/片尾 MP4、完整旁白 WAV、关键试听 WAV、
`manifest.json`、`edit-plan.json`、`review-history.json`、封面 PNG 及其生成提示词。
本期完整/预览文件名见私有附录；不要把这些生成媒体加入 Git。

应进入 Git：本手册、教程 JSON、参数预设、脚本、模板、断言、必要测试。
不进入 Git：私有配置/环境附录、原始录音/音色、音频/视频/封面、Trace、会话、
缓存、模型、账号/密钥和依赖目录。当前 `.local/` 已整体忽略。
本期文档新增不代表自动提交或发布；私有附录与媒体需按团队私有备份机制交接。

## 14. 标题与封面复用

标题采用产品名 + 具体变化 + 可学会的动作。本期标题是用户最终选定的版本，
不再用“更快/绝对准确”等无法由单场测试证明的说法。

封面采用 1200×900：iStoreOS 首页新玩法 / FastNet / 并发测速 /
安装·实测·添加卡片；真实首页截图和测速卡片作为依据，右下保留真实 921 Mbps
及上传 65.6 Mbps，署名酷友社小宝。深蓝、青绿、清楚的大字与软件色系保持一致。

本期封面用 imagegen 内置工具生成，再按相同比例精确缩放到 1200×900。
后续封面可以使用生成式设计；正式教程的软件界面仍须真实录制。
若封面涉及精确 UI 或数字，应逐项核对，必要时用真实截图确定性合成。
提示词和成品存放在私有交付目录，详见环境附录。

## 15. 下次交给 AI 的简短任务说明

> 先阅读 `docs/product/tutorials/video-production-playbook.md` 和私有环境附录。
> 以用户认可的 FastNet voice-b-v5 为制作基线：视频服务器真实录浏览器，
> 原始 `speech_studio_tts.py --style b`、gange-natural、speed=1，无字幕；
> iStoreOS 显示不变，实际读 I Store OS，短停顿；片尾使用手册固定原句。
> 先核对新功能代码/版本与公开安装入口，写分镜、目标、任务树和成功断言，
> 再适配现有 FastNet 专用录制/编译/验收脚本。开头按产品全景→新功能入口→
> 动态实测→真实结果→回到使用场景，不要直接放大一个无上下文的数字。
> 先试听开头，再批量生成；音频保持自然速度，画面按真实词级时间对齐。
> 每项修改只重做受影响步骤，保留认可素材及父版本，最多三轮验收后交付。
