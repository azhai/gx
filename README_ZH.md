# gre

用 Go 编写的快速文件搜索和批量重命名工具。

## 功能特性

- **replace**: 快速文件内容搜索和替换工具（灵感来自 ripgrep）
  - 多线程搜索和替换
  - 支持正则表达式
  - 彩色高亮输出
  - 行号显示
  - 通过 glob 模式过滤文件
  - 二进制文件检测
  - 干跑模式（执行前预览）

- **rename**: 批量文件重命名工具（灵感来自 f2）
  - 正则表达式匹配
  - 支持捕获组替换（$1, $2 等）
  - 干跑模式（执行前预览）
  - 冲突检测
  - 支持目录重命名
  - 强制模式解决冲突

## 安装

```bash
go install github.com/azhai/gre/cmd/replace@latest
go install github.com/azhai/gre/cmd/rename@latest
```

## 使用说明

### replace - 文件内容搜索和替换

```bash
# 基础搜索
replace "pattern"                    # 在当前目录搜索
replace "pattern" /path/to/dir       # 在指定目录搜索

# 搜索选项
replace -i "pattern"                 # 忽略大小写搜索
replace -n "pattern"                 # 显示行号（默认）
replace -N "pattern"                 # 隐藏行号
replace --no-color "pattern"         # 禁用彩色输出
replace -g "*.go" "func"             # 只在 Go 文件中搜索
replace -j 4 "pattern"               # 使用 4 个工作线程

# 替换（默认干跑模式）
replace "TODO" "FIXME"               # 预览：将 TODO 替换为 FIXME
replace "foo" "bar" -x               # 执行：将 foo 替换为 bar

# 示例
replace "TODO" src/                  # 在 src/ 目录搜索 TODO
replace -i "error" -g "*.go"         # 在 Go 文件中忽略大小写搜索 error
replace "TODO" src/ test/            # 在多个目录搜索 TODO
replace "foo" "bar" -x               # 将所有 'foo' 替换为 'bar'
```

### rename - 批量文件重命名

```bash
# 基础用法
rename "foo" "bar"                   # 将 'foo' 替换为 'bar'（干跑模式）
rename "foo" "bar" /path/to/dir      # 指定目录

# 选项
rename -f "pattern" "replace"        # 明确指定查找模式
rename -r "replace"                  # 明确指定替换字符串
rename -i "pattern" "replace"        # 忽略大小写匹配
rename -g "*.jpg" "pattern" "replace" # 按文件模式过滤
rename -x "pattern" "replace"        # 执行（默认：干跑模式）
rename --force "pattern" "replace"   # 强制重命名（即使有冲突）
rename -d "pattern" "replace"        # 包含目录
rename -F "pattern" "replace"        # 将模式视为字面字符串

# 示例
rename "foo" "bar"                   # 预览：将 'foo' 替换为 'bar'
rename "foo" "bar" -x                # 执行：将 'foo' 替换为 'bar'
rename -f "\.txt$" ".md" -x          # 将 .txt 扩展名改为 .md
rename -f "(\d+)" "prefix_$1" -x     # 在数字前添加前缀
rename -i "IMG" "img" -g "*.jpg"     # 将 jpg 文件的 IMG 转换为小写
rename -f "^" "2024_" -x             # 为所有文件添加 2024_ 前缀
```

## 从源码构建

```bash
git clone https://github.com/azhai/gre.git
cd gre
go build ./cmd/replace
go build ./cmd/rename
```

## 运行测试

```bash
go test ./... -v
```

## 许可证

MIT 许可证 - 详情请参见 [LICENSE](LICENSE) 文件。

## 致谢

- **replace** 灵感来自 [ripgrep](https://github.com/BurntSushi/ripgrep)
- **rename** 灵感来自 [f2](https://github.com/ivek/Vim)
