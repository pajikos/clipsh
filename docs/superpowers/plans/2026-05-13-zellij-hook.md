# Zellij Post-Upload Hook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `zellij:<session>` and `zellij-submit:<session>` post-upload hook kinds to clipsh, parallel to the existing `tmux:` pair, so users running zellij as their remote multiplexer get the same one-keypress "upload + path-into-pane" flow as tmux users.

**Architecture:** Two new branches in the existing `internal/hook/hook.go` dispatcher. A new `BuildZellijCommand` helper composes a remote shell command that (1) preflights the session with `zellij list-sessions --no-formatting | awk ...` because zellij 0.44.2 exits 0 even when an action targets a missing session, then (2) runs `zellij --session <s> action write-chars <path>`, and optionally (3) sends Enter via `zellij --session <s> action write 13`. No new transport, options, or flags — the existing `--hook` / `hook =` surface absorbs the new kinds. Spec: `docs/superpowers/specs/2026-05-13-zellij-hook-design.md`.

**Tech Stack:** Go 1.x, standard library only. Test command: `go test -race -count=1 ./...` (also exposed via `task test`).

---

## File Structure

**Modify:**
- `internal/hook/hook.go` — add `BuildZellijCommand`, `runZellij`, two `case` branches in `Run`, extend package doc-comment, extend the `unknown kind` error string.
- `internal/hook/hook_test.go` — add six tests covering build output, single-quote escaping in path and session, and dispatcher session-required guards.
- `cmd/clipsh/main.go` — extend `--hook` flag description (lines 204, 245-246) and `Examples:` block (lines 254-260).
- `docs/config.md` — add two zellij rows to the Hooks table (after line 116); fix existing drift on the `tmux:<session>` row (it claims to type `/image <path>` but `internal/hook/hook.go` types only the path); update the running prose at lines 150-159; add a "running zellij session" admonition counterpart.
- `docs/usage.md` — add one `--hook zellij:main` example near line 111.
- `docs/examples.md` — add one sentence at line 14 noting that zellij users substitute `zellij:` for `tmux:`.
- `docs/index.md` — rewrite the post-upload hook bullet at lines 30-32 to cover both multiplexers and stop describing the typed content as `/image <path>`.
- `README.md` — extend the Post-upload hooks feature bullet (line 107) to list the two zellij forms.
- `CHANGELOG.md` — add an entry under the existing `## [Unreleased]` `### Added` block (around line 10).

**No file creation.** All work lives in already-existing files.

---

## Task 1: Add `BuildZellijCommand` helper (TDD)

**Files:**
- Test: `internal/hook/hook_test.go`
- Modify: `internal/hook/hook.go`

Builds the remote shell command without dispatcher wiring. The command must include the active-session preflight: `transport.Exec` only sees the exit status of the remote command, and zellij 0.44.2 exits 0 even when `action write-chars` targets a missing session (verified on 2026-05-13). The preflight forces a non-zero exit before the write action runs.

- [ ] **Step 1: Write failing tests for `BuildZellijCommand`**

Append to `internal/hook/hook_test.go`:

```go
func TestBuildZellijCommand_NoSubmit(t *testing.T) {
	got := BuildZellijCommand("main", "/tmp/x.png", false)
	want := `zellij list-sessions --no-formatting | awk -v s='main' '{ name=$0; sub(/ \[Created .*/, "", name); if (name == s && index($0, "(EXITED") == 0) found=1 } END { if (!found) { printf "zellij session not active: %s\n", s > "/dev/stderr"; exit 1 } }' && zellij --session 'main' action write-chars '/tmp/x.png'`
	if got != want {
		t.Errorf("\n  got:  %s\n  want: %s", got, want)
	}
	if strings.Contains(got, "write 13") {
		t.Errorf("no-submit build should not include write 13: %q", got)
	}
	if strings.Contains(got, "Enter") {
		t.Errorf("no-submit build should not include Enter: %q", got)
	}
}

func TestBuildZellijCommand_Submit(t *testing.T) {
	got := BuildZellijCommand("main", "/tmp/x.png", true)
	want := `zellij list-sessions --no-formatting | awk -v s='main' '{ name=$0; sub(/ \[Created .*/, "", name); if (name == s && index($0, "(EXITED") == 0) found=1 } END { if (!found) { printf "zellij session not active: %s\n", s > "/dev/stderr"; exit 1 } }' && zellij --session 'main' action write-chars '/tmp/x.png' && zellij --session 'main' action write 13`
	if got != want {
		t.Errorf("\n  got:  %s\n  want: %s", got, want)
	}
}

func TestBuildZellijCommand_SingleQuoteInPath(t *testing.T) {
	got := BuildZellijCommand("main", "/tmp/it's.png", false)
	// Single quote must be escaped as '\'' inside the single-quoted payload.
	if !strings.Contains(got, `'/tmp/it'\''s.png'`) {
		t.Errorf("single quote in path not escaped: %q", got)
	}
}

