# clipsh

> Clipboard transport over SSH — screenshots, text, or any file, to a remote path.

`clipsh` takes what's on your local clipboard (or a file you name) and streams it
to a remote host over SSH, then copies the remote path back to your clipboard so
you can paste it straight into a terminal-based tool — Claude Code, `vim`, a
chat prompt.

```
$ clipsh user@myvm                 # send clipboard image (or text)
Uploaded: /tmp/clipsh-1713657600.png
Path copied to clipboard - paste it directly

$ clipsh user@myvm ./report.pdf    # send a file
Uploaded: /tmp/clipsh-1713657611.pdf
```

## Install

```sh
# Homebrew (macOS / Linux)
brew install pajikos/tap/clipsh

# Go
go install github.com/pajikos/clipsh/cmd/clipsh@latest

# Binaries
# see https://github.com/pajikos/clipsh/releases
```

## Why not clipssh?

| | clipssh | clipsh |
|---|---|---|
| PNG paste | ✓ | ✓ |
| Other image formats (JPEG, HEIC, WebP) | — | ✓ |
| Text clipboard | — | ✓ |
| Explicit file argument | — | ✓ |
| Custom SSH port | — | ✓ (`--port`) |
| ProxyJump / identity / ssh opts | — | ✓ |
| Remote path template | fixed | ✓ (`{timestamp}`, `{ext}`, `{basename}`…) |
| Named profiles (`~/.config/clipsh/config.toml`) | — | ✓ |
| Post-upload hook (auto `tmux send-keys` …) | — | ✓ (opt-in) |
| Cross-arch pre-built binaries | — | ✓ |
| Homebrew tap / `go install` | — | ✓ |
| Tests + CI | — | ✓ |

## Usage

```
clipsh [flags] [TARGET] [FILE]
```

See `clipsh --help` or the [docs](https://pajikos.github.io/clipsh/).

## License

MIT — see [LICENSE](./LICENSE).
