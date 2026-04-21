# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-04-21

### Added
- **Named profiles** in `~/.config/clipsh/config.toml` (respects
  `$XDG_CONFIG_HOME`). A profile can set `host`, `port`, `identity`,
  `jump`, `remote_path`, `ssh_opts`, and `hook`. Select with
  `--profile / -P`, or `CLIPSH_PROFILE` env var, or `default_profile` in
  the config itself. Missing config file is not an error.
- **Post-upload hooks.** Two forms:
  - `tmux:<session>` — runs `tmux send-keys -t <session> '/image <path>' Enter`
    on the remote. Drives a Claude Code / editor prompt without a second
    paste step.
  - `exec:<cmd>` — runs an arbitrary remote command; `{path}` expands to
    the shell-quoted uploaded path.
  Hook failures are logged but do not fail the overall command, since the
  file is already uploaded.
- `--hook SPEC` CLI flag and `transport.Exec` for arbitrary SSH command
  execution (used by hooks).

### Changed
- Per-field precedence is now: CLI flag > positional (for target) >
  profile value > built-in default. All values remain overridable per
  invocation.

## [0.1.0] - 2026-04-21

### Added
- CLI entrypoint `clipsh [flags] [TARGET] [FILE]` with full SSH flag surface
  (`--port`, `--identity`, `--jump`, `--ssh-opt`, repeatable `-o`), remote
  path templating, `--dry-run`, `--no-copy`, `--source`, `--verbose`,
  `--version`, `--help`.
- Clipboard readers for macOS (pbpaste + optional pngpaste) and Linux
  (xclip for X11, wl-clipboard for Wayland), with graceful fallback to text
  when the image helper is missing on macOS.
- Path template engine (`internal/pathtmpl`) with placeholders `{timestamp}`,
  `{ext}`, `{basename}`, `{hostname}`, `{user}`, `{random}`. Unknown
  placeholders error at render time.
- SSH transport (`internal/transport`) that shells out to ssh(1), honoring
  the user's `~/.ssh/config`, ssh-agent, and ProxyJump configuration.
  Remote paths are single-quoted on the wire; embedded quotes escape safely.
- GoReleaser config producing static binaries for macOS and Linux
  (amd64 + arm64) and publishing a Homebrew formula to
  `pajikos/homebrew-tap`.
- GitHub Actions workflows: `ci` (go vet + test with race detector +
  golangci-lint), `release` (goreleaser on tag), `pages` (mkdocs-material
  deploy on docs changes).
- Documentation site at https://pajikos.github.io/clipsh/ covering install,
  usage, configuration (profiles/hooks design for v0.2), examples, and a
  feature comparison with clipssh.
