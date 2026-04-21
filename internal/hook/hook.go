// Package hook runs a post-upload action on the remote host after clipsh has
// successfully written the file. Hooks are opt-in and described by a short
// string spec of the form "<kind>:<payload>".
//
// Supported kinds:
//
//	tmux:<session>         — type the uploaded path into session <session>
//	                         on the remote tmux server WITHOUT pressing
//	                         Enter. The path lands in the focused pane's
//	                         prompt; the user prefixes it with whatever
//	                         command the target tool expects (e.g. '@' in
//	                         Claude Code, ':e ' in vim) and submits. Safer
//	                         and tool-agnostic default.
//	tmux-submit:<session>  — like tmux: but also sends Enter after typing.
//	                         Submits the bare path as-is; useful only for
//	                         tools that treat a raw path as meaningful
//	                         input.
//	exec:<command>         — run an arbitrary remote command. The literal
//	                         token {path} in <command> is substituted with
//	                         the shell-quoted uploaded path.
//
// Unknown kinds are an error — hooks are user-typed, so fail loud.
package hook

import (
	"context"
	"fmt"
	"strings"

	"github.com/pajikos/clipsh/internal/transport"
)

// Run dispatches a hook spec against the given transport target and the
// uploaded remote path. An empty spec is a no-op.
func Run(ctx context.Context, opts transport.Options, spec, remotePath string) error {
	if spec == "" {
		return nil
	}
	kind, payload, ok := strings.Cut(spec, ":")
	if !ok {
		return fmt.Errorf("hook: spec %q missing ':'", spec)
	}
	switch kind {
	case "tmux":
		return runTmux(ctx, opts, payload, remotePath, false)
	case "tmux-submit":
		return runTmux(ctx, opts, payload, remotePath, true)
	case "exec":
		return runExec(ctx, opts, payload, remotePath)
	default:
		return fmt.Errorf("hook: unknown kind %q (want tmux|tmux-submit|exec)", kind)
	}
}

// BuildTmuxCommand is the shell command run on the remote to type the
// uploaded path into tmux session <session>. If submit is true, an Enter
// key is appended; otherwise the path is typed and left unsent for the
// user to prefix and submit. Exposed for testing.
func BuildTmuxCommand(session, remotePath string, submit bool) string {
	// send-keys -l (literal) types the text without key-name interpretation,
	// so a ';' in the path is safe. Enter is a separate call because mixing
	// literal text and named keys in one invocation is fragile.
	cmd := fmt.Sprintf(
		"tmux send-keys -l -t %s %s",
		shellQuote(session), shellQuote(remotePath),
	)
	if submit {
		cmd += fmt.Sprintf(" && tmux send-keys -t %s Enter", shellQuote(session))
	}
	return cmd
}

// BuildExecCommand substitutes {path} in the user-supplied command. Exposed
// for testing.
func BuildExecCommand(userCmd, remotePath string) string {
	return strings.ReplaceAll(userCmd, "{path}", shellQuote(remotePath))
}

func runTmux(ctx context.Context, opts transport.Options, session, path string, submit bool) error {
	if session == "" {
		return fmt.Errorf("hook: tmux hook needs a session name")
	}
	return transport.Exec(ctx, opts, BuildTmuxCommand(session, path, submit))
}

func runExec(ctx context.Context, opts transport.Options, userCmd, path string) error {
	if userCmd == "" {
		return fmt.Errorf("hook: exec hook needs a command")
	}
	return transport.Exec(ctx, opts, BuildExecCommand(userCmd, path))
}

// shellQuote wraps s in single quotes, escaping embedded single quotes with
// the standard '\'' trick. Safe to interpolate into a POSIX shell command.
func shellQuote(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '\'')
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\\', '\'', '\'')
			continue
		}
		out = append(out, s[i])
	}
	out = append(out, '\'')
	return string(out)
}
