# Usage

```
clipsh [flags] [TARGET] [FILE]
```

## Flags

| Flag | Description |
|---|---|
| `-P, --profile NAME` | Config profile name (overrides `CLIPSH_PROFILE` env var) |
| `-H, --host TARGET` | SSH target (alternative to positional) |
| `-p, --port N` | SSH port |
| `-i, --identity PATH` | SSH identity file |
| `-J, --jump HOST` | ProxyJump host |
| `-o, --ssh-opt KEY=VAL` | Extra `ssh -o` option (repeatable) |
| `-r, --remote-path TMPL` | Remote path template (default `/tmp/clipsh-{timestamp}.{ext}`) |
| `--hook SPEC` | Post-upload hook — see [Hooks](config.md#hooks) |
| `--source auto\|clip\|file` | Force content source (default `auto`) |
| `--no-copy` | Do not copy the remote path to the local clipboard |
| `-n, --dry-run` | Print the plan and exit |
| `-v, --verbose` | Echo the ssh invocation to stderr |
| `-V, --version` | Print version |
| `--help` | Show help |

Every flag overrides the corresponding profile value for that single
invocation; see [Configuration](config.md) for how profiles are
resolved.

## Positional arguments

- **`TARGET`** — `user@host` or a `~/.ssh/config` alias. May be omitted
  when `--host` or a profile provides one.
- **`FILE`** — an explicit local file to send instead of the clipboard.

When only one positional is given, `clipsh` infers which role it plays:
an argument that looks like an SSH target (`user@host`, or a bare word
that isn't a local file) is treated as `TARGET`; otherwise as `FILE`.
Pass both to be explicit.

## Clipboard source priority (macOS)

1. **File reference** — Finder `Cmd+C` or right-click Copy. Reads the
   original file from disk; the real extension and basename are used.
2. **Image** — a screenshot on the pasteboard (`pngpaste` required;
   `brew install pngpaste`). Produces `.png`.
3. **Text** — `pbpaste` output. Produces `.txt`.

If a file is copied that's also rendered to an image on the pasteboard
(common for image files), clipsh prefers the file reference — so a
copied `report.pdf` uploads as a PDF rather than a rasterized PNG.

## Clipboard source priority (Linux)

- **Wayland:** `wl-paste --list-types` dictates preference. Image MIMEs
  (`image/png`, `image/jpeg`, `image/webp`, `image/heic`) win over text.
- **X11:** same strategy via `xclip -t TARGETS -o`.

Missing helpers (`xclip`, `wl-clipboard`) surface a clear
"install with …" hint.

## Path templates

Templates substitute named placeholders:

| Placeholder | Value |
|---|---|
| `{timestamp}` | Unix seconds |
| `{ext}` | File extension (`png`, `jpg`, `txt`, `pdf`, `md`, …) |
| `{basename}` | Original filename stem — the real name when a file is on the clipboard or named on the CLI; `"clipboard"` for raw image / text paste |
| `{hostname}` | Local hostname |
| `{user}` | Local `$USER` |
| `{random}` | 8 random hex characters |

Unknown placeholders produce an error at render time — typos never
silently vanish into an empty string.

The parent directory of the rendered path is `mkdir -p`'d on the
remote before the upload, so templates pointing at not-yet-existing
subdirectories just work.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic error |
| `2` | Usage error (wrong or missing flags) |
| `3` | Empty clipboard / empty file |
| `4` | SSH / network error |
| `5` | Config or template error |

## Examples

Minimal — send the clipboard to a host:

```sh
clipsh user@dev.example.com
```

Send a file, preserving its real name via `{basename}`:

```sh
clipsh -r '/srv/inbox/{basename}.{ext}' user@dev.example.com ./report.pdf
# → /srv/inbox/report.pdf
```

Use a profile, override the hook ad-hoc:

```sh
clipsh -P dev --hook tmux-submit:main
```

Force the source to text (useful when the clipboard also has a file
reference you want to ignore):

```sh
clipsh --source clip user@host
```

Preview without uploading:

```sh
clipsh -n -P dev
# would upload 108 bytes (png) to dev.local:/tmp/clipboard.png
# would run hook: tmux:main
```
