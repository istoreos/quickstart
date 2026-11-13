# FastNet 首期浏览器视频

## 后续制作先看最终基线

用户已认可最终 `voice-b-v5`。下一期请先读
[视频制作手册](../../docs/product/tutorials/video-production-playbook.md)，以及私有
`scripts/tutorial-video/.local/video-production-environment.md`。默认是原始 B 风格、
`I Store OS` 发音、固定小宝片尾、无字幕；下面首版/A 风格/v3 部分是历史记录，
不代表当前默认。CLI 的默认 spec 仍为首版，调用时必须明确 `--spec`。

轻量实现：Playwright 录真实界面，Speech Studio 生成分段中文旁白，FFmpeg
按结构化计划合成；无 Remotion/React 工程，不改产品前后端。

## 配置

复制 `config.example.json` 至 `.local/config.json`，填入自己的服务器、路径和
两个测试设备。也可用 `TUTORIAL_CONFIG` 指向仓库外配置。SSH 使用已有身份，
不要复制私钥到本目录。素材、音色、会话和安装备份均不进入 Git。
声音参数在 `docs/product/tutorials/home-fastnet-card.json`，服务地址在私有配置。

## 运行

在当前开发机运行 Python 编排，浏览器、TTS、渲染在视频服务器执行：

```sh
python3 scripts/tutorial-video/cli.py sync
python3 scripts/tutorial-video/cli.py bootstrap
python3 scripts/tutorial-video/cli.py preflight
python3 scripts/tutorial-video/cli.py tts
python3 scripts/tutorial-video/cli.py record --mode install
python3 scripts/tutorial-video/cli.py record --mode installed
python3 scripts/tutorial-video/cli.py record --mode measure --real-test
python3 scripts/tutorial-video/cli.py record --mode beta
python3 scripts/tutorial-video/cli.py record --mode home
python3 scripts/tutorial-video/cli.py render
python3 scripts/tutorial-video/cli.py review --round 1
# 打开输出目录的 review-round-1 图像逐张检查，再记录结果：
python3 scripts/tutorial-video/cli.py review --round 1 --visual-pass
python3 scripts/tutorial-video/cli.py collect
```

Shell 运行时遵循工作区 RTK 约定；本机 Node 测试通过 SSD 入口：

```sh
rtk proxy /var/cache/build-cache/run.sh -- node --test scripts/tutorial-video/test/*.test.mjs
```

安装模式会升级配置的安装测试机，先保留 FastNet 文件和包数据库备份。
`measure` 是真实外网/内网检测，不作为每轮审片的自动重试步骤。
`home` 在片外移除既有卡片，录制真实添加、保存、刷新，最后按配置版本恢复
原布局；发生并发写入时停止恢复，不覆盖别人的修改。

## 产物与重做

- `capture/`：各步骤原始视频、私有 Trace、断言、事件和实测结果。
- `audio/`：按文案、参数和音色哈希缓存的旁白与真实时长。
- `edit-plan.json`：实际取景、事件、旁白和剪辑区间。
- `render/calibration.json`：片外色块与录屏帧的时间校准。
- `fastnet-home-card-zh-CN.mp4`、`.srt`、`manifest.json`：交付产物。
- `review-history.json`：最多三轮，第四轮会拒绝。

单步重录用对应 `record --mode`。只改文案可重新运行 `tts`，未改段落复用
缓存；然后重新编译和渲染。原始事件迟到时根据实际点击重新安排音频，保留
`captureErrorMs`，不把执行偏差藏掉。非必要等待按明确区间剪掉，不改变结果。

只重做某个剪辑章节并重新封装已有章节：

```sh
python3 scripts/tutorial-video/cli.py render --scene intro
python3 scripts/tutorial-video/cli.py render --assemble-only
```

每个独立制作任务使用独立的私有输出目录。同一任务修正沿用其目录和三轮
验收记录，不能删除记录绕过上限。本次任务已在第三轮通过，不再自动追加修正。

## 用户授权的开头修订 opening-v2

分镜和任务树见 `docs/product/tutorials/opening-v2-plan.md`，实际文案见
`home-fastnet-card-opening-v2.json`。此修订独立保留原片，首页采用现有取整显示
921 Mbps，完整界面保留同场实测 921.4 Mbps。不开新测速，不重复安装。

私有配置增加 `parentOutput`（首版目录），`output` 使用新的修订目录。每个命令
均通过 `TUTORIAL_CONFIG` 指向该配置。以下为新任务初始化流程，已完成的本次
修订不要再次运行录制或 prepare：

```sh
export TUTORIAL_CONFIG=scripts/tutorial-video/.local/opening-v2.config.json
python3 scripts/tutorial-video/cli.py sync --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py prepare --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py tts --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py record --mode opening --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py render --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py review --round 1 --spec home-fastnet-card-opening-v2.json
# 完成视觉抽查后记录同一轮结论，再 collect；不要与 review 并发收集。
python3 scripts/tutorial-video/cli.py review --round 1 --visual-pass --spec home-fastnet-card-opening-v2.json
python3 scripts/tutorial-video/cli.py collect --delivery delivery-opening-v2 --spec home-fastnet-card-opening-v2.json
```

本次修订把两次录制检查失败也计入上限，第三轮通过；`capture` 与 `review`
都会拒绝在该目录开启第四轮。`prepare` 拒绝覆盖已初始化的修订。未改动的六个
后续章节复用首版中间视频/音频，验收逐一比对哈希。新增开头和并发说明章节
独立渲染，整片同时生成 `fastnet-opening-v2-preview.mp4` 方便审查开头。

