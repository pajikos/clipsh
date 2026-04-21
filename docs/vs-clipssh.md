# clipsh vs clipssh

`clipssh` is a clean ~100-line bash tool that does exactly one thing: PNG
from the macOS pasteboard → remote `/tmp/clipboard-<epoch>.png` over SSH.
`clipsh` keeps the core ergonomic and expands the envelope.

## Feature matrix

| | clipssh | clipsh |
|---|---|---|
| PNG from clipboard | ✓ | ✓ |
| Other image formats (JPEG, HEIC, WebP) | ✗ | ✓ |
| Text clipboard | ✗ | ✓ |
| Explicit file argument | ✗ | ✓ |
| Custom SSH port | ✗ | ✓ (`--port`) |
| SSH identity | ✗ | ✓ (`--identity`) |
| ProxyJump | ✗ | ✓ (`--jump`) |
| Pass-through ssh options | ✗ | ✓ (`--ssh-opt KEY=VAL` repeatable) |
| Remote path template | fixed | ✓ with placeholders |
| Named profiles | ✗ | ✓ (v0.2.0) |
| Post-upload hook (tmux send-keys) | ✗ | ✓ (v0.2.0) |
| Dry-run | ✗ | ✓ |
| Pre-built cross-arch binaries | ✗ | ✓ |
| Homebrew tap | ✗ | ✓ |
| Tests + CI | ✗ | ✓ |
| Hosted docs | ✗ | ✓ |

## Migration

Drop-in replacement for a simple `clipssh user@host` invocation:

```sh
clipsh user@host
```

If you relied on the hardcoded `/tmp/clipboard-<epoch>.png` path, set it
explicitly:

```sh
clipsh -r '/tmp/clipboard-{timestamp}.png' user@host
```

## When to prefer clipssh

`clipssh` is worth keeping if you need a single-file bash script with zero
build toolchain — it audits in under a minute. `clipsh` is worth adopting
when you want more than PNG, non-standard ports, or a config-driven
workflow.
