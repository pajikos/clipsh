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
