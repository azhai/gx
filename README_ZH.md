# gx

![gx](docs/gx-logo.svg)

用 Go 编写的快速文本处理工具：文件搜索（find/list）、批量重命名、以及类 Unix 文本过滤器（cut/trans/script）。

（本项目代码由AI生成，再经人工检查和修正）

## 功能特性

- **list**: 列出文件和目录（类似 `ls`）
  - 递归列举文件/目录，支持 glob、大小和时间过滤
  - 按名称/大小/修改时间/创建时间/路径排序，支持逆序
  - 多种输出格式：path（默认）、long（权限+大小+时间+路径）、name
  - 文件类型过滤（普通文件、目录、符号链接）
  - 递归深度控制
  - 大小过滤表达式（`>1M`、`<=100K`）
  - 时间过滤表达式（相对时间 `<=1h`、`>7d`，绝对时间 `>=2025-01-01`）

- **rename**: 批量文件重命名工具（灵感来自 f2）
  - 正则表达式匹配
  - 支持捕获组替换（$1, $2 等）
  - 干跑模式（执行前预览）
  - 冲突检测
  - 支持目录重命名
  - 强制模式解决冲突
  - 大小写不敏感文件系统支持

- **find**: 快速文件内容搜索工具（灵感来自 ripgrep）
  - 默认单进程，`-j N` / `-j 0`（全部核心）启用多进程
  - 支持正则表达式
  - 彩色高亮输出
  - 行号显示
  - 通过 glob 模式过滤文件
  - 二进制文件检测
  - 默认遵循 `.gitignore`，使用 `--all`/`-A` 搜索被忽略的文件
  - `-l`/`--files-with-matches` 模式：仅输出文件路径（类似 `grep -l`）

- **replace**: 快速文件内容搜索和替换工具（灵感来自 ripgrep）
  - 默认单进程，`-j N` / `-j 0`（全部核心）启用多进程
  - 支持正则表达式
  - 彩色高亮输出
  - 行号显示
  - 通过 glob 模式过滤文件
  - 二进制文件检测
  - 干跑模式（执行前预览）
  - 直观的参数解析（2参数智能模式）
  - 默认遵循 `.gitignore`，使用 `--all`/`-A` 搜索被忽略的文件

- **cut**: 从分隔文本中提取字段（POSIX `cut` 子集）
  - 仅 `-f` 字段模式（不支持 `-c`/`-b`）
  - 字段列表：`N`、`N-M`、`N-`（开放范围）、`-M`（1..M）、混合 `1,3-5,7-`
  - 自定义分隔符（`-d`，支持 `\t` `\n` `\\` 转义）
  - `-s` 跳过不含分隔符的行
  - `--output-delimiter` 重组输出
  - 从文件或 stdin 读取（无路径 / `-` → stdin）

- **trans**: 逐行文本变换
  - `upper`、`lower`、`trim`、`squeeze`（折叠连续空白）、`reverse`（按 rune 反转）
  - 从文件或 stdin 读取，输出到 stdout
  - 管道友好：`cat f | gx trans upper | gx trans trim`

