# remote-control-skills

面向 AI agent 分发的 iStoreOS/OpenWrt 远程控制 skill pack。

## 内容

- `quickstart-router-api`: 通过 QuickStart/LuCI API 读取和控制路由器，默认 Go helper，curl fallback。
- `istoreos-ssh-ops`: SSH 入口 skill，只读识别系统、磁盘、服务、Docker、日志，并按症状转入专项 skill。
- `istoreos-*`: 磁盘、LuCI、Docker、opkg、备份恢复、服务修复等专项 skills。

## 安装

从源码目录安装到当前 Codex/OpenCode 环境：

```sh
sh remote-control-skills/install.sh --codex --copy
```

更通用的安装方式：

```sh
sh remote-control-skills/install.sh --target /path/to/agent/skills --copy
```

开发时可用 symlink：

```sh
sh remote-control-skills/install.sh --target /path/to/agent/skills --symlink --force
```

`install.sh --codex` 的解析顺序：

1. `$CODEX_HOME/skills`
2. `list-codex-skills --roots` 返回的第一个 root
3. `/config/.codex/skills`
4. `$HOME/.codex/skills`
5. `/config/.agents/skills`
6. `$HOME/.agents/skills`

OpenCode 兼容安装：

```sh
sh remote-control-skills/install.sh --opencode --copy
```

## 跨 skill 路径约定

不要在 skill 里写死某一个 agent 的目录。需要从一个 skill 调用另一个 skill 时，按这个顺序解析：

1. 用户或 agent 显式提供的 `SKILLS_DIR`
2. 当前脚本所在目录推断出的 skill pack 根目录
3. Codex 常见目录
4. OpenCode config/cache 目录

文档示例保留 OpenCode fallback，是为了兼容旧运行环境；它不再是唯一默认路径。

## 打包发布

生成 zip 压缩包和校验和：

```sh
sh remote-control-skills/package.sh --zip
```

输出：

- `remote-control-skills.zip`
- `remote-control-skills.zip.sha256`

也可以同时生成 zip 和 tar.gz：

```sh
sh remote-control-skills/package.sh --all
```

tar.gz 输出：

- `remote-control-skills.tar.gz`
- `remote-control-skills.tar.gz.sha256`

发布时同时提供下载 URL 和 sha256。建议在 GitHub Release、对象存储或内部制品库发布 archive 和 `.sha256`。

## 让 AI 通过 zip 链接安装

可以让用户把 `remote-control-skills.zip` 的链接发给 AI，让 AI 下载并安装成对应 skills。AI 必须先说明以下信息并等待确认：

- 下载 URL
- 期望 sha256
- 解压目录
- 安装目标 skill 目录
- 使用 copy 还是 symlink

确认后执行类似流程：

```sh
url="https://example.com/remote-control-skills.zip"
sha256="<expected-sha256>"
tmp="${TMPDIR:-/tmp}/remote-control-skills-install"

rm -rf "$tmp"
mkdir -p "$tmp"
curl -fL "$url" -o "$tmp/remote-control-skills.zip"
printf '%s  %s\n' "$sha256" "$tmp/remote-control-skills.zip" | sha256sum -c -
unzip -q "$tmp/remote-control-skills.zip" -d "$tmp"
sh "$tmp/remote-control-skills/install.sh" --codex --copy
```

如果没有 `sha256sum`，macOS 可用：

```sh
printf '%s  %s\n' "$sha256" "$tmp/remote-control-skills.zip" | shasum -a 256 -c -
```

如果用户只给 zip 链接、没有 sha256，AI 应该先要求用户提供 sha256；如果用户明确要求跳过校验，AI 也必须先说明风险并等待确认。不要推荐 `curl URL | sh`，也不要在未校验 checksum 或未得到跳过校验确认前执行压缩包里的脚本。

tar.gz 安装流程只需要把 `unzip` 换成：

```sh
tar -xzf "$tmp/remote-control-skills.tar.gz" -C "$tmp"
```