func TestBuildZellijCommand_SingleQuoteInSession(t *testing.T) {
	got := BuildZellijCommand("it's", "/tmp/x.png", false)
	// Session is interpolated in two positions: awk -v s=... and --session ...
	// Both must use the '\'' escape.
	if !strings.Contains(got, `awk -v s='it'\''s'`) {
		t.Errorf("single quote in session not escaped in awk -v: %q", got)
	}
	if !strings.Contains(got, `--session 'it'\''s'`) {
		t.Errorf("single quote in session not escaped in --session: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (function undefined)**

Run: `go test -run TestBuildZellijCommand ./internal/hook/ -v`
Expected: build error referencing `BuildZellijCommand` undefined.

- [ ] **Step 3: Implement `BuildZellijCommand` in `internal/hook/hook.go`**

Append after the existing `BuildExecCommand` definition (currently ending at line 77):

```go
// BuildZellijCommand is the shell command run on the remote to type the
// uploaded path into the focused pane of zellij session <session>.
//
// It runs a list-sessions preflight before the write action because zellij
// 0.44.2 exits 0 even when `action write-chars` targets a missing session;
// transport.Exec only sees the remote command's exit status, so the
// preflight forces a non-zero status when the session is missing or in the
// resurrectable (EXITED) state. If submit is true, an additional invocation
// writes byte 13 (CR) — the byte zellij itself uses for the Enter key.
// Exposed for testing.
func BuildZellijCommand(session, remotePath string, submit bool) string {
	qSession := shellQuote(session)
	preflight := "zellij list-sessions --no-formatting | awk -v s=" + qSession +
		` '{ name=$0; sub(/ \[Created .*/, "", name); if (name == s && index($0, "(EXITED") == 0) found=1 } END { if (!found) { printf "zellij session not active: %s\n", s > "/dev/stderr"; exit 1 } }'`
	write := fmt.Sprintf(
		"zellij --session %s action write-chars %s",
		qSession, shellQuote(remotePath),
	)
	cmd := preflight + " && " + write
	if submit {
		cmd += " && " + fmt.Sprintf("zellij --session %s action write 13", qSession)
	}
	return cmd
}
```

Notes for the implementer:
- Do NOT use a Go format verb for the awk message — concatenate the quoted session into the awk script with `+`, so the literal `%s` inside `printf "zellij session not active: %s\n"` survives untouched. The example above does this correctly.
- `shellQuote` is the existing helper at the bottom of `hook.go` (lines 95-107). Reuse it; do not add a second quoting routine.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestBuildZellijCommand ./internal/hook/ -v`
Expected: four PASS lines (`TestBuildZellijCommand_NoSubmit`, `_Submit`, `_SingleQuoteInPath`, `_SingleQuoteInSession`).

- [ ] **Step 5: Commit**

```bash
git add internal/hook/hook.go internal/hook/hook_test.go
git commit -m "$(cat <<'EOF'
feat(hook): add BuildZellijCommand helper

Composes a remote shell command that preflights `zellij list-sessions`
before running `action write-chars`. zellij 0.44.2 exits 0 even when an
action targets a missing session, so transport.Exec needs the preflight
to surface that as a hook error.
EOF
)"
```

---

## Task 2: Wire the dispatcher to the new kinds (TDD)

**Files:**
- Test: `internal/hook/hook_test.go`
- Modify: `internal/hook/hook.go`

Adds the two new `case` branches and a `runZellij` helper. Also updates the package doc-comment and the `unknown kind` error string so the user-facing surface stays honest.

- [ ] **Step 1: Write failing dispatcher tests**

Append to `internal/hook/hook_test.go`:

```go
func TestRun_ZellijNeedsSession(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "zellij:", "/p")
	if err == nil || !strings.Contains(err.Error(), "session") {
		t.Errorf("expected session-required error, got %v", err)
	}
}

func TestRun_ZellijSubmitNeedsSession(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "zellij-submit:", "/p")
	if err == nil || !strings.Contains(err.Error(), "session") {
		t.Errorf("expected session-required error, got %v", err)
	}
}
```

Also update the existing `TestRun_UnknownKind` if it asserts on the exact error string. (As of this writing it only asserts `"unknown kind"`, which still holds after the kind list grows — so leave it untouched. Verify before editing.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestRun_Zellij ./internal/hook/ -v`
Expected: both tests fail with an error containing `"unknown kind"` instead of `"session"` — that confirms the dispatcher does not yet know about the zellij kinds.