- **script**: 用 [Tengo](https://github.com/d5/tengo) 脚本处理输入（sed/awk 风格）
  - 行模式（默认）：注入 `line`/`lineno`/`filename`
  - 文件模式（`--whole`）：注入 `content`/`filename`
  - 通过 `__out` 控制输出（string → 输出，`false` → 过滤）
  - 默认安全模块白名单；`--unsafe` 开启全部
  - 单次执行超时（`--timeout`）

## 项目结构

```
gx/
├── common/           # 共享工具（大小/时间表达式、gitignore）
├── cmd/
│   ├── find/          # 文件内容搜索命令（类似 grep）
│   ├── list/          # 列出文件和目录（类似 ls）
│   ├── replace/       # 文件内容搜索和替换命令
│   ├── rename/        # 批量文件重命名命令
│   ├── cut/           # 字段提取（cut -f 子集）
│   ├── trans/         # 逐行文本变换（upper/lower/...）
│   └── script/        # Tengo 脚本运行器（sed/awk 风格）
├── processor/         # 共享的遍历 + 工作池引擎
├── stream/            # 统一的 stdin / 文件输入
├── args/              # 共享参数解析功能
├── regex/             # 正则表达式匹配封装
├── walker/            # 并发文件系统遍历
├── README.md          # 英文文档
├── README_ZH.md       # 中文文档
└── LICENSE            # MIT 许可证
```

### 核心组件

- **args**: 提供通用参数解析功能，支持：
  - 简单的2参数模式（模式 + 路径 或 模式 + 替换字符串）
  - 短选项和长选项处理
  - 自动生成帮助信息

- **regex**: 封装 Go 的 regexp 包，提供：
  - 大小写不敏感匹配
  - 固定字符串支持
  - 捕获组替换

- **walker**: 并发文件系统遍历，支持：
  - 目录跳过（.git、node_modules 等）
  - `.gitignore` 感知遍历（惰性加载、多级继承）
  - Glob 模式过滤
  - 二进制文件检测
  - 递归深度控制

- **common**: 共享工具库：
  - `sizeexpr`: 大小过滤表达式解析器（`>1M`、`<=100K`）
  - `timeexpr`: 时间过滤表达式解析器（相对时间 `<=1h`、绝对时间 `>=2025-01-01`）
  - `gitignore`: .gitignore 规则解析和匹配器

- **processor**: 共享的文件处理引擎：
  - 基于 walker 输出的工作池流水线
  - `FileProcessor` 接口（ProcessFile + HandleResult），find/replace 复用

- **stream**: 统一的 stdin / 文件输入：
  - `IsStdin` / `OpenInput` / `ReadAll`
  - 虚拟文件名 `<stdin>`，用于匹配结果上报（供后续命令使用）

## 安装

```bash
go install github.com/azhai/gx@latest
```

或从源码构建：

```bash
git clone https://github.com/azhai/gx.git
cd gx
make build
```

## 使用说明

```
gx - A handy text-processing utility (sed/awk style)

Usage: gx <command> [OPTIONS] [ARGS...]

Commands:
  find     Search for patterns in files (like grep)
  list     List files containing matches (like grep -l)
  replace  Search and replace text in files
  rename   Batch rename files
  cut      Extract fields from delimited text (like cut -f)
  trans    Apply text transformations (upper/lower/trim/...)
  script   Run a Tengo script over input (sed/awk style)

Global Flags:
  -h, --help       Show help
  -V, --version    Show version

Use "gx <command> --help" for command-specific options.
```

### 退出码

| Code | 含义                                          |
|------|-----------------------------------------------|
| 0    | 成功，且有匹配 / 有文件被重命名               |
| 1    | 成功，但无匹配 / 无文件被重命名               |
| 2    | 错误（参数错误、IO 失败等）                   |

退出码遵循 grep 约定，使 `gx` 在 shell 管道和条件判断中自然组合：

```bash
if gx find "TODO" src/; then
  echo "存在 TODO"
fi
```

### 版本

```bash
gx --version      # 打印版本 + 提交号
gx -V             # 简写形式
```

版本和提交号在构建时通过 git 注入（见 `make one` / `make build`），因此二进制会报告它所基于的确切标签和提交。

### find - 文件内容搜索

```bash
# 基础搜索
gx find "pattern"                    # 在当前目录搜索
gx find "pattern" /path/to/dir       # 在指定目录搜索

# 搜索选项
gx find -A "pattern"                 # 搜索所有文件（忽略 .gitignore）
gx find -F "pattern"                 # 将模式视为字面字符串
gx find -g "*.go" "func"             # 只在 Go 文件中搜索
gx find -i "pattern"                 # 忽略大小写搜索
gx find -j 4 "pattern"               # 使用 4 个工作线程（默认 1，-j 0 = 全部核心）
gx find -l "pattern"                 # 仅输出包含匹配的文件路径（类似 grep -l）
gx find -n "pattern"                 # 显示行号（默认）
gx find -N "pattern"                 # 隐藏行号
gx find --no-color "pattern"         # 禁用彩色输出

# 示例
gx find "TODO" src/                  # 在 src/ 目录搜索 TODO
gx find -i "error" -g "*.go"         # 在 Go 文件中忽略大小写搜索 error
gx find "TODO" src/ test/            # 在多个目录搜索 TODO
gx find --all "debug" ./project      # 搜索被 .gitignore 忽略的文件
gx find -l -i "error" -g "*.go" ./src  # 列出包含 error 的 Go 文件（忽略大小写）
gx find -j 0 "pattern" ./large-project  # 多线程搜索（使用全部核心）
gx find -l "deprecated" -g "*.py" | xargs sed -i 's/deprecated/legacy/g'  # 管道组合
```

### replace - 文件内容搜索和替换

```bash
# 基础搜索和替换
gx replace "pattern" "replace"       # 预览替换（干跑模式）
gx replace "pattern" "replace" -x    # 执行替换

# 使用显式选项
gx replace -f "pattern" -r "replace" # 显式指定查找和替换
gx replace -f "TODO" -r "FIXME" -x   # 执行替换

# 替换选项
gx replace -A "pattern" "replace"    # 搜索所有文件（忽略 .gitignore）
gx replace -F "pattern" "replace"    # 将模式视为字面字符串
gx replace -g "*.go" "func" "FUNC"   # 只在 Go 文件中替换
gx replace -i "pattern" "replace"    # 忽略大小写搜索
gx replace -j 4 "pattern" "replace"  # 使用 4 个工作线程（默认 1，-j 0 = 全部核心）
gx replace -n "pattern" "replace"    # 显示行号（默认）
gx replace -N "pattern" "replace"    # 隐藏行号
gx replace --no-color "pattern" "replace"  # 禁用彩色输出
gx replace -x                        # 执行替换（默认：干跑模式）

# 示例
gx replace "TODO" "FIXME"            # 预览：将 TODO 替换为 FIXME
gx replace "foo" "bar" -x            # 执行：将所有 'foo' 替换为 'bar'
gx replace -F "[test]" "demo" -x     # 替换字面字符串 '[test]'
gx replace -i "error" "warning" -g "*.go" -x  # 在 Go 文件中忽略大小写替换
gx replace --all "old" "new" ./project  # 替换被 .gitignore 忽略的文件
gx replace "TODO" "FIXME" src/ test/     # 在多个目录替换
gx find -l "deprecated" -g "*.go" | xargs -I{} gx replace "deprecated" "legacy" -x "{}"  # 管道组合
```

### rename - 批量文件重命名

```bash
# 基础用法
gx rename "foo" "bar"                # 预览：将 'foo' 替换为 'bar'（干跑模式）
gx rename "foo" "bar" /path/to/dir   # 指定目录

# 使用显式选项
gx rename -f "pattern" -r "replace"  # 显式指定查找和替换
gx rename -f "foo" -r "bar" -x       # 执行重命名

# 选项
gx rename -d                         # 包含目录
gx rename -F "pattern" "replace"     # 将模式视为字面字符串
gx rename -g "*.jpg" "pattern" "replace"  # 按文件模式过滤
gx rename -i "pattern" "replace"     # 忽略大小写匹配
gx rename -x                         # 执行（默认：干跑模式）
gx rename --force                    # 强制重命名（即使有冲突）

# 示例
gx rename "foo" "bar"                # 预览：将 'foo' 替换为 'bar'
gx rename "foo" "bar" -x             # 执行：将 'foo' 替换为 'bar'
gx rename "\.txt$" ".md" -x          # 将 .txt 扩展名改为 .md
gx rename "(\d+)" "prefix_$1" -x     # 在数字前添加前缀
gx rename -i "IMG" "img" -g "*.jpg"  # 将 jpg 文件的 IMG 转换为小写
gx rename "^" "2024_" -x             # 为所有文件添加 2024_ 前缀
gx rename -F "[test]" "demo" -x      # 替换字面字符串 '[test]'
```

### list - 列出文件和目录

`list` 提供类 `ls` 的递归文件/目录列举功能，支持强大的过滤、排序和格式化选项。

```bash
# 基础列举
gx list                              # 递归列举当前目录所有文件
gx list /path/to/dir                 # 列举指定目录

# 过滤选项
gx list -g "*.go"                    # 按 glob 模式过滤
gx list -t f                         # 类型：f=文件, d=目录, l=符号链接, a=全部（默认）
gx list -S ">1M"                     # 大小过滤：>1M, <=100K, 1024 等
gx list -M "<=1h"                    # 修改时间过滤：相对时间（<=1h, >7d）或绝对时间（>=2025-01-01）
gx list --ctime "<=1d"               # 创建时间过滤：语法同修改时间

# 排序和格式
gx list -s name                      # 排序字段：name, size, mtime, ctime, path
gx list -s size -r                   # 逆序排序
gx list --format long                # 长格式：权限、大小、修改时间、路径
gx list --format name                # 仅文件名格式
gx list -L 2                         # 最大递归深度（0 = 无限制）

# 示例
gx list ./src                        # 递归列举 src/ 下所有文件
gx list -g "*.go" --size ">1K" --mtime "<=1d" ./src  # 今天修改过的 >1K 的 Go 文件
gx list --format long --sort size --reverse ./data    # 长格式按大小降序
gx list --type d --max-depth 2 ./project             # 列举深度 2 以内的目录
gx list -g "*.log" -S ">100M" /var/log               # 查找大于 100MB 的日志文件
gx list -M "<=1h" -t f ./src                         # 最近 1 小时修改过的文件
```

### cut - 从分隔文本中提取字段

`cut` 是 POSIX `cut` 的子集：仅支持 `-f`（字段模式）。从文件或
stdin 读取（无路径 / `-` → stdin），输出到 stdout。

```bash
# 字段选择
echo "a,b,c" | gx cut -f 2 -d ,                # → b
echo "a,b,c,d" | gx cut -f 2-3 -d ,            # → b,c
echo "a,b,c,d,e" | gx cut -f 2- -d ,           # → b,c,d,e（开放范围）
echo "a,b,c,d" | gx cut -f -2 -d ,            # → a,b（1..2）
echo "a,b,c,d,e" | gx cut -f 1,3-4 -d ,        # → a,c,d（混合）

# Tab 分隔（默认分隔符）
printf "x\ty\tz\n" | gx cut -f 2               # → y

# 跳过不含分隔符的行
printf "a,b\nno-delim\nc,d\n" | gx cut -f 1 -d , -s  # → a\nc

# 自定义输出分隔符
echo "a,b,c" | gx cut -f 1,3 -d , --output-delimiter=:  # → a:c

# 从文件读取
gx cut -f 1 -d , data.csv
```

字段规范语法（1-based，同 POSIX cut）：
- `N`       单字段
- `N-M`     闭区间
- `N-`      开放范围（N 到行尾）
- `-M`      1 到 M
- 混合      逗号分隔，如 `1,3-5,7-`

`-d` / `--output-delimiter` 支持转义：`\t` `\n` `\\`。

### trans - 文本变换

`trans` 对每行输入应用一种内置变换。从文件或 stdin 读取（无路径 /
`-` → stdin），输出到 stdout。

```bash
echo "hello" | gx trans upper                 # → HELLO
echo "  Hi  " | gx trans trim                 # → Hi
echo "  a   b   c  " | gx trans squeeze       # → a b c
echo "abc" | gx trans reverse                # → cba

# 从文件读取
gx trans lower names.txt

# 管道（链式变换）
cat file | gx trans trim | gx trans lower
```

可用变换：

| 变换      | 效果                                          |
|-----------|-----------------------------------------------|
| `upper`   | 全部转大写                                    |
| `lower`   | 全部转小写                                    |
| `trim`    | 去除首尾空白                                  |
| `squeeze` | 折叠连续空白为单个空格，并 trim               |
| `reverse` | 按 rune 反转字符串（Unicode 安全）            |

### script - 用 Tengo 脚本处理输入（sed/awk 风格）

`script` 用 [Tengo](https://github.com/d5/tengo) 脚本对输入文本做任意
变换，是 sed/awk 的轻量替代。脚本编译一次，按行或按文件执行，结果
写入 stdout，结果赋值给 `__out` 变量（string → 输出，`false`/`undefined`
→ 过滤，其他 → `fmt.Sprint`）。

**两种模式：**

- **行模式（默认）**：注入 `line`（string）、`lineno`（int，1-based）、
  `filename`（string，stdin 时为 `<stdin>`）。
- **文件模式（`--whole`）**：注入 `content`（整个文件内容 string）、
  `filename`（string）。

```bash
# -e：内联表达式；安全模块（text/json/...）自动预导入
echo "hi" | gx script -e 'text.to_upper(line)'          # → HI

# 过滤：丢弃偶数行
seq 4 | gx script -e 'lineno % 2 == 1 ? line : false'

# 用 fmt.sprintf 加行号
seq 3 | gx script -e 'fmt.sprintf("%d: %s", lineno, line)'

# 文件模式：统计换行数
printf 'a\nb\nc\n' | gx script --whole -e 'text.count(content, "\n")'  # → 3

# -f：加载完整脚本（自己控制 __out）
gx script --whole -f agg.tengo *.log
```

**安全模型** — 默认只开放纯计算类 Tengo 模块：`fmt`、`text`
（strings/strconv/regexp）、`json`、`math`、`times`、`base64`、`hex`、
`enum`。`--unsafe` 开启全部模块（`os`/`exec`/`file`/...）——仅在可信
输入下使用。

`-e` 会自动预导入所有安全模块，无需手写 `import`；`-f` 脚本需自行
`import`。

其他标志：`--timeout D`（单次执行超时，默认 `1s`）。

完整模块/API 参考：[docs/script-api.md](docs/script-api.md)。

## 从源码构建

```bash
git clone https://github.com/azhai/gx.git
cd gx
make one       # 构建本机二进制（bin/gx）
make build     # 交叉编译全平台
make release   # 交叉编译 + 生成 SHA256SUMS
```

交叉编译产物以 git 标签和短提交号命名：

```
bin/gx-<version>-<os>-<arch>     # 例：bin/gx-v0.2.0-darwin-arm64
```

支持的目标：`darwin-arm64`、`darwin-amd64`、`linux-arm64`、
`linux-amd64`、`windows-amd64`。

版本和提交号通过 `-ldflags` 注入，因此 `./bin/gx --version`
会报告它所基于的确切标签和提交。

其它 Makefile 目标：

```bash
make clean     # 删除旧的二进制文件
make upx       # 构建并使用 upx 压缩
make upxx      # 构建并使用 upx --ultra-brute 压缩
```

## 运行测试

```bash
go test ./... -v
```

## 许可证

MIT 许可证 - 详情请参见 [LICENSE](LICENSE) 文件。

## 致谢

- **find/replace** 灵感来自 [ripgrep](https://github.com/BurntSushi/ripgrep)
- **rename** 灵感来自 [f2](https://github.com/ayoisaiah/f2)
