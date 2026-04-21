# Configuration

!!! note "Coming in v0.2.0"
    Named profiles land in the v0.2.0 release. This page documents the
    intended shape so downstream tooling can plan against it.

`clipsh` looks for a TOML config at `$XDG_CONFIG_HOME/clipsh/config.toml`
(defaulting to `~/.config/clipsh/config.toml`). All settings are optional;
every profile value is overridable on the command line.

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

1. `--host` / `-H` flag
2. Positional `TARGET`
3. `--profile` / `-P` flag
4. `CLIPSH_PROFILE` environment variable
5. `default_profile` from config
6. Error

Flags always override profile values for that invocation.

## Hooks

| Form | Effect |
|---|---|
| `tmux:<session>` | Run `tmux send-keys -t <session> '/image <path>' Enter` on the remote after upload |
| `exec:<cmd>` | Run arbitrary command on the remote; `{path}` expands to the uploaded path |

Hooks run over the same SSH connection used for the upload.
