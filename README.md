# clipsh

> Clipboard transport over SSH — screenshots, text, or any file, to a remote path.

[![ci](https://github.com/pajikos/clipsh/actions/workflows/ci.yml/badge.svg)](https://github.com/pajikos/clipsh/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/pajikos/clipsh?sort=semver)](https://github.com/pajikos/clipsh/releases)
[![license](https://img.shields.io/github/license/pajikos/clipsh)](./LICENSE)
[![go](https://img.shields.io/github/go-mod/go-version/pajikos/clipsh)](./go.mod)

`clipsh` streams whatever is on your local clipboard — a screenshot, a
Finder-copied file, selected text, or an explicit file you name — to a
remote host over SSH, then copies the remote path back to your clipboard.
Paste that path straight into a terminal app running on the remote (an
editor, an interactive AI prompt, a chat client) without juggling `scp`
commands.

<!-- TODO: replace with an asciinema / GIF demo of the end-to-end flow -->

## Install

```sh
# Homebrew (macOS, Linux)
brew install pajikos/tap/clipsh

# Go
go install github.com/pajikos/clipsh/cmd/clipsh@latest
```

Pre-built binaries for every release: <https://github.com/pajikos/clipsh/releases>.

On macOS, add `pngpaste` to unlock screenshot support from the pasteboard:

```sh
brew install pngpaste
```

## Usage

Screenshot in clipboard → upload + drop the remote path on your local clipboard:

```console
$ clipsh user@dev.example.com
Uploaded: /tmp/clipsh-1713657600.png
Path copied to clipboard — paste it directly.
```

Finder-copied file → uploaded with its real name and extension:

```console
$ clipsh user@dev.example.com
Uploaded: /tmp/report.pdf
Path copied to clipboard — paste it directly.
```

Arbitrary local file:

```console
$ clipsh user@dev.example.com ./design.pdf
Uploaded: /tmp/clipsh-1713657611.pdf
```

Dry-run before uploading:

```console
$ clipsh -n -r '/uploads/{hostname}-{random}.{ext}' user@dev.example.com
would upload 2481 bytes (png) to user@dev.example.com:/uploads/mymac-a1b2c3d4.png
```

## One-keypress workflow

Define the host once in `~/.ssh/config`:

```sshconfig
Host dev.local
  HostName localhost
  Port 2222
  User me
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
```

A matching profile in `~/.config/clipsh/config.toml`:

```toml
default_profile = "dev"

[profile.dev]
host = "dev.local"
remote_path = "/tmp/{basename}.{ext}"
hook = "tmux:main"
```

Now `clipsh` with no arguments uploads the clipboard to `dev.local`, and
the `tmux:main` hook types the resulting path into tmux session `main`
on the remote — ready for you to prefix with whatever the pane expects
(`@path` for Claude Code, `:e path` for vim, a bare path for a shell)
and press Enter.

## Features

- **Source auto-detection:** Finder files / drag-drop, PNG/JPEG/HEIC/WebP screenshots, text — in that priority.
- **SSH via `~/.ssh/config` aliases** — port, identity, ProxyJump, options all resolve normally.
- **Remote path templates** with `{timestamp}`, `{ext}`, `{basename}`, `{hostname}`, `{user}`, `{random}`.
- **Named profiles** in `~/.config/clipsh/config.toml`; override any field with a flag.
- **Post-upload hooks:** `tmux:<session>` to type the path into a remote tmux pane, `tmux-submit:<session>` to also press Enter, `exec:<cmd>` for arbitrary remote commands.
- **`mkdir -p` on the remote** before writing, so templates pointing at subdirs just work.
- **Dry-run** (`-n`), **verbose** (`-v`), **`--no-copy`** (skip the clipboard return trip).
- Pre-built static binaries for macOS and Linux, amd64 + arm64.

## Documentation

Full configuration reference, hook specs, and more examples:
<https://pajikos.github.io/clipsh/>

## Contributing

Bug reports and feature requests: <https://github.com/pajikos/clipsh/issues>.

Pull requests are welcome. Please run `task test` and `task lint` before
submitting; CI runs the same commands on macOS and Linux.

## License

MIT — see [LICENSE](./LICENSE).
