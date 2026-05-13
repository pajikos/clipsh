package hook

import (
	"context"
	"strings"
	"testing"

	"github.com/pajikos/clipsh/internal/transport"
)

func TestBuildTmuxCommand_NoSubmit(t *testing.T) {
	got := BuildTmuxCommand("main", "/tmp/x.png", false)
	want := `tmux send-keys -l -t 'main' '/tmp/x.png'`
	if got != want {
		t.Errorf("\n  got:  %s\n  want: %s", got, want)
	}
	if strings.Contains(got, "Enter") {
		t.Errorf("no-submit build should not include Enter: %q", got)
	}
	if strings.Contains(got, "/image") {
		t.Errorf("tmux hook should not prepend /image: %q", got)
	}
}

func TestBuildTmuxCommand_Submit(t *testing.T) {
	got := BuildTmuxCommand("main", "/tmp/x.png", true)
	want := `tmux send-keys -l -t 'main' '/tmp/x.png' && tmux send-keys -t 'main' Enter`
	if got != want {
		t.Errorf("\n  got:  %s\n  want: %s", got, want)
	}
}

func TestBuildTmuxCommand_SingleQuoteInPath(t *testing.T) {
	got := BuildTmuxCommand("main", "/tmp/it's.png", false)
	// Single quote must be escaped as '\'' inside the single-quoted payload.
	if !strings.Contains(got, `'\''`) {
		t.Errorf("single quote not escaped: %q", got)
	}
}

func TestBuildExecCommand_PathSubstitution(t *testing.T) {
	got := BuildExecCommand("open {path} && echo done", "/tmp/x.png")
	want := `open '/tmp/x.png' && echo done`
	if got != want {
		t.Errorf("\n  got:  %s\n  want: %s", got, want)
	}
}

func TestBuildExecCommand_NoPlaceholder(t *testing.T) {
	// A command without {path} is legal — the user may not need the path.
	got := BuildExecCommand("notify-send hi", "/tmp/x.png")
	if got != "notify-send hi" {
		t.Errorf("unexpected substitution: %q", got)
	}
}

func TestRun_EmptySpecIsNoOp(t *testing.T) {
	if err := Run(context.Background(), transport.Options{}, "", "/tmp/x.png"); err != nil {
		t.Errorf("empty spec should be no-op, got %v", err)
	}
}

func TestRun_MissingColon(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "tmuxmain", "/p")
	if err == nil || !strings.Contains(err.Error(), "missing ':'") {
		t.Errorf("expected missing-colon error, got %v", err)
	}
}

func TestRun_UnknownKind(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "http:foo", "/p")
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("expected unknown-kind error, got %v", err)
	}
}

func TestRun_TmuxNeedsSession(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "tmux:", "/p")
	if err == nil || !strings.Contains(err.Error(), "session") {
		t.Errorf("expected session-required error, got %v", err)
	}
}

func TestRun_TmuxSubmitNeedsSession(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "tmux-submit:", "/p")
	if err == nil || !strings.Contains(err.Error(), "session") {
		t.Errorf("expected session-required error, got %v", err)
	}
}

func TestRun_ExecNeedsCommand(t *testing.T) {
	err := Run(context.Background(), transport.Options{Host: "h"}, "exec:", "/p")
	if err == nil || !strings.Contains(err.Error(), "command") {
		t.Errorf("expected command-required error, got %v", err)
	}
}

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
