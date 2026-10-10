# ignoretrail：一条命令看懂 .dockerignore 为什么排除这个文件

一句话用途：为指定路径列出 Docker 忽略规则的包含状态，以及改变该状态的原始规则行。

## 安装与最小示例

需要 Go 1.24+ 和 Make；本轮实际使用 Linux amd64、Go 1.27.2。依赖已放入
`vendor/`，构建与运行均不需要登录、联网或 Docker daemon。

在项目根目录执行：

```sh
make build
./bin/ignoretrail examples/negation sub/b.txt
```

交付环境的 Go 在项目内部；如果当前 PATH 没有 Go，构建命令是：

```sh
PATH="$PWD/.tools/go/bin:$PATH" make build
```

示例的 `.dockerignore` 是：

```dockerignore
# Exclude the directory, then keep one file.
sub/
!sub/b.txt
```

输出：

```text
INCLUDED "sub/b.txt"
  exclude .dockerignore:2 "sub/"
  include .dockerignore:3 "!sub/b.txt"
```

它说明第 2 行先排除了父目录，第 3 行又把这个文件放了回来。
再查 `sub/a.txt`，它仍然是 `EXCLUDED`，不能因为存在任何 `!` 规则就认为所有文件
都可用。需要批量检查时仍然只是一条命令：

```sh
./bin/ignoretrail examples/negation sub/a.txt sub/b.txt
```

返回码 0 表示全部包含，1 表示至少一个路径被排除，2 表示参数、规则或文件读取错误。
返回 1 是诊断结果，不是程序崩溃。

## 怎样用于自己的项目

命令格式为 `ignoretrail [-f Dockerfile] CONTEXT PATH...`。在其他目录可以通过
本工具二进制的绝对路径运行。待查询路径和 `-f` 都相对于 CONTEXT；参数中的路径
按字面值检查，不展开 glob，也不要求文件已经存在。

```sh
/path/to/ignoretrail -f docker/build.Dockerfile /path/to/project src/app.py
```

这里优先读取 `/path/to/project/docker/build.Dockerfile.dockerignore`，没有才读取
`/path/to/project/.dockerignore`；两份文件不会合并。默认 `-f Dockerfile`，因此
根目录的 `Dockerfile.dockerignore` 也会生效。大小写不同的 `dockerfile` 需要显式指定。

## 实现思路与 TDD

匹配交给 Moby 官方 `patternmatcher v0.6.1`，包括目录继承、`**`、`!`、字符集合
等语义；本项目不另写一套类似但不相同的 glob 算法。逐行预处理也复用官方
`ignorefile.ReadAll`，保留原始行号，并确保只有第一行的 BOM 被当作文件头。

对每个查询路径，依次计算“读完前 N 条规则”时的状态。只有状态改变才输出该行，
所以轨迹表达的是排除与重新包含的过程；不会把未改变状态的重复规则误称为新原因。
上下文根目录 `.` 始终包含。源文件内容不参与计算，程序也不会写回 ignore 文件。

21 个垂直 TDD 循环均保留真实日志：先执行新测试看到失败，再只补齐该行为。
第一轮的失败是尚无生产代码，随后各轮是输出或返回码的行为失败。测试通过子进程
执行真实二进制，没有用 mock 代替匹配器或文件系统。FIFO 测试首次实际超时，加入
普通文件检查后通过；符号链接测试首次读出了外部规则，改用 Go `os.Root` 限定读取
范围后通过。所有“外部”测试夹具也都在本项目的 `.tmp/` 内。

```sh
make test
make vet
```

Makefile 把构建缓存和临时目录固定在本项目内，并禁用 Go 网络下载。
公开来源、版本比较和唯一选题在 [problem.md](problem.md)，测试记录在
[tdd.md](tdd.md)。

## 限制与自评

这是规则解释器：不会修复第三方 SDK、自动改规则、解析 COPY/ADD、检查镜像内容或
保证构建成功；Docker 对 Dockerfile/ignore 文件的特殊传输行为不在这个答案内。
不遍历整个项目，也不验证目标文件是否存在。只支持本地上下文；越出上下文的
ignore 文件符号链接会被拒绝。查询过程中不要同时修改 ignore 文件。

单行长度受上游约 64 KiB 限制；当前逐前缀解释的工作量随规则数量平方增长，适合
少量路径和通常大小的 ignore 文件。已验证 Linux，尚未验证 Windows。

我认为值得作为小型开源工具公开：它把必须写代码或启动构建器才能完成的逐行排查
做成了一个离线命令。价值在可解释的使用方式与清楚的边界，匹配算法仍然属于上游。
本轮没有做任何 GitHub 写操作；是否发布由林峰决定。
