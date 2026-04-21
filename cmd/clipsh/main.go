// clipsh — send clipboard content (or a file) to a remote host over SSH.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pajikos/clipsh/internal/clipboard"
	"github.com/pajikos/clipsh/internal/template"
	"github.com/pajikos/clipsh/internal/transport"
)

// version is overwritten by the release build (goreleaser ldflags).
var version = "dev"

const defaultPathTmpl = "/tmp/clipsh-{timestamp}.{ext}"

type stringList []string

func (s *stringList) String() string       { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error   { *s = append(*s, v); return nil }

type flags struct {
	host       string
	port       int
	identity   string
	jump       string
	sshOpts    stringList
	remoteTmpl string
	source     string
	noCopy     bool
	dryRun     bool
	verbose    bool
	showVer    bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	f := parseFlags(argv, stderr)
	if f == nil {
		return 2
	}
	if f.showVer {
		fmt.Fprintln(stdout, "clipsh", version)
		return 0
	}

	tail := leftoverArgs

	var target, fileArg string
	switch len(tail) {
	case 0:
		if f.host == "" {
			fmt.Fprintln(stderr, "clipsh: no target specified. Pass user@host or --host.")
			return 2
		}
	case 1:
		// one arg: if it looks like user@host and no --host given, treat as target.
		// otherwise treat as file.
		if f.host == "" && looksLikeSSHTarget(tail[0]) {
			target = tail[0]
		} else {
			fileArg = tail[0]
		}
	case 2:
		target = tail[0]
		fileArg = tail[1]
	default:
		fmt.Fprintln(stderr, "clipsh: too many positional arguments")
		return 2
	}
	if target == "" {
		target = f.host
	}
	if target == "" {
		fmt.Fprintln(stderr, "clipsh: no target specified")
		return 2
	}

	// 1. Obtain source bytes + extension.
	data, ext, basename, err := readSource(fileArg, f.source)
	if err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		if errors.Is(err, clipboard.ErrEmpty) {
			return 3
		}
		return 1
	}

	// 2. Render remote path.
	ctx := template.Context{Ext: ext, Basename: basename}
	ctx.Auto()
	remotePath, err := template.Render(pickTmpl(f.remoteTmpl), ctx)
	if err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		return 5
	}

	// 3. Dry-run: print plan and stop.
	if f.dryRun {
		fmt.Fprintf(stdout, "would upload %d bytes (%s) to %s:%s\n",
			len(data), ext, target, remotePath)
		return 0
	}

	// 4. Upload.
	bgCtx, cancel := signalContext()
	defer cancel()

	err = transport.Upload(bgCtx, transport.Options{
		Host:     target,
		Port:     f.port,
		Identity: f.identity,
		Jump:     f.jump,
		SSHOpts:  f.sshOpts,
		Verbose:  f.verbose,
	}, remotePath, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		return 4
	}

	// 5. Copy remote path to local clipboard (unless suppressed).
	if !f.noCopy {
		if err := clipboard.Copy(remotePath); err != nil {
			fmt.Fprintf(stderr, "clipsh: uploaded, but clipboard copy failed: %v\n", err)
			fmt.Fprintln(stdout, remotePath)
			return 0 // upload succeeded; treat clipboard failure as soft
		}
	}

	fmt.Fprintf(stdout, "Uploaded: %s\n", remotePath)
	if !f.noCopy {
		fmt.Fprintln(stdout, "Path copied to clipboard — paste it directly.")
	}
	return 0
}

