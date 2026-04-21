# Configuration

`clipsh` looks for a TOML config at `$XDG_CONFIG_HOME/clipsh/config.toml`
(defaulting to `~/.config/clipsh/config.toml`). All settings are optional;
every profile value is overridable on the command line.

A missing config file is not an error. Profiles are purely an ergonomic
shortcut — every field they set can also be passed as a flag.

## Recommended shape — defer to `~/.ssh/config`

`clipsh` does not re-implement SSH — it shells out to the system `ssh(1)`
binary. That means any alias defined in `~/.ssh/config` already works as a
`host` value in a profile, and you only need to put **clipsh-specific**
settings (remote path, hook) in the TOML.

**`~/.ssh/config`** — the authoritative connection details:

```sshconfig
Host dev.local
  HostName localhost
  Port 2222
  User me
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
  LogLevel ERROR

Host prod
  HostName prod.example.com
  User deploy
  ProxyJump bastion.example.com
```

**`~/.config/clipsh/config.toml`** — just the clipsh bits:

```toml
default_profile = "dev"

[profile.dev]
host = "dev.local"                                               # SSH alias
remote_path = "/home/me/.clipboard.{ext}"
hook = "tmux:main"

[profile.prod]
host = "prod"
remote_path = "/srv/inbox/{user}-{timestamp}.{ext}"
```

Benefits:

- No duplicated port/user/identity/options between two configs.
- `scp dev.local:path`, `ssh dev.local`, and `clipsh -P dev` all route the
  same way — a single source of truth.
- Adding a new host is one SSH config block + one TOML section, both short.

## Profile fields

| Field | CLI flag | Purpose |
|---|---|---|
| `host` | `-H`/`--host` (or positional) | SSH alias or `user@host` |
| `port` | `-p`/`--port` | SSH port (when not using an alias) |
| `identity` | `-i`/`--identity` | Path to private key |
| `jump` | `-J`/`--jump` | ProxyJump host |
| `ssh_opts` | `-o`/`--ssh-opt` (repeatable) | Extra `ssh -o KEY=VALUE` items |
| `remote_path` | `-r`/`--remote-path` | Template with `{timestamp}`, `{ext}`, `{basename}`, `{hostname}`, `{user}`, `{random}` |
| `hook` | `--hook` | Post-upload action (see Hooks below) |

Anything you can put in `~/.ssh/config` (`ProxyCommand`, `ControlMaster`,
`ForwardAgent`, `Include`, etc.) is inherited automatically when the host
value names an alias. Keep those in `~/.ssh/config`; keep clipsh-specific
behavior in the TOML.

## Inline example — when SSH config isn't an option

If you can't (or don't want to) edit `~/.ssh/config`, you can inline the
connection settings in the profile:

```toml
[profile.dev-inline]
host = "me@myvm.example.com"
port = 2222
identity = "~/.ssh/id_ed25519"
ssh_opts = ["StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"]
remote_path = "/home/me/.clipboard.{ext}"
hook = "tmux:main"
```

This works identically — it just duplicates what `~/.ssh/config` would
otherwise hold.

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

## Hooks

| Form | Effect |
|---|---|
| `tmux:<session>` | Type `/image <path>` into session `<session>` on the remote tmux server **without pressing Enter**. The text lands in the focused pane's prompt so you can review, edit, or add context before submitting. Safer default. |
| `tmux-submit:<session>` | Like `tmux:` but also sends `Enter` after typing. Use this only when the target tool won't take destructive action on implicit submit. |
| `exec:<cmd>` | Run an arbitrary remote command. The literal token `{path}` in `<cmd>` is substituted with the shell-quoted uploaded path. |

Hooks run as a separate SSH session after the upload completes. A hook
failure does not fail the overall command — the file is already on the
remote regardless.

### Example: one-command screenshot into a remote tmux session

`~/.ssh/config`:
```sshconfig
Host dev.local
  HostName localhost
  Port 2222
  User me
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
```

`~/.config/clipsh/config.toml`:
```toml
default_profile = "dev"

[profile.dev]
host = "dev.local"
remote_path = "/home/me/.clipboard.{ext}"
hook = "tmux:main"
```

```sh
# Screenshot → copy to clipboard, then:
clipsh
```

No path typing, no paste — the remote tmux session `main` sees
`/image /home/me/.clipboard.png` typed into its focused pane. Press
Enter yourself when you're ready to submit (or use `tmux-submit:main`
to auto-submit, if that's safe for the target tool).

!!! note "Requires a running tmux server"
    The hook fails if no tmux server is running on the remote (you'll see
    `error connecting to /tmp/tmux-<uid>/default`). Attach once with
    `ssh <host> -t tmux new -s <session>` to start the server; it persists
    across SSH sessions after that.
