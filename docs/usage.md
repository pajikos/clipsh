# Usage

```
clipsh [flags] [TARGET] [FILE]
```

| Flag | Description |
|---|---|
| `-H, --host TARGET` | SSH target (alternative to positional) |
| `-p, --port N` | SSH port |
| `-i, --identity PATH` | SSH identity file |
| `-J, --jump HOST` | ProxyJump host |
| `-o, --ssh-opt KEY=VAL` | Extra `ssh -o` option (repeatable) |
| `-r, --remote-path TMPL` | Remote path template (default `/tmp/clipsh-{timestamp}.{ext}`) |
| `--source auto\|clip\|file` | Force content source |
| `--no-copy` | Do not copy remote path to local clipboard |
| `-n, --dry-run` | Print the plan and exit |
| `-v, --verbose` | Echo ssh invocation to stderr |
| `-V, --version` | Print version |

## Positional arguments

- **`TARGET`** — `user@host` or a `~/.ssh/config` alias.
- **`FILE`** — an explicit file to send instead of the clipboard.

Either may be omitted; if only one argument is present, `clipsh` infers
whether it is a target or a file based on whether a file by that name exists
locally. Pass both (`clipsh user@host file.pdf`) to be explicit.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic error |
| `2` | Usage error (wrong or missing flags) |
| `3` | Empty clipboard / empty file |
| `4` | SSH / network error |
| `5` | Config or template error |

## Path templates

Templates substitute named placeholders:

| Placeholder | Value |
|---|---|
| `{timestamp}` | Unix seconds |
| `{ext}` | File extension (`png`, `jpg`, `txt`, `pdf`, …) |
| `{basename}` | Filename stem when an explicit file is given |
| `{hostname}` | Local hostname |
| `{user}` | Local `$USER` |
| `{random}` | 8 random hex chars |

Unknown placeholders error at render time — typos never silently vanish.