// parseFlags returns nil on parse error (after printing usage).
func parseFlags(argv []string, stderr io.Writer) *flags {
	f := &flags{}
	fs := flag.NewFlagSet("clipsh", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { usage(fs.Output()) }

	fs.StringVar(&f.host, "H", "", "SSH target (alternative to positional arg)")
	fs.StringVar(&f.host, "host", "", "SSH target (alternative to positional arg)")
	fs.IntVar(&f.port, "p", 0, "SSH port")
	fs.IntVar(&f.port, "port", 0, "SSH port")
	fs.StringVar(&f.identity, "i", "", "SSH identity file")
	fs.StringVar(&f.identity, "identity", "", "SSH identity file")
	fs.StringVar(&f.jump, "J", "", "ProxyJump host")
	fs.StringVar(&f.jump, "jump", "", "ProxyJump host")
	fs.Var(&f.sshOpts, "o", "Extra ssh -o KEY=VALUE (repeatable)")
	fs.Var(&f.sshOpts, "ssh-opt", "Extra ssh -o KEY=VALUE (repeatable)")
	fs.StringVar(&f.remoteTmpl, "r", "", "Remote path template (default: "+defaultPathTmpl+")")
	fs.StringVar(&f.remoteTmpl, "remote-path", "", "Remote path template (default: "+defaultPathTmpl+")")
	fs.StringVar(&f.source, "source", "auto", "Force source: auto|clip|file")
	fs.BoolVar(&f.noCopy, "no-copy", false, "Do not copy remote path to local clipboard")
	fs.BoolVar(&f.dryRun, "n", false, "Print what would happen, do nothing")
	fs.BoolVar(&f.dryRun, "dry-run", false, "Print what would happen, do nothing")
	fs.BoolVar(&f.verbose, "v", false, "Print ssh invocation to stderr")
	fs.BoolVar(&f.verbose, "verbose", false, "Print ssh invocation to stderr")
	fs.BoolVar(&f.showVer, "V", false, "Print version and exit")
	fs.BoolVar(&f.showVer, "version", false, "Print version and exit")

	if err := fs.Parse(argv); err != nil {
		return nil
	}
	// Stash positional args globally for run() via leftover().
	leftoverArgs = fs.Args()
	return f
}

// leftoverArgs holds positional args after flag parsing. Package-level because
// parseFlags constructs the FlagSet locally and we still want run() to see
// what was left after flag extraction.
var leftoverArgs []string

func usage(w io.Writer) {
	fmt.Fprintln(w, `clipsh — clipboard transport over SSH

Usage:
  clipsh [flags] [TARGET] [FILE]

TARGET is user@host or an ssh_config alias. FILE, when given, is sent instead
of the clipboard contents.

Flags:
  -H, --host TARGET         SSH target (alternative to positional)
  -p, --port N              SSH port
  -i, --identity PATH       SSH identity file
  -J, --jump HOST           ProxyJump host
  -o, --ssh-opt KEY=VAL     Extra SSH -o (repeatable)
  -r, --remote-path TMPL    Remote path template with {timestamp}, {ext},
                            {basename}, {hostname}, {user}, {random}
                            (default: /tmp/clipsh-{timestamp}.{ext})
      --source auto|clip|file  Force content source (default: auto)
      --no-copy             Do not copy remote path to local clipboard
  -n, --dry-run             Print plan, do nothing
  -v, --verbose             Echo ssh invocation to stderr
  -V, --version             Print version
      --help                Show this help

Examples:
  clipsh user@myvm                  # send clipboard image or text
  clipsh user@myvm ./report.pdf     # send a file
  clipsh -p 2222 user@box           # non-standard SSH port
  clipsh -n user@myvm               # dry-run: show what would happen`)
}

// readSource returns the bytes to upload, the file extension (without dot),
// and a basename (for {basename} in the remote template). If fileArg is
// non-empty, reads that file. Otherwise reads the clipboard.
func readSource(fileArg, srcMode string) ([]byte, string, string, error) {
	if fileArg != "" && srcMode == "clip" {
		return nil, "", "", fmt.Errorf("--source=clip conflicts with positional file %q", fileArg)
	}
	if fileArg != "" || srcMode == "file" {
		if fileArg == "" {
			return nil, "", "", fmt.Errorf("--source=file requires a file argument")
		}
		data, err := os.ReadFile(fileArg)
		if err != nil {
			return nil, "", "", err
		}
		base := filepath.Base(fileArg)
		ext := strings.TrimPrefix(filepath.Ext(base), ".")
		if ext == "" {
			ext = "bin"
		}
		stem := strings.TrimSuffix(base, "."+ext)
		return data, ext, stem, nil
	}
	// clipboard path
	c, err := clipboard.Read()
	if err != nil {
		return nil, "", "", err
	}
	return c.Bytes, c.Extension, "clipboard", nil
}

func pickTmpl(user string) string {
	if user != "" {
		return user
	}
	return defaultPathTmpl
}

// looksLikeSSHTarget is a heuristic: "user@host" or a hostname with a dot or
// known alias pattern. Anything else is treated as a file.
func looksLikeSSHTarget(s string) bool {
	if strings.Contains(s, "@") {
		return true
	}
	// Treat as target if it's a single token with no path separators AND
	// not obviously a local file.
	if strings.ContainsAny(s, "/\\") {
		return false
	}
	if _, err := os.Stat(s); err == nil {
		return false // exists locally → file
	}
	return true
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}
