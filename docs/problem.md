# 选题：ignoretrail

调研日期：2026-10-10（Asia/Shanghai；抓取时 UTC 为 2026-10-09）。
近三个月窗口：2026-07-10 至调研时刻。以下只按原帖或评论的发布时间计数，
不用更新时间，不把同一用户的多条帖子重复计数。网络仅用于公开上游资料和官方依赖。

## 最多三个候选

| 候选 | 能做的本地动作 | 调研判断 | 决定 |
| --- | --- | --- | --- |
| Python 同名文件遮蔽导入 | 找出项目文件与模块名的冲突 | Ruff 当前 0.16.10 的 A005 已覆盖标准库模块遮蔽；本轮没有核实三位近期用户针对仍未覆盖的同一场景的报告。图分析误判、循环导入和真正遮蔽不能混算。 | 不选 |
| 锁文件中的旧 registry URL | 离线重写明确指定的 URL 前缀 | npm 当前 12.2.0、pnpm 当前 12.10.1；近期结果多为 scope 路由、可选 peer、供应链验证或版本解析问题，不能证明改锁文件 URL 就能解决。未凑齐三个符合这一具体场景的用户。 | 不选 |
| `.dockerignore` 的目录排除与 `!` 重新包含难以排查 | 对给定路径打印包含状态、改变状态的规则及原始行号 | 三位不同用户、三个高星上游项目的近期独立报告成立；无需构建、登录、读取源文件正文或网络。 | **唯一入选** |

没有选择 shebang、CRLF 或其变体，也不添加其他开发者工具功能。

## 三条近期独立原话

