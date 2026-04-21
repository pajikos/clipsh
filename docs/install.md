# Install

## Homebrew (macOS, Linux)

```sh
brew install pajikos/tap/clipsh
```

On macOS, install the optional `pngpaste` dependency if you want to send
screenshots directly from the pasteboard:

```sh
brew install pngpaste
```

## Go

```sh
go install github.com/pajikos/clipsh/cmd/clipsh@latest
```

## Pre-built binaries

Download a tarball from the [releases page](https://github.com/pajikos/clipsh/releases)
and extract the `clipsh` binary onto your `$PATH`.

## Clipboard prerequisites

| Platform | Tools used |
|---|---|
| macOS  | `pbpaste` (built-in), `pngpaste` (optional, for image clipboard) |
| Linux / X11 | `xclip` |
| Linux / Wayland | `wl-clipboard` (`wl-paste`, `wl-copy`) |

`clipsh` falls back to text if the image helper is missing — missing
`pngpaste` is non-fatal.
