# gx 架构

`gx` 是一个用 Go 编写的命令行文本处理工具，定位为 ripgrep + sed + awk
的轻量、单二进制替代。本文解释 gx 的内部结构，便于阅读和贡献代码。

## 顶层布局

```
gx/
├── main.go              # 入口，命令路由 + 退出码
├── args/                # 共享参数解析
├── regex/               # 正则匹配器（grep 风格）
├── walker/              # 文件树遍历
├── processor/           # 文件级工作池引擎
├── stream/              # stdin / 文件统一输入
├── cmd/                 # 各子命令实现
│   ├── find/            # ripgrep 风格内容搜索
│   ├── list/            # grep -l 风格文件列举
│   ├── replace/         # 搜索 + 替换
│   ├── rename/          # 批量文件重命名
│   ├── cut/             # 字段提取（cut -f 子集）
│   ├── trans/           # 行变换（upper/lower/...）
│   └── script/          # Tengo 脚本（sed/awk 风格）
└── docs/                # 设计文档与 API 参考
```

`args/regex/walker/processor/stream` 是所有子命令共享的内部包。

## 入口与命令路由

[main.go](file:///opt/repos/gx/main.go) 非常薄：

1. 读 `os.Args[1]` 决定子命令。
2. 把 `os.Args[1:]`（去掉 `gx` 二进制名）传给子命令自己的 `ParseArgs`。
3. 根据子命令返回的 exit code 调用 `os.Exit`。

可测试入口是 `runMain(argv, out, errOut) int`，子命令仍由 `os.Exit`
终止进程，单元测试只覆盖 dispatch 部分（usage/version/未知命令）。

### 退出码

| Code | 含义                                            |
|------|-------------------------------------------------|
| 0    | 成功，至少有一行/一个文件被处理                 |
| 1    | 成功但无输出（grep "no match" 风格）            |
| 2    | 参数错误、编译错误、IO 错误、超时                |

注意：不同子命令对 exit 1 的语义略有差异：
- `find` / `list` / `trans` / `script`：1 = 成功但没有匹配/输出
- `replace`：1 同时覆盖"无匹配"和"参数错误"（历史行为）
- `rename`：1 = dry-run 无操作；2 = 冲突未解决
- `cut`：总是 0（设计就是无匹配也有空输出）

## 各包职责

### args

统一的参数注册与解析框架。`Option{Short, Long, HasValue, ValueName, Help,
Handler}` 描述一个 flag，子命令把选项注册进 slice 即可。每个子命令的
`getOptions()` 返回自己的选项列表。

`CommonConfig` 是各子命令共有的字段（`Pattern` / `Paths` / `FilePattern`
/ `IgnoreCase` / `DryRun` 等），由 `ParseSimple` 填充。

### regex

[regex.Matcher](file:///opt/repos/gx/regex/matcher.go) 包装 Go 正则，
对外暴露 `Find/FindAll/MatchFile` 三个方法。它在内部预编译并缓存
`regexp.Regexp`，向 `processor.FileProcessor` 屏蔽细节。

### walker

[walker.Walker](file:///opt/repos/gx/walker/walker.go) 把任意多个输入
路径展开为文件 channel：支持目录递归、glob 过滤、隐藏文件/二进制跳过。
输出流是 `<-chan walker.FileInfo`，无界缓冲由消费者控制。

### processor

[processor.Engine](file:///opt/repos/gx/processor/processor.go) 串联
walker + worker pool：

```
walker.Walk() → filesChan → N workers → proc.ProcessFile → proc.HandleResult
```

子命令实现 `FileProcessor` 接口（`ProcessFile(path) []Result` +
`HandleResult(Result)`）。`list` 用短路实现（首匹配即返回）；
`find/replace` 扫描整文件。

workers 默认 1（与 grep/sed 默认一致）。`-j 0` 走 `runtime.NumCPU()`，
`-j N` 走显式数量。

### stream

[stream](file:///opt/repos/gx/stream/stream.go) 统一 stdin/文件的读取
入口。`cut/trans/script` 用它读单输入（`ReadAll(paths)`），stdin 时返回
虚拟文件名 `<stdin>`。`find/list` 主要走文件树，但 grep 模式（无 path
参数）也通过它读 stdin。

## 各子命令要点

### find / list / replace

三者都走 `processor.Engine`：

- `find` 扫描每行，打印匹配行（含行号、彩色高亮）。
- `list` 每个文件首匹配即返回，只打印文件路径。
- `replace` 复用 search 流程；`-x` 时执行写回。

`replace` 的写回在 [cmd/replace/run.go](file:///opt/repos/gx/cmd/replace/run.go) 中，
默认 dry-run，加 `-x` 才落盘。

### rename

[cmd/rename/run.go](file:///opt/repos/gx/cmd/rename/run.go) 不走 processor
引擎——它处理的是"文件名"而不是"文件内容"。先 walk 一次得到候选集合，
按 pattern 生成新名，检查冲突，再原子重命名。冲突策略：
- 默认：拒绝整个批次
- `-f`：跳过冲突文件，继续
- `-i`：交互式（无 TTY 时退化为 `-f`）

### cut / trans / script

这三个是 `gx` 的 sed/awk 风格主力，都支持 stdin：

- `cut -f 1,3-5 -d ,`：按分隔符切字段，输出指定范围。`0` 是整个行
  （GNU cut 兼容）。
- `trans upper|squeeze|...`：每行做一次变换。多个变换按声明顺序串联。
- `script -e EXPR | -f FILE`：行模式（默认）/ 文件模式（`--whole`），
  Tengo 脚本编译一次，每行/每文件克隆 VM 执行。详见
  [script-api.md](file:///opt/repos/gx/docs/script-api.md)。

它们都不走 walker/processor，独立处理单输入流。

## 构建与版本嵌入

[Makefile](file:///opt/repos/gx/Makefile) 用 `-ldflags` 把版本号和
commit 注入 `main.version` / `main.commit`：

```
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
```

发布目标在 `bin/gx-<version>-<os>-<arch>`。`make one` 编译当前平台；
`make build` 出全部 5 个 target。`make upx` / `upxx` 二次压缩。
