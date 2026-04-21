# clipsh

> Clipboard transport over SSH — screenshots, text, or any file, to a remote path.

`clipsh` streams what's on your local clipboard (or a file you name) to a
remote host over SSH, then copies the remote path back to your clipboard so
you can paste it straight into a terminal-based tool — `vim`, an interactive
AI prompt, a chat client running under tmux, anything that reads a file path.

## The 60-second demo

```console
$ clipsh user@myvm
Uploaded: /tmp/clipsh-1713657600.png
Path copied to clipboard — paste it directly.
```

```console
$ clipsh -p 2222 -o StrictHostKeyChecking=no me@dev.example.com ./report.pdf
Uploaded: /tmp/clipsh-1713657611.pdf
```

## What's different from clipssh

- Works with **images, text, and arbitrary files**, not just PNG.
- First-class SSH configuration: port, identity, ProxyJump, ssh options.
- Configurable **remote path templates** with `{timestamp}`, `{ext}`,
  `{basename}`, `{hostname}`, `{user}`, `{random}`.
- **Named profiles** in `~/.config/clipsh/config.toml`.
- Optional **post-upload hook** — auto-drive `tmux send-keys` on the remote
  so the attached pane receives `/image <path>` (or any command) without a
  second paste.
- Pre-built binaries for macOS + Linux, installable via Homebrew tap.

See [vs clipssh](vs-clipssh.md) for the full comparison.

## Next steps

- [Install](install.md)
- [Usage](usage.md)
- [Configuration](config.md)
- [Examples](examples.md)
