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
	"github.com/pajikos/clipsh/internal/config"
	"github.com/pajikos/clipsh/internal/hook"
	"github.com/pajikos/clipsh/internal/pathtmpl"
	"github.com/pajikos/clipsh/internal/transport"
)

// version is overwritten by the release build (goreleaser ldflags).
var version = "dev"

const defaultPathTmpl = "/tmp/clipsh-{timestamp}.{ext}"

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

type flags struct {
	profile    string
	host       string
	port       int
	identity   string
	jump       string
	sshOpts    stringList
	remoteTmpl string
	hook       string
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

	// Load config (missing file is fine — empty Config).
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		return 5
	}
	profileName := cfg.ResolveProfileName(f.profile)
	profile := cfg.ProfileOrEmpty(profileName)

	// Positional args: [TARGET] [FILE]. Disambiguation: a single positional
	// that looks like user@host (or an existing-local-file negation) is the
	// target; otherwise it's a file.
	var target, fileArg string
	tail := leftoverArgs
	switch len(tail) {
	case 0:
		// nothing — host comes from --host or profile
	case 1:
		switch {
		case f.host != "":
			// --host was explicit → positional is the file.
			fileArg = tail[0]
		case looksLikeSSHTarget(tail[0]):
			// Looks like user@host (or a bare non-file token) → target.
			// Overrides any profile.host.
			target = tail[0]
		default:
			fileArg = tail[0]
		}
	case 2:
		target = tail[0]
		fileArg = tail[1]
	default:
		fmt.Fprintln(stderr, "clipsh: too many positional arguments")
		return 2
	}
	// Merge: positional > --host > profile.host.
	if target == "" {
		target = firstNonEmpty(f.host, profile.Host)
	}
	if target == "" {
		fmt.Fprintln(stderr, "clipsh: no target specified. Pass user@host, --host, or configure a profile.")
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

	// 2. Render remote path — flag > profile > built-in default.
	tmpl := firstNonEmpty(f.remoteTmpl, profile.RemotePath, defaultPathTmpl)
	tctx := pathtmpl.Context{Ext: ext, Basename: basename}
	tctx.Auto()
	remotePath, err := pathtmpl.Render(tmpl, tctx)
	if err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		return 5
	}

	// 3. Resolve effective transport options — flag > profile.
	opts := transport.Options{
		Host:     target,
		Port:     firstNonZero(f.port, profile.Port),
		Identity: firstNonEmpty(f.identity, profile.Identity),
		Jump:     firstNonEmpty(f.jump, profile.Jump),
		SSHOpts:  mergeSSHOpts(f.sshOpts, profile.SSHOpts),
		Verbose:  f.verbose,
	}
	hookSpec := firstNonEmpty(f.hook, profile.Hook)

	// 4. Dry-run: print plan and stop.
	if f.dryRun {
		fmt.Fprintf(stdout, "would upload %d bytes (%s) to %s:%s\n",
			len(data), ext, target, remotePath)
		if hookSpec != "" {
			fmt.Fprintf(stdout, "would run hook: %s\n", hookSpec)
		}
		return 0
	}

	// 5. Upload.
	bgCtx, cancel := signalContext()
	defer cancel()

	if err := transport.Upload(bgCtx, opts, remotePath, bytes.NewReader(data)); err != nil {
		fmt.Fprintf(stderr, "clipsh: %v\n", err)
		return 4
	}

	// 6. Run hook (opt-in, best-effort — a hook failure is non-fatal since
	// the upload itself succeeded).
	if hookSpec != "" {
		if err := hook.Run(bgCtx, opts, hookSpec, remotePath); err != nil {
			fmt.Fprintf(stderr, "clipsh: hook %q failed: %v\n", hookSpec, err)
		}
	}

	// 7. Copy remote path to local clipboard (unless suppressed).
	if !f.noCopy {
		if err := clipboard.Copy(remotePath); err != nil {
			fmt.Fprintf(stderr, "clipsh: uploaded, but clipboard copy failed: %v\n", err)
			fmt.Fprintln(stdout, remotePath)
			return 0
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

	fs.StringVar(&f.profile, "P", "", "Config profile name")
	fs.StringVar(&f.profile, "profile", "", "Config profile name")
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
	fs.StringVar(&f.hook, "hook", "", "Post-upload hook: tmux:<session> | exec:<cmd>")
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
  -P, --profile NAME        Config profile name (overrides CLIPSH_PROFILE env)
  -H, --host TARGET         SSH target (alternative to positional)
  -p, --port N              SSH port
  -i, --identity PATH       SSH identity file
  -J, --jump HOST           ProxyJump host
  -o, --ssh-opt KEY=VAL     Extra SSH -o (repeatable)
  -r, --remote-path TMPL    Remote path template with {timestamp}, {ext},
                            {basename}, {hostname}, {user}, {random}
                            (default: /tmp/clipsh-{timestamp}.{ext})
      --hook SPEC           Post-upload hook: tmux:<session> | exec:<cmd>
                            (use {path} in exec for the uploaded path)
      --source auto|clip|file  Force content source (default: auto)
      --no-copy             Do not copy remote path to local clipboard
  -n, --dry-run             Print plan, do nothing
  -v, --verbose             Echo ssh invocation to stderr
  -V, --version             Print version
      --help                Show this help

Examples:
  clipsh user@myvm                       # send clipboard image or text
  clipsh user@myvm ./report.pdf          # send a file
  clipsh -p 2222 user@box                # non-standard SSH port
  clipsh -P dev                          # use 'dev' profile from ~/.config/clipsh/config.toml
  clipsh -P dev --hook tmux:main         # profile + ad-hoc hook override
  clipsh -n user@myvm                    # dry-run: show what would happen`)
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

// looksLikeSSHTarget is a heuristic: "user@host" or a single bare hostname
// token that doesn't exist as a local file.
func looksLikeSSHTarget(s string) bool {
	if strings.Contains(s, "@") {
		return true
	}
	if strings.ContainsAny(s, "/\\") {
		return false
	}
	if _, err := os.Stat(s); err == nil {
		return false
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

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(vs ...int) int {
	for _, v := range vs {
		if v != 0 {
			return v
		}
	}
	return 0
}

// mergeSSHOpts concatenates CLI-given -o options with profile-given ones.
// CLI first, then profile — duplicates are kept (ssh applies them left-to-
// right and later values win, matching our flag > profile precedence).
func mergeSSHOpts(cli []string, profile []string) []string {
	if len(cli) == 0 {
		return profile
	}
	if len(profile) == 0 {
		return cli
	}
	out := make([]string, 0, len(cli)+len(profile))
	out = append(out, profile...)
	out = append(out, cli...) // CLI wins via later position
	return out
}