- [ ] **Step 3: Add `runZellij` and the two `case` branches**

Edit `internal/hook/hook.go`. In the `switch kind` block in `Run` (currently lines 43-52), insert between `case "tmux-submit":` and `case "exec":`:

```go
	case "zellij":
		return runZellij(ctx, opts, payload, remotePath, false)
	case "zellij-submit":
		return runZellij(ctx, opts, payload, remotePath, true)
```

Update the `default` branch's error message — old:

```go
return fmt.Errorf("hook: unknown kind %q (want tmux|tmux-submit|exec)", kind)
```

New:

```go
return fmt.Errorf("hook: unknown kind %q (want tmux|tmux-submit|zellij|zellij-submit|exec)", kind)
```

Append `runZellij` after the existing `runExec` definition (currently ending at line 91):

```go
func runZellij(ctx context.Context, opts transport.Options, session, path string, submit bool) error {
	if session == "" {
		return fmt.Errorf("hook: zellij hook needs a session name")
	}
	return transport.Exec(ctx, opts, BuildZellijCommand(session, path, submit))
}
```

- [ ] **Step 4: Update the package doc-comment**

The package doc at the top of `internal/hook/hook.go` lists supported kinds (currently lines 5-21). Add two new entries immediately after the existing `tmux-submit:` entry and before `exec:`, matching the indentation and tab alignment of the existing block:

```go
//	zellij:<session>       — type the uploaded path into the focused pane of
//	                         zellij session <session>, WITHOUT pressing
//	                         Enter. Same prefix-and-submit ergonomics as
//	                         tmux:.
//	zellij-submit:<session> — like zellij: but also sends Enter (byte 13)
//	                         after typing.
```

- [ ] **Step 5: Run all hook tests to verify they pass and nothing regressed**

Run: `go test -race -count=1 ./internal/hook/...`
Expected: all tests pass — the two new ones plus the eleven preexisting (six tmux tests, two exec tests, three Run tests).

- [ ] **Step 6: Run the full suite**

Run: `task test` (or `go test -race -count=1 ./...`)
Expected: all packages pass. `golangci-lint` is run by `task lint` and is reasonable to check too, but is not required to pass this task.

- [ ] **Step 7: Commit**

```bash
git add internal/hook/hook.go internal/hook/hook_test.go
git commit -m "$(cat <<'EOF'
feat(hook): dispatch zellij: / zellij-submit: hook kinds

Adds two case branches and a runZellij helper that delegates to
BuildZellijCommand. Extends the package doc-comment and the unknown-kind
error message to reflect the new surface.
EOF
)"
```

---

## Task 3: Extend the CLI help text

**Files:**
- Modify: `cmd/clipsh/main.go`

No tests — the CLI help string isn't covered by the existing test suite. The change is mechanical.

- [ ] **Step 1: Update the `--hook` flag description (around line 204)**

