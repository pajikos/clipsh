# Configuration

`clipsh` looks for a TOML config at `$XDG_CONFIG_HOME/clipsh/config.toml`
(defaulting to `~/.config/clipsh/config.toml`). All settings are optional;
every profile value is overridable on the command line.

A missing config file is not an error. Profiles are purely an ergonomic
shortcut — every field they set can also be passed as a flag.

## Example

```toml
default_profile = "dev"

[profile.dev]
host = "vscode@myvm.mesh"
port = 2222
identity = "~/.ssh/id_ed25519"
remote_path = "/home/vscode/repos/myproject/.clipboard.{ext}"
ssh_opts = ["StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"]
hook = "tmux:main"

[profile.prod]
host = "deploy@prod.example.com"
jump = "bastion.example.com"
remote_path = "/srv/inbox/{user}-{timestamp}.{ext}"
```

## Resolution order

**Profile selection:**

1. `--profile` / `-P` flag
2. `CLIPSH_PROFILE` environment variable
3. `default_profile` from config
4. No profile (pure-CLI mode)

**Per-field precedence** (for `host`, `port`, `identity`, `jump`,
`remote_path`, `hook`):

1. CLI flag
2. Positional `TARGET` (for host only)
3. Selected profile's value
4. Built-in default

Flags never mutate the config file; they only override values for the
single invocation.

Flags always override profile values for that invocation.

## Hooks

| Form | Effect |
|---|---|
| `tmux:<session>` | Run `tmux send-keys -t <session> '/image <path>' Enter` on the remote after upload. Types the `/image` command into an attached Claude Code or editor prompt. |
| `exec:<cmd>` | Run an arbitrary remote command. The literal token `{path}` in `<cmd>` is substituted with the shell-quoted uploaded path. |

Hooks run as a separate SSH session after the upload completes. A hook
failure does not fail the overall command — the file is already on the
remote regardless.

### Example: one-command screenshot to Claude Code

```toml
[profile.claude-dev]
host = "vscode@devbox.mesh"
port = 2222
remote_path = "/home/vscode/repos/myproject/.clipboard.{ext}"
hook = "tmux:main"
```

```sh
# Screenshot → Cmd+Shift+Ctrl+4 (to clipboard), then:
clipsh -P claude-dev
```

No path typing, no paste — the remote tmux session `main` receives
`/image /home/vscode/repos/myproject/.clipboard.png` + Enter and Claude
Code ingests the image immediately.
