# Zellij post-upload hook

**Date:** 2026-05-13
**Status:** Approved, awaiting implementation plan

## Summary

Add `zellij:<session>` and `zellij-submit:<session>` to the post-upload hook
dispatcher in `internal/hook/hook.go`, parallel to the existing
`tmux:` / `tmux-submit:` pair. After clipsh writes the file to the remote
host, the hook types the uploaded path into the focused pane of the named
zellij session, optionally followed by Enter.

## Motivation

clipsh today drives tmux through `tmux send-keys -l`. Users running zellij
as their multiplexer — for example the devcontainers project at
`/Users/pavel/repos/devcontainers`, which installs zellij v0.44.2 alongside
tmux in every container and exposes both via `task tmux` / `task zellij` —
have no equivalent automation. They must paste the path manually.

A zellij hook restores parity. Both multiplexers run side by side in the same
containers, so the hook spec should let the user pick per upload without any
detection magic.

## Non-goals

- Tab or pane targeting beyond the focused pane of the named session.
- An auto-detecting `mux:<session>` kind that probes the remote.
- New transport, options, or flags. The dispatcher and the existing
  `--hook` / `hook =` surface absorb the new kinds.
- A clipboard-paste flavor (`action copy`); the hook's value is putting the
  path into the prompt, not on the remote clipboard.
- Layout-aware actions (new pane, new tab). Users who need that can already
  reach for `exec:<command>`.

## Hook spec surface

Two new kinds added to the package-level dispatcher:

| Spec | Effect |
| --- | --- |
| `zellij:<session>` | Type the uploaded path literally into the focused pane of zellij session `<session>`. No Enter. User prefixes with whatever the target tool wants (`@`, `:e `, …) and submits. |
| `zellij-submit:<session>` | Like `zellij:` but also sends Enter after typing. Submits the bare path; useful only for tools that treat a raw path as meaningful input. |

The spec parses with the existing `strings.Cut(spec, ":")` — kind on the
left, session name on the right. An empty session name is a user-facing
error (`hook: zellij hook needs a session name`), matching how the tmux
kinds handle a missing payload.

Precedence, dry-run preview, profile resolution, and CLI override behavior
are unchanged: the new kinds flow through the same `firstNonEmpty(f.hook,
profile.Hook)` path in `cmd/clipsh/main.go`.

## Remote command shape

A new exported helper builds the remote shell command, mirroring
`BuildTmuxCommand`:

```go
func BuildZellijCommand(session, remotePath string, submit bool) string
```

Output:

```sh
# zellij:<session>
zellij --session 'main' action write-chars '/tmp/foo.png'

# zellij-submit:<session>
zellij --session 'main' action write-chars '/tmp/foo.png' && \
  zellij --session 'main' action write 13
```

Reasoning:

- `action write-chars` writes text literally; no key-name interpretation, so
  paths containing `;`, spaces, or other shell-significant characters are
  safe. Direct counterpart to `tmux send-keys -l`.
- Enter is byte `13` (CR) — the same byte zellij itself uses for the Enter
  key. Splitting it into a second `action write 13` invocation avoids mixing
  literal text and named keys in one call, which mirrors the tmux helper's
  rationale for splitting Enter onto a separate `send-keys` invocation.
- Session name and path are wrapped with the existing `shellQuote` helper.
  No new quoting code.
- `--session <name>` is required because the SSH command runs outside any
  zellij client; without it, zellij refuses to dispatch the action.

`action write-chars` and `action write` have been stable since zellij 0.32;
the target environment (devcontainers, v0.44.2) is well past that floor.

## Dispatcher changes

In `internal/hook/hook.go`:

- Extend the package doc-comment with two new bullet rows for `zellij:` and
  `zellij-submit:` parallel to the existing tmux entries.
- Add two cases to the `switch kind` block in `Run`:
  ```go
  case "zellij":        return runZellij(ctx, opts, payload, remotePath, false)
  case "zellij-submit": return runZellij(ctx, opts, payload, remotePath, true)
  ```
- Update the `unknown kind` error string to
  `(want tmux|tmux-submit|zellij|zellij-submit|exec)`.
- Add `runZellij`, structured like `runTmux`: empty-session check, then
  `transport.Exec(ctx, opts, BuildZellijCommand(...))`.

No changes to `transport`, `config`, or `pathtmpl`. No new struct fields.

## Error handling

Identical to the tmux hook:

- Empty session payload → returned from `Run`, surfaced to the user as a
  hook error before any SSH happens.
- Remote `zellij` missing, session absent, or `write-chars` exiting non-zero
  → `transport.Exec` returns the error → `cmd/clipsh/main.go` logs
  `clipsh: hook "<spec>" failed: <err>` to stderr. The upload itself is
  still treated as successful and the remote path is still copied to the
  local clipboard. The existing comment at `main.go:159` already documents
  this "best-effort, non-fatal" stance; the new kinds inherit it without
  modification.

## Tests

Add to `internal/hook/hook_test.go`, parallel to the existing tmux tests:

- `TestBuildZellijCommand_NoSubmit` — asserts the exact command
  `zellij --session 'main' action write-chars '/tmp/x.png'` and that the
  output contains neither `write 13` nor `Enter`.
- `TestBuildZellijCommand_Submit` — asserts the full `&&` chain ending in
  `zellij --session 'main' action write 13`.
- `TestBuildZellijCommand_SingleQuoteInPath` — verifies the `'\''` escape
  in a path like `/tmp/it's.png`.
- `TestRun_ZellijNeedsSession` — `Run(..., "zellij:", ...)` returns an
  error containing `"session"`.
- `TestRun_ZellijSubmitNeedsSession` — same for `zellij-submit:`.

No remote integration test; the tmux hook doesn't have one either, and the
unit tests cover the only logic that's actually in clipsh's code (command
construction and dispatcher routing).

## CLI and documentation

- `cmd/clipsh/main.go`: extend the `--hook` flag description from the
  current `tmux:<session> | exec:<cmd>` to
  `tmux:<s> | tmux-submit:<s> | zellij:<s> | zellij-submit:<s> | exec:<cmd>`.
  Note this also fixes existing drift — `tmux-submit:` was previously
  supported by the dispatcher but missing from the help string. Also add
  one zellij example to the `Examples:` block (e.g.,
  `clipsh -P dev --hook zellij:main`).
- `docs/config.md`: add two rows to the Hooks table for the zellij kinds
  with the same wording style used for tmux, and a "Requires a running
  zellij session" admonition pointing to
  `ssh <host> -t zellij --session <s>` as the bootstrap command instead of
  the tmux equivalent.
- `docs/usage.md`: add one `--hook zellij:main` example to the section that
  already shows `tmux-submit:main`.
- `docs/examples.md`: short paragraph noting that zellij users substitute
  `zellij:` for `tmux:` in the existing examples; no separate page.
- `CHANGELOG.md`: one entry under the Unreleased / next-version heading.

No change to `docs/vs-clipssh.md` — the comparison there is per-feature, not
per-multiplexer.

## Risk

Low.

- The change is additive: two new `case` branches, one new helper, one new
  `runZellij`. No existing path is modified beyond the doc comment and the
  unknown-kind error message.
- Remote-side failure modes are bounded by the same SSH command return code
  path the tmux hook already uses.
- The shell-quoting helper is shared, well-tested, and unchanged.
- Behavior for users not running zellij is unchanged; the new kinds are
  only reachable when a user explicitly types `zellij:` or `zellij-submit:`
  in their hook spec.
