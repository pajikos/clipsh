package transport

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildArgs_Minimal(t *testing.T) {
	got := BuildArgs(Options{Host: "user@host"}, "/tmp/x.png")
	want := []string{"user@host", "cat > '/tmp/x.png'"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildArgs minimal:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestBuildArgs_AllOptions(t *testing.T) {
	opts := Options{
		Host:     "user@host",
		Port:     2222,
		Identity: "/home/p/.ssh/id_ed25519",
		Jump:     "bastion",
		SSHOpts:  []string{"StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"},
	}
	got := BuildArgs(opts, "/tmp/x.png")
	want := []string{
		"-p", "2222",
		"-i", "/home/p/.ssh/id_ed25519",
		"-J", "bastion",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"user@host",
		"cat > '/tmp/x.png'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildArgs full:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestBuildArgs_QuotesPathWithSpaces(t *testing.T) {
	got := BuildArgs(Options{Host: "h"}, "/tmp/my file.png")
	last := got[len(got)-1]
	if last != "cat > '/tmp/my file.png'" {
		t.Errorf("path with space not single-quoted: %q", last)
	}
}

func TestBuildArgs_EscapesSingleQuote(t *testing.T) {
	// A path containing ' must be escaped so the remote shell sees it intact.
	got := BuildArgs(Options{Host: "h"}, "/tmp/it's.png")
	last := got[len(got)-1]
	if !strings.Contains(last, `'\''`) {
		t.Errorf("single quote not escaped in remote command: %q", last)
	}
}

func TestShellQuote_RoundTrip(t *testing.T) {
	cases := []string{
		"/tmp/plain.png",
		"/tmp/with space.png",
		`/tmp/it's.png`,
		"/tmp/$HOME.png", // must not expand
	}
	for _, in := range cases {
		q := shellQuote(in)
		if !strings.HasPrefix(q, "'") || !strings.HasSuffix(q, "'") {
			t.Errorf("shellQuote(%q) = %q, not single-quoted", in, q)
		}
	}
}
