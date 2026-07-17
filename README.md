# kvjson

`kv` 是一个很小的命令行工具：把紧凑的 `key:value` 输入转换成单行 JSON，方便在 `curl` 等命令中使用。

不带参数运行时，会打开一个 inline 交互编辑器：

```console
$ kv
kv · interactive JSON builder
› 1  key [name        ] │ value [Alice       ]
  2  key [role        ] │ value [admin       ]
tab/⇧tab fields · ←/→ boundaries · ↑/↓ rows · enter next/finish · ctrl+d done · esc cancel
{"name":"Alice","role":"admin"}
```

也可以继续直接传入紧凑语法：

```console
$ kv 'a:b, c:d'
{"a":"b","c":"d"}
```

在 sh、bash 或 zsh 中，用命令替换把结果传给 `curl`：

```bash
curl https://example.com/api \
  -H 'Content-Type: application/json' \
  --data "$(kv 'a:b, c:d')"
```

> 请保留 `"$(...)"` 外面的双引号，确保生成的 JSON 作为一个完整参数传递。

交互界面直接使用控制终端，最终 JSON 单独写入 stdout，因此无参数模式也可以用于命令替换：

```bash
curl https://example.com/api --data "$(kv)"
```

## 交互模式

直接运行 `kv`，或者运行 `kv --pretty`：

| 按键 | 行为 |
| --- | --- |
| `Tab` / `Shift+Tab` | 在 key、value 和相邻行之间移动 |
| `←` / `→` | 在文本内移动；到输入框边界时切换输入框或行 |
| `↑` / `↓` | 在不同行之间移动，保持当前 key/value 栏 |
| `Enter` | 校验当前行并创建/进入下一行；在空白行按下时生成 JSON |
| `Ctrl+D` | 校验全部行并立即生成 JSON |
| `Esc` / `Ctrl+C` | 取消，不输出 JSON，退出码为 130 |

value 可以是空字符串。完全清空一行会在生成时忽略该行；空 key、非空 value 以及重复 key 会在界面中提示错误。

## 语法

一个参数中可以写多个键值对，也可以传入多个参数：

```bash
kv 'name:Alice, role:admin'
kv 'name:Alice' 'role:admin'
```

- 未转义的逗号 `,` 分隔键值对。
- 每个键值对中第一个未转义的冒号 `:` 分隔 key 与 value。
- key 和 value 两边的空白会被清理。
- `\,` 表示逗号，`\:` 表示冒号，`\\` 表示反斜杠。
- 重复 key 会报错，不会静默覆盖。
- 当前版本的所有 value 都会编码为 JSON 字符串。

```console
$ kv 'message:hello\, world, url:https://example.com/a:b'
{"message":"hello, world","url":"https://example.com/a:b"}

$ kv --pretty '语言:中文, emoji:🐝'
{
  "emoji": "🐝",
  "语言": "中文"
}
```

## 本地开发

需要 Go 1.25 或更高版本：

```bash
make build
./bin/kv 'a:b, c:d'

make test
make check
```

交互界面使用 Bubble Tea 和 Bubbles，并继续以 `CGO_ENABLED=0` 构建 Release 二进制。

## CI 与 Release

`.github/workflows/ci.yml` 会在 push 和 pull request 时，在 Linux 与 macOS 上执行：

1. `gofmt` 检查；
2. `go vet`；
3. 带 race detector 的测试；
4. 编译 CLI。

推送符合 SemVer 的 tag 会触发 `.github/workflows/release.yml`：

```bash
git tag v0.1.0
git push origin v0.1.0
```

工作流会测试代码，然后交叉编译并发布以下压缩包：

- `kv_<version>_darwin_amd64.tar.gz`
- `kv_<version>_darwin_arm64.tar.gz`
- `kv_<version>_linux_amd64.tar.gz`
- `kv_<version>_linux_arm64.tar.gz`
- `checksums.txt`

每个压缩包包含 `kv`、`README.md` 和 `LICENSE`。

## Roadmap

- 显式的 JSON 类型值，例如数字、布尔值和 `null`；
- 嵌套 key；
- Homebrew 安装；
- 构建来源证明与签名。

## License

[MIT](LICENSE)
