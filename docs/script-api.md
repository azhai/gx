# gx script — Tengo 脚本命令参考

`gx script` 用 [Tengo](https://github.com/d5/tengo) 脚本对输入文本做任意
变换，定位为 sed/awk 的轻量替代。脚本编译一次，按行或按文件执行，结果
写入 stdout。

## 用法

```
gx script (-e EXPR | -f FILE) [--whole] [--unsafe] [--timeout D] [FILE...]
```

- `-e EXPR`：内联表达式，自动包装为 `__out := (EXPR)` 并预导入安全模块。
- `-f FILE`：从文件加载完整脚本（自己控制 `__out`）。
- `--whole`：文件模式（默认行模式）。
- `--unsafe`：额外开启 `os` 和 `rand` 模块（默认禁用）。
- `--timeout D`：单次执行超时（Go duration 字符串，默认 `1s`）。

无 FILE 或 FILE 为 `-` 时读 stdin。

## 两种模式

### 行模式（默认）

每行执行一次脚本，注入：

| 变量       | 类型   | 含义                  |
|------------|--------|-----------------------|
| `line`     | string | 当前行的内容（无 `\n`）|
| `lineno`   | int    | 1-based 行号          |
| `filename` | string | 文件名，stdin 为 `<stdin>` |

```bash
$ printf 'hello\nworld\n' | gx script -e 'text.to_upper(line)'
HELLO
WORLD

# 过滤：偶数行跳过
$ printf 'a\nb\nc\nd\n' | gx script -e 'lineno % 2 == 1 ? line : false'
a
c
```

### 文件模式（`--whole`）

每个文件执行一次脚本，注入：

| 变量       | 类型   | 含义                         |
|------------|--------|------------------------------|
| `content`  | string | 文件完整内容（含换行符）     |
| `filename` | string | 文件名                       |

```bash
$ printf 'a\nb\nc\n' > /tmp/lines.txt
$ gx script --whole -e 'text.count(content, "\n")' /tmp/lines.txt
3
```

## 输出约定（`__out`）

脚本把结果赋值给 `__out` 变量，gx 根据其类型决定行为：

| `__out` 类型          | 行为                              |
|-----------------------|-----------------------------------|
| `string`              | 输出 + 换行                       |
| `undefined` / `false` | 跳过该行/文件（用于过滤）        |
| 其他类型              | `fmt.Sprint(__out)` + 换行        |

行模式下不输出即等价于过滤；文件模式下 `__out = false` 跳过整个文件。

## 安全模型

默认仅导入纯计算类 Tengo 标准库模块：

| 模块     | 说明                                                     |
|----------|----------------------------------------------------------|
| `fmt`    | 格式化输出（`fmt.sprintf` 等）                           |
| `text`   | strings/strconv/regexp 包装（`to_upper`、`contains`、`atoi`、`re_match`…）|
| `json`   | JSON 编解码                                              |
| `math`   | 数学函数                                                 |
| `times`  | 时间相关                                                 |
| `base64` | Base64 编解码                                            |
| `hex`    | 十六进制编解码                                           |
| `enum`   | 枚举/迭代工具（map/filter/reduce…）                      |

> **注意**：Tengo 没有独立的 `strings` / `strconv` / `regexp` / `time`
> 模块——它们都并入了 `text`（snake_case API）和 `times`。

`--unsafe` 在白名单基础上额外开启 `os` 和 `rand`，可执行外部命令、读写
环境变量、生成随机数。仅在可信输入下使用。

`-e` 表达式会自动预导入全部安全模块，无需手写
`text := import("text")`。`-f` 脚本需自行 `import`。

## 关键 Tengo API 速查（`text` 模块）

```tenego
text.to_upper(s)              # 大写
text.to_lower(s)              # 小写
text.trim_space(s)            # 去首尾空白
text.contains(s, sub)         # 是否包含子串
text.has_prefix(s, p)         # 前缀判断
text.has_suffix(s, s)         # 后缀判断
text.index(s, sub)            # 子串位置
text.count(s, sub)            # 子串计数
text.replace(s, old, new, n)  # 替换 n 次
text.split(s, sep)            # 分割，返回数组
text.join(arr, sep)           # 数组拼接
text.substr(s, lo, hi)        # 子串
text.atoi(s)                  # 字符串 → int
text.itoa(i)                  # int → 字符串
text.re_match(pat, s)         # 正则匹配 → bool
text.re_find(pat, s, n)       # 正则查找
text.re_replace(pat, s, repl) # 正则替换
text.re_split(pat, s, n)      # 正则分割
text.re_compile(pat)          # 编译正则，返回 Regexp 对象
```

## 示例

### 行模式

```bash
# 加感叹号
echo hi | gx script -e 'line + "!"'
# → hi!

# 偶数行大写
seq 4 | gx script -e 'lineno % 2 == 0 ? text.to_upper(line) : line'
# 1
# 2  → TWO
# 3
# 4  → FOUR

# 只输出匹配 TODO 的行（grep 模拟）
gx script -e 'text.contains(line, "TODO") ? line : false' file.go

# 行首加行号
seq 3 | gx script -e 'fmt.sprintf("%d: %s", lineno, line)'
# 1: 1
# 2: 2
# 3: 3
```

### 文件模式

```tengo
// agg.tengo: 统计文件行数 + 字符数
__out := fmt.sprintf("%s: %d lines, %d chars",
    filename,
    text.count(content, "\n"),
    len(content))
```

```bash
$ gx script --whole -f agg.tengo *.go
```

## 退出码

| Code | 含义                                    |
|------|-----------------------------------------|
| 0    | 成功，且至少输出了一行/一个文件         |
| 1    | 成功，但无输出（所有行/文件被过滤）     |
| 2    | 参数错误、编译错误、运行时错误、IO 错误 |

## 超时

单次执行超过 `--timeout` 即中断，整个命令退出 2。默认 1s；处理大文件
或脚本中存在循环时按需调大：

```bash
gx script --timeout 30s --whole -f agg.tengo big.log
```