Replace:

```go
fs.StringVar(&f.hook, "hook", "", "Post-upload hook: tmux:<session> | exec:<cmd>")
```

With:

```go
fs.StringVar(&f.hook, "hook", "", "Post-upload hook: tmux:<s> | tmux-submit:<s> | zellij:<s> | zellij-submit:<s> | exec:<cmd>")
```

Note: this string is **not rendered** by the binary at runtime. `main.go:188` overrides `fs.Usage` with the hardcoded block in the `usage()` function (lines 226-260), so `flag.PrintDefaults` is never called and the per-flag short description is dead text. Updating it anyway because (a) it's misleading drift for anyone reading the source, and (b) `tmux-submit:` was missing from it. The Step 2 change below is the one that affects what users see.

- [ ] **Step 2: Update the long-form usage block (around lines 245-246)**

Replace:

```
      --hook SPEC           Post-upload hook: tmux:<session> | exec:<cmd>
                            (use {path} in exec for the uploaded path)
```

With:

```
      --hook SPEC           Post-upload hook:
                            tmux:<s> | tmux-submit:<s> |
                            zellij:<s> | zellij-submit:<s> |
                            exec:<cmd>  (use {path} in exec for the path)
```

- [ ] **Step 3: Add a zellij example to the `Examples:` block (around line 259)**

Insert a new line immediately after the existing `clipsh -P dev --hook tmux:main` example:

```
  clipsh -P dev --hook zellij:main       # ditto for zellij users
```

- [ ] **Step 4: Build to verify the file still compiles**

Run: `go build ./cmd/clipsh`
Expected: silent success. No binary needs to be kept (delete the `clipsh` binary it produces if anything).

- [ ] **Step 5: Spot-check the rendered usage block**

Run: `go run ./cmd/clipsh --help 2>&1 | grep -A 3 -- '--hook'`
Expected: the `--hook SPEC` long-form block (from Step 2) appears with `zellij:<s>` and `zellij-submit:<s>` listed. The output goes to stderr (`fs.SetOutput(stderr)` at `main.go:187`), so the `2>&1` is required — a stdout-only pipe captures nothing.

Also verify the (non-rendered) short flag description was updated in source:

Run: `grep -n 'Post-upload hook' cmd/clipsh/main.go`
Expected: the `fs.StringVar` line lists all four `tmux:`/`zellij:` variants.

- [ ] **Step 6: Commit**

```bash
git add cmd/clipsh/main.go
git commit -m "$(cat <<'EOF'
feat(cli): surface zellij hooks in --hook help and usage

Also fixes preexisting drift in the short --hook description, which did
not list tmux-submit: even though it has been dispatched since v0.2.0.
EOF
)"
```

---

## Task 4: Update `docs/config.md` Hooks section

**Files:**
- Modify: `docs/config.md`

Two changes bundled because they're adjacent and both touch the Hooks table:
1. Add `zellij:` and `zellij-submit:` rows.
2. Fix the `tmux:<session>` row — it claims to type `/image <path>`, but `internal/hook/hook.go` has typed only the path since the v0.2.0 → next release breaking change (see CHANGELOG `## [Unreleased] ### Changed (BREAKING)`).

- [ ] **Step 1: Fix the existing `tmux:<session>` row description**

Replace the row at line 115:

```markdown
| `tmux:<session>` | Type `/image <path>` into session `<session>` on the remote tmux server **without pressing Enter**. The text lands in the focused pane's prompt so you can review, edit, or add context before submitting. Safer default. |
```

With:

```markdown
| `tmux:<session>` | Type the uploaded path into session `<session>` on the remote tmux server **without pressing Enter**. The path lands in the focused pane's prompt; prefix it with whatever the target tool wants (`@` for Claude Code, `:e ` for vim, nothing for a bare shell) and submit yourself. Safer default. |
```

- [ ] **Step 2: Add two new rows for the zellij kinds**

Insert immediately after the `tmux-submit:<session>` row (currently line 116), before the `exec:<cmd>` row:

