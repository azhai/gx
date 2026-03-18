# gx

用 Go 编写的快速文件搜索和批量重命名工具。

（本项目代码由AI生成，再经人工检查和修正）

## 功能特性

- **find**: 快速文件内容搜索工具（灵感来自 ripgrep）
  - 多线程搜索
  - 支持正则表达式
  - 彩色高亮输出
  - 行号显示
  - 通过 glob 模式过滤文件
  - 二进制文件检测

- **replace**: 快速文件内容搜索和替换工具（灵感来自 ripgrep）
  - 多线程搜索和替换
  - 支持正则表达式
  - 彩色高亮输出
  - 行号显示
  - 通过 glob 模式过滤文件
  - 二进制文件检测
  - 干跑模式（执行前预览）
  - 直观的参数解析（2参数智能模式）

- **rename**: 批量文件重命名工具（灵感来自 f2）
  - 正则表达式匹配
  - 支持捕获组替换（$1, $2 等）
  - 干跑模式（执行前预览）
  - 冲突检测
  - 支持目录重命名
  - 强制模式解决冲突
  - 大小写不敏感文件系统支持

## 项目结构

```
gx/
├── cmd/
│   ├── replace/       # 文件内容搜索和替换命令
│   └── rename/        # 批量文件重命名命令
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
  - Glob 模式过滤
  - 二进制文件检测

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
gx - A collection of file utilities

Usage: gx <command> [OPTIONS] [ARGS...]

Commands:
  find    Search for patterns in files (like grep)
  replace Search and replace text in files
  rename  Batch rename files

Use "gx <command> --help" for more information about a command.
```

### find - 文件内容搜索

```bash
# 基础搜索
gx find "pattern"                    # 在当前目录搜索
gx find "pattern" /path/to/dir       # 在指定目录搜索

# 搜索选项
gx find -F "pattern"                 # 将模式视为字面字符串
gx find -g "*.go" "func"             # 只在 Go 文件中搜索
gx find -i "pattern"                 # 忽略大小写搜索
gx find -j 4 "pattern"               # 使用 4 个工作线程
gx find -n "pattern"                 # 显示行号（默认）
gx find -N "pattern"                 # 隐藏行号
gx find --no-color "pattern"         # 禁用彩色输出

# 示例
gx find "TODO" src/                  # 在 src/ 目录搜索 TODO
gx find -i "error" -g "*.go"         # 在 Go 文件中忽略大小写搜索 error
gx find "TODO" src/ test/            # 在多个目录搜索 TODO
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
gx replace -F "pattern" "replace"    # 将模式视为字面字符串
gx replace -g "*.go" "func" "FUNC"   # 只在 Go 文件中替换
gx replace -i "pattern" "replace"    # 忽略大小写搜索
gx replace -j 4 "pattern" "replace"  # 使用 4 个工作线程
gx replace -n "pattern" "replace"    # 显示行号（默认）
gx replace -N "pattern" "replace"    # 隐藏行号
gx replace --no-color "pattern" "replace"  # 禁用彩色输出
gx replace -x                        # 执行替换（默认：干跑模式）

# 示例
gx replace "TODO" "FIXME"            # 预览：将 TODO 替换为 FIXME
gx replace "foo" "bar" -x            # 执行：将所有 'foo' 替换为 'bar'
gx replace -F "[test]" "demo" -x     # 替换字面字符串 '[test]'
gx replace -i "error" "warning" -g "*.go" -x  # 在 Go 文件中忽略大小写替换
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

## 从源码构建

```bash
git clone https://github.com/azhai/gx.git
cd gx
make build
```

或使用 Makefile 目标：

```bash
make gx        # 构建 gx 二进制文件
make clean     # 删除旧的二进制文件
make upx       # 构建并使用 upx 压缩
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