字幕来自批准文案；点击提示以短句音频结束为 cue，编码后按帧检查。
句内字幕按文本分块分配，不宣称逐字强制对齐。源视频经同步色块校准。

软件升级先运行 preflight，并针对目标模式彩排。首版不包含跨版本自动修复
或批量调度；断言定位到具体失败步骤后再维护脚本。

## BETA 更新

Quickstart 已安装当前 BETA 时只展示官方入口与检查，不重复升级同版本。
完整系统备份、适配检查和官方安装器的交互确认应保留。
视频简介附官方版本目录及 `is-opkg install app-meta-fastnet`。

## 配音修订 voice-v3（正在试听，未替换成片）

用户反馈v2机械、停顿多，首个连续句群7.6秒样本又太赶；当前提供相同内容的
9.68秒自然断句版和11.38秒原生慢语速提示版，等待听感反馈。完整参数以
`docs/product/tutorials/gange-natural-style-a.json`为准，实际请求保留在私有音频
目录的`.request.json`中。所有数值保持原A风格，`speed=1.0`；慢语速候选只在
input加入Higgs支持的`<|prosody:speed_slow|>`，不使用后期atempo。

`render.py`已支持`subtitles:false`和独立连续旁白时间轴`narrationTracks`；
禁止在无字幕修订中复用旧的带烧录字幕章节。新成片需从原始素材重渲染。
目前尚无voice-v3成片，不要将草稿旁白或参数通过当作用户听感通过。

可运行无编码的渲染契约检查：

```sh
rtk proxy /var/cache/build-cache/run.sh -- python3 -m unittest discover -s scripts/tutorial-video/test -p test_render.py -v
```

TTS缓存指纹包含参考WAV、逐字稿、音色元数据及已有words文件，避免仅WAV
未变时误用旧逐字稿缓存。声音自然度需要实际试听，机器检查仅证明请求、
文件和字幕/变速处理符合预期。

## 用户指定的原始B风格 voice-b-v4

原始B预设见`docs/product/tutorials/gange-natural-style-b.json`；本次不是v3试听里
在A预设上加speed_slow标签的候选B。使用原始seed76421、temperature0.88、
top_p0.95、top_k80、speed1和原始instructions，所有章节统一使用B。

私有配置在原配置基础上设置：`speechPython`为已安装的Python3.13.18解释器，
`speechDriver`为用户原仓库的`scripts/speech_studio_tts.py`实际路径，`speechStyle`
为`b`；`parentOutput`为已验收的opening-v2目录，`output`使用独立voice-b-v4目录。
每段先运行原脚本的`--dry-run`，核对request完全一致后调用`--style b`生成，
语音回执必须报告复用existing音色；不重新克隆声音。

```sh
export TUTORIAL_CONFIG=scripts/tutorial-video/.local/voice-b-v4.config.json
python3 scripts/tutorial-video/cli.py sync --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py prepare --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py tts --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py align --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py render --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py review --round 1 --spec home-fastnet-card-voice-b-v4.json
# 画面抽查后记录同轮结论，再收集；无SRT/ASS交付。
python3 scripts/tutorial-video/cli.py review --round 1 --visual-pass --spec home-fastnet-card-voice-b-v4.json
python3 scripts/tutorial-video/cli.py collect --delivery delivery-voice-b-v4 --spec home-fastnet-card-voice-b-v4.json
```

`compile-voice.mjs`根据实际ASR词尾安排点击，连续旁白可以跨镜头；
`voice-timeline.mjs`只重排原始视频片段，点击前后短窗口保持正常播放速度。
音频不变速、不变调，所有场景重渲染以去除旧版烧录字幕。原参考/音色、
录屏、测试结果和旧视频保持不变。词级JSON是内部编辑数据，不生成字幕。

品牌的显示形式与送入TTS的发音形式分别记录在`pronunciations`和`spokenText`。
连续拼写只是一种发音引导，不能用ASR结果替代用户对英语品牌连读的试听判断。

## v5：I Store OS读法与小宝片尾

用户认可v4的B风格后，v5只更正开头发音和追加指定片尾。配置文件为
`home-fastnet-card-voice-b-v5.json`；`parentOutput`指向v4，`output`使用新目录。
画面保留iStoreOS，pronunciations改为`I Store OS`，空格提示英文边界，不加标点。
本次实测Store与OS间存在一段145ms低电平静音；`brand_pronunciation.py`只删除
这段空白，保留约60ms静音以及全部其余PCM样本，不做音频变速或变调。

制作顺序为：prepare→复制v4的audio/alignment缓存和无字幕video中间章节→tts→
align→在视频服务器运行`python brand_pronunciation.py --spec home-fastnet-card-voice-b-v5.json`
→再次tts刷新实际音频哈希/时长（缓存命中，不重新合成）→align→render→review→collect。
原生开头保留为`.native.wav`，静音编辑以`.brand-edit.json`保存原始/产出哈希
和采样级切点，验收会重建PCM验证发声部分没有改动。

其余11段认可的B旁白直接复用；10个无字幕章节复用原视频并比对哈希，避免
配音以外的画面变化。片尾新增完整口播和“酷友社小宝”像素提示卡。
collect同时导出开头预览及`fastnet-style-b-v5-ending.mp4`片尾预览。