```markdown
| `zellij:<session>` | Type the uploaded path into the focused pane of zellij session `<session>` **without pressing Enter**. Parallel to `tmux:`. |
| `zellij-submit:<session>` | Like `zellij:` but also sends Enter (byte 13) after typing. |
```

- [ ] **Step 3: Fix the prose around the tmux example (lines 150-153)**

Replace:

```markdown
No path typing, no paste — the remote tmux session `main` sees
`/image /home/me/.clipboard.png` typed into its focused pane. Press
Enter yourself when you're ready to submit (or use `tmux-submit:main`
to auto-submit, if that's safe for the target tool).
```

With:

```markdown
No path typing, no paste — the remote tmux session `main` sees
`/home/me/.clipboard.png` typed into its focused pane. Prefix it with
whatever the pane expects (`@` for Claude Code, `:e ` for vim, nothing
for a bare shell), then press Enter (or use `tmux-submit:main` to
auto-submit, if that's safe for the target tool).
```

- [ ] **Step 4: Add a "running zellij session" admonition after the existing tmux one**

Insert immediately after the tmux admonition (ends at line 159) — keep one blank line between the two:

```markdown

!!! note "Requires a running zellij session"
    The hook fails (`zellij session not active: <name>`) if no zellij
    session of that name is live on the remote. Attach once with
    `ssh <host> -t zellij attach -c <session>` to create or resume it;
    background zellij daemons persist between SSH sessions the same way
    tmux servers do.
```

- [ ] **Step 5: Sanity-check the rendered docs locally (optional)**

Run: `task docs-serve &` then open `http://localhost:8000/config/`, visually confirm the Hooks table reads cleanly with the new rows and the corrected `tmux:` row. Kill the server when done. `task lint` does not lint markdown, so this manual pass is the only structural check.

- [ ] **Step 6: Commit**

```bash
git add docs/config.md
git commit -m "$(cat <<'EOF'
docs(config): document zellij hooks; fix tmux row drift

The `tmux:` row claimed to type `/image <path>`, but the dispatcher
stopped doing that in the next-release breaking change. Aligns the
description with current behavior and adds parallel rows for the two
zellij kinds plus a "running zellij session" admonition.
EOF
)"
```

---

## Task 5: Update the rest of the docs (`usage`, `examples`, `index`, README)

**Files:**
- Modify: `docs/usage.md`, `docs/examples.md`, `docs/index.md`, `README.md`

Four small touch-ups bundled into one commit because each is one or two lines and they all say the same thing: hooks now cover zellij.

- [ ] **Step 1: `docs/usage.md` — add a zellij example near line 111**

Find the block:

```markdown
Use a profile, override the hook ad-hoc:

​```sh
clipsh -P dev --hook tmux-submit:main
​```
```

Insert immediately below the closing fence, before the next prose:

```markdown
Same flow for zellij users:

​```sh
clipsh -P dev --hook zellij:main
​```
```

- [ ] **Step 2: `docs/examples.md` — extend the prose at line 12-14**

Replace:

```markdown
Then in the remote terminal: type `/image` (or whatever the app expects)
and paste the path from your clipboard. For a fully automated flow, pair
this with a `tmux:` hook — see [Configuration](config.md).
```

With:

```markdown
Then in the remote terminal: type `/image` (or whatever the app expects)
and paste the path from your clipboard. For a fully automated flow, pair
this with a `tmux:` or `zellij:` hook — see [Configuration](config.md).
```

- [ ] **Step 3: `docs/index.md` — rewrite the post-upload hook bullet (lines 30-32)**

Replace:

```markdown
- Optional **post-upload hook** — auto-drive `tmux send-keys` on the remote
  so the attached pane receives `/image <path>` (or any command) without a
  second paste.
```

With:

```markdown
- Optional **post-upload hook** — auto-drive `tmux send-keys` or
  `zellij action write-chars` on the remote so the attached pane receives
  the uploaded path (or any command) without a second paste.
```

- [ ] **Step 4: `README.md` — extend the Post-upload hooks feature bullet (line 107)**

Replace:

```markdown
- **Post-upload hooks:** `tmux:<session>` to type the path into a remote tmux pane, `tmux-submit:<session>` to also press Enter, `exec:<cmd>` for arbitrary remote commands.
```