1. **saeedturk1969**，**2026-09-06 15:32:09 UTC**，
   [moby/buildkit #7125](https://github.com/moby/buildkit/issues/7125)：

   > `CopyIgnoredFile` stops firing entirely once the `.dockerignore` contains **any** negated (`!`) pattern — including a negation for an unrelated directory, and including one naming a path that does not exist.

   > That is not hypothetical. It happened to us twice in five days.

   报告给出 `sub/` 与 `!sub/b.txt` 等组合：`sub/a.txt` 仍被排除，但当时的检查静默通过。
   报告使用 BuildKit 0.31.2。当前版本已经是 0.34.0，**不将旧版本的这个具体 bug
   未经复测地宣称为当前版本仍存在**。证据支持的是排除/重新包含的排查成本。

2. **jyn514**，**2026-10-07 10:11:41 UTC**，
   [docker/docker-py #3448](https://github.com/docker/docker-py/issues/3448)：

   > When I generate a build context with `docker.utils.tar`, the rules `**` and `!**/a.txt` omit every nested `a.txt`. Docker's BuildKit keeps those files with the same input. The SDK therefore drops files that the exception explicitly includes.

   报告包含 SDK 输出与 BuildKit 对照，并说明使用 LLM 协助、本人审核。
   引用按其公开发布的原话保留；不是本项目亲自运行该 SDK 的结论。
   同一作者的 #3449、#3450 不另计人数。

3. **anuragpaul602-netizen**，**2026-07-20 13:13:59 UTC**，
   [langchain-ai/langgraph #8387](https://github.com/langchain-ai/langgraph/issues/8387)：

   > N/A — no exception is raised. The bug is silent data loss: the archive is built successfully but is missing a file the .dockerignore explicitly re-includes.

   > In the repro above, `.dockerignore` excludes `vendor/` but explicitly re-includes `vendor/keep/**`. Expected output is `['.dockerignore', 'app.py', 'vendor/keep/important.py']`. Actual output is `['.dockerignore', 'app.py']` — `vendor/keep/important.py` is silently missing from the deploy archive.

   这是用户自己给出复现、原始输出和本地补丁的报告，版本为 langgraph-cli 0.4.31。
   当前官方 PyPI 版本为 0.4.33；调研时 main 的 `archive.py` 仍显示按目录匹配
   剪枝，但本项目不声称已经运行或修复其发布版本。

抓取时三个仓库的公开 star 数分别为 **10,312 / 7,216 / 42,963**。
原始链接、时间、用户和正文保留在 `research/evidence.json`，便于复核。
三条报告不是同一个底层 bug；它们共同证明需要独立核对 Docker 规则下一个路径
应当被排除还是重新包含。工具的范围严格限于这个核对步骤。

## 长期存在的佐证，不计入三条

[jest 的 BuildKit #6926](https://github.com/moby/buildkit/issues/6926)，
2026-07-03：`.*` 与 `COPY . .` 的误报。已超出窗口，计数为零。

## 现有工具与当前版本核查

- **Docker / BuildKit**：[BuildKit v0.34.0](https://github.com/moby/buildkit/releases/tag/v0.34.0)，
  2026-10-07 发布。[Docker 官方规则文档](https://docs.docker.com/build/building/context/#dockerignore-files)
  解释 `**`、`!` 和 Dockerfile 专属 ignore 文件优先级。`docker build --check`
  很有用，但需要构建前端/构建器和 Dockerfile；不提供任意给定路径的逐行状态变化记录。
  ignoretrail 的价值是无 daemon、无构建、无镜像下载的直接解释，而不是重新实现构建器。
- **Docker SDK for Python**：[7.2.0](https://github.com/docker/docker-py/releases/tag/7.2.0)，
  2026-07-09 发布。近期报告使用比它更新的 7.2.1.dev26 开发提交。
  SDK 是程序库，生成 tar 与 Moby 匹配可能有差异；不能把 SDK 的结果作为 Docker
  规则的唯一参考，也不能声称本 CLI 会修复 SDK。
- **langgraph-cli**：[PyPI 0.4.33](https://pypi.org/project/langgraph-cli/0.4.33/)。
  它负责应用打包/部署，不是 `.dockerignore` 逐行解释工具。替换平台打包器、
  发起部署或调用服务均不在本项目范围。
- **Moby patternmatcher**：[v0.6.1](https://github.com/moby/patternmatcher/releases/tag/v0.6.1)，
  2026-03-24 发布。它是现成的准确匹配库，本项目直接依赖，不宣称发明新的 glob 语义。
  开发者仍需自己编程来连接 ignore 文件预处理、文件选择、规则行号、状态轨迹和 CLI。
- **docker-show-context**：已检索到原项目
  [pwaller/docker-show-context](https://github.com/pwaller/docker-show-context)，
  最新 tag 为 **v1.1.1**，仓库 HEAD 为
  [`19f133f`](https://github.com/pwaller/docker-show-context/tree/19f133f08f92074bb8a6bb89f482532e5632d18b)
  （2021-04-12）。已读取该提交的 README：主要统计 tar 的时间、大小、目录和扩展名，
  只支持以构建上下文为当前目录且 Dockerfile 名称固定。它可用独立二进制离线运行，
  不是必须依赖 daemon；区别在于没有本工具的任意路径参数、逐行状态变化与原始行号。
- 未选候选的对照：[Ruff 0.16.10](https://github.com/astral-sh/ruff/releases/tag/0.16.10)
  与 [A005](https://docs.astral.sh/ruff/rules/stdlib-module-shadowing/)；
  [npm 12.2.0](https://github.com/npm/cli/releases/tag/v12.2.0)；
  [pnpm 12.10.1](https://github.com/pnpm/pnpm/releases/tag/v12.10.1)。

## 实施范围与原创点

命令：`ignoretrail [-f Dockerfile] CONTEXT PATH...`。
给定上下文和相对路径，输出 `INCLUDED` / `EXCLUDED`，以及实际改变状态的
`exclude` / `include` 规则、原始行号和文本；有排除返回 1，全部包含返回 0，
输入或文件错误返回 2。显式 `-f` 选择对应的 `<Dockerfile>.dockerignore`，没有则
使用上下文根的 `.dockerignore`。`-f` 默认值为 `Dockerfile`；路径相对于 CONTEXT。

原创部分是这一狭小工作流和可解释的输出，不是 glob 算法。用户一条命令即可回答
“按 Docker 规则这个路径该不该在上下文中，以及哪些行改变了它的状态”。
不解析 Dockerfile 的 COPY/ADD，不检查镜像，不生成 tar，不自动改 ignore 文件，
不读取路径所指文件的内容，也不验证该文件是否实际存在。
因此它解决**规则诊断**，不保证一次命令修复所有构建或第三方 SDK 的问题。

版本核查快照在 `research/tools.json`，检索范围与排除结果在 `research/searches.json`。
LangGraph 归档代码参考
[`archive.py@8657df8`](https://github.com/langchain-ai/langgraph/blob/8657df80f36625490b05a0bfa14e63ac747a8243/libs/cli/langgraph_cli/archive.py)。
本轮只读取公开上游资料，没有创建仓库、推送、PR、评论或消息。
