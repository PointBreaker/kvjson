# kvjson

`kv` 是一个很小的命令行工具：把紧凑的 `key:value` 输入转换成单行 JSON，方便在 `curl` 等命令中使用。

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

需要 Go 1.24 或更高版本：

```bash
make build
./bin/kv 'a:b, c:d'

make test
make check
```

本项目只使用 Go 标准库，并以 `CGO_ENABLED=0` 构建 Release 二进制。

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
