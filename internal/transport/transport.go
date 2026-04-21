// Package transport streams bytes to a remote path over SSH by shelling out
// to the system ssh(1) binary.
//
// Using the system ssh client is a deliberate choice: it respects the user's
// ~/.ssh/config (ProxyJump, IdentityFile, aliases), their ssh-agent, and any
// of the hundred edge cases that a reimplemented SSH client would mis-handle.
package transport

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// Options configures a single upload.
type Options struct {
	Host     string   // user@host or ssh_config alias. Required.
	Port     int      // 0 = default (22 / from ssh_config)
	Identity string   // path to private key; "" = default
	Jump     string   // ProxyJump host; "" = none
	SSHOpts  []string // additional "KEY=VALUE" items → "-o KEY=VALUE"
	Verbose  bool     // print the resolved ssh invocation to stderr
}

// Upload writes data to remotePath on opts.Host over SSH.
//
// The remote command is `cat > <shell-escaped-path>`; remotePath is single-
// quoted on the wire to prevent injection from path templates and profile
// data. Returns an error with the ssh stderr content for diagnosability.
func Upload(ctx context.Context, opts Options, remotePath string, data io.Reader) error {
	if opts.Host == "" {
		return fmt.Errorf("transport: Host is required")
	}
	args := BuildArgs(opts, remotePath)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = data
	// Capture stderr so we can surface the real ssh error on failure.
	var errBuf stderrBuffer
	cmd.Stderr = &errBuf

	if opts.Verbose {
		fmt.Fprintf(errBuf.Tee(), "+ ssh %v\n", args)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh upload failed: %w\n%s", err, errBuf.String())
	}
	return nil
}

// Exec runs an arbitrary command on opts.Host over SSH. The remoteCmd string
// is passed verbatim to ssh — callers are responsible for shell-quoting.
func Exec(ctx context.Context, opts Options, remoteCmd string) error {
	if opts.Host == "" {
		return fmt.Errorf("transport: Host is required")
	}
	args := BuildExecArgs(opts, remoteCmd)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var errBuf stderrBuffer
	cmd.Stderr = &errBuf

	if opts.Verbose {
		fmt.Fprintf(errBuf.Tee(), "+ ssh %v\n", args)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh exec failed: %w\n%s", err, errBuf.String())
	}
	return nil
}

// BuildArgs returns the argv for `ssh` given opts and remotePath. Exposed
// for testing.
func BuildArgs(opts Options, remotePath string) []string {
	args := sshBaseArgs(opts)
	args = append(args, opts.Host, "cat > "+shellQuote(remotePath))
	return args
}

// BuildExecArgs returns the argv for `ssh` that runs an arbitrary remote
// command. Exposed for testing.
func BuildExecArgs(opts Options, remoteCmd string) []string {
	args := sshBaseArgs(opts)
	args = append(args, opts.Host, remoteCmd)
	return args
}

// sshBaseArgs is the common prefix (port/identity/jump/opts) shared by the
// Upload and Exec codepaths.
func sshBaseArgs(opts Options) []string {
	var args []string
	if opts.Port != 0 {
		args = append(args, "-p", strconv.Itoa(opts.Port))
	}
	if opts.Identity != "" {
		args = append(args, "-i", opts.Identity)
	}
	if opts.Jump != "" {
		args = append(args, "-J", opts.Jump)
	}
	for _, o := range opts.SSHOpts {
		args = append(args, "-o", o)
	}
	return args
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes
// via the standard '\'' trick. Safe for interpolation into a POSIX shell
// command.
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

// stderrBuffer captures ssh's stderr so we can both tee it and include the
// final content in an error message. It avoids pulling in bytes.Buffer just
// so the test binary stays small, but a bytes.Buffer would work identically.
type stderrBuffer struct {
	data []byte
	tee  io.Writer
}

func (b *stderrBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if b.tee != nil {
		_, _ = b.tee.Write(p)
	}
	return len(p), nil
}

func (b *stderrBuffer) String() string { return string(b.data) }
func (b *stderrBuffer) Tee() io.Writer {
	if b.tee == nil {
		b.tee = stderrSink{}
	}
	return b.tee
}

// stderrSink writes to os.Stderr without pulling os into this file at package
// load; we still use exec.Command from os/exec but this keeps layering clear.
type stderrSink struct{}

func (stderrSink) Write(p []byte) (int, error) {
	return stderrWrite(p)
}