With:

```markdown
- **Post-upload hooks:** `tmux:<session>` / `zellij:<session>` to type the path into a remote pane, `tmux-submit:<session>` / `zellij-submit:<session>` to also press Enter, `exec:<cmd>` for arbitrary remote commands.
```

- [ ] **Step 5: Commit**

```bash
git add docs/usage.md docs/examples.md docs/index.md README.md
git commit -m "$(cat <<'EOF'
docs: mention zellij alongside tmux in hook references

usage.md gets a parallel example; examples.md, index.md and the README
update their hook descriptions to cover both multiplexers.
EOF
)"
```

---

## Task 6: Update `CHANGELOG.md`

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add an `Added` bullet under the existing `## [Unreleased]` block**

The current `Unreleased` block at lines 8-15 already has an `### Added` heading. Append a new bullet under it:

```markdown
- `zellij:<session>` and `zellij-submit:<session>` post-upload hook
  kinds, parallel to the existing `tmux:` pair. Drives
  `zellij --session <s> action write-chars` over SSH, with a
  `list-sessions` preflight so a missing or `EXITED` session surfaces
  as a hook error (zellij 0.44.2 exits 0 from `action write-chars`
  against a missing session, so an explicit preflight is required).
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "$(cat <<'EOF'
docs(changelog): note zellij: / zellij-submit: hook kinds

EOF
)"
```

---

## Task 7: Final verification

- [ ] **Step 1: Full test suite**

Run: `task test`
Expected: all packages pass with the race detector clean.

- [ ] **Step 2: Lint**

Run: `task lint`
Expected: clean. If `golangci-lint` flags the new code, fix in place — do NOT silence rules with directives.

- [ ] **Step 3: Manual `--help` sanity check**

Run: `go run ./cmd/clipsh --help 2>&1 | grep -A 4 -- '--hook'`
Expected: the long-form usage block (from `cmd/clipsh/main.go:226-260`) lists the four `tmux:`/`zellij:` variants. The `2>&1` is required — the FlagSet's output is redirected to stderr at `main.go:187`. The short `fs.StringVar` description is dead text in this codebase (the custom `usage()` block doesn't call `flag.PrintDefaults`), so don't expect it in the rendered help.

- [ ] **Step 4: Manual dry-run sanity check**

Run:

```sh
printf hi > /tmp/clipsh-zellij-smoke.txt
go run ./cmd/clipsh -n --hook zellij:main user@nowhere /tmp/clipsh-zellij-smoke.txt
rm /tmp/clipsh-zellij-smoke.txt
```

Expected stdout: two lines — `would upload 2 bytes (txt) to user@nowhere:/tmp/clipsh-<epoch>.txt` and `would run hook: zellij:main`. A positional file is used because `readSource` (`cmd/clipsh/main.go:266-296`) accepts only file or clipboard sources — there is no stdin path.

- [ ] **Step 5: Verify spec coverage**

Reread `docs/superpowers/specs/2026-05-13-zellij-hook-design.md` quickly. Confirm every section's requirements were touched by Tasks 1-6:
- Hook spec surface (Task 2)
- Remote command shape, preflight, byte 13 Enter (Task 1)
- Dispatcher changes (Task 2)
- Error handling — relies on `transport.Exec`'s non-zero propagation, validated by Task 1's preflight and Task 2's empty-session guard
- Tests — six tests added across Tasks 1 and 2
- CLI and documentation — Tasks 3, 4, 5, 6

No remaining gaps.

---

## What's intentionally NOT in this plan

- No remote integration test. The existing tmux hook has none either, and the unit tests already cover every line of new Go code (command construction, dispatcher routing, empty-payload guard). Adding an SSH-spinning integration test would dwarf the change itself.
- No `mux:<session>` auto-detect kind, no tab/pane targeting, no clipboard-paste flavor, no new transport options. All listed as non-goals in the spec.
- No work in `internal/transport`, `internal/config`, or `internal/pathtmpl`. The new kinds use the same plumbing as `tmux:` and `exec:`.
