//go:build linux

package clipboard

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Read captures the Linux clipboard, preferring images.
//
// readWayland and readX11 each try image MIMEs before text, so a plain-text
// clipboard on a system without an image-capable target returns text normally.
// A missing helper (xclip/wl-paste) is returned as ErrToolMissing — unlike
// macOS, Linux has no text-only fallback that bypasses those tools.
func Read() (*Content, error) {
	if onWayland() {
		return readWayland()
	}
	return readX11()
}

// Copy writes text to the Linux clipboard.
func Copy(text string) error {
	if onWayland() {
		return pipeTo("wl-copy", text)
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return pipeTo("xclip -selection clipboard", text)
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return pipeTo("xsel --clipboard --input", text)
	}
	return &ErrToolMissing{Tool: "xclip|xsel|wl-copy"}
}

func onWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

func readWayland() (*Content, error) {
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return nil, &ErrToolMissing{Tool: "wl-paste", Hint: "install wl-clipboard"}
	}
	types, err := exec.Command("wl-paste", "--list-types").Output()
	if err != nil {
		return nil, ErrEmpty
	}
	avail := string(types)

	for _, mime := range []string{"image/png", "image/jpeg", "image/webp", "image/heic"} {
		if strings.Contains(avail, mime) {
			out, err := exec.Command("wl-paste", "--type", mime, "--no-newline").Output()
			if err != nil || len(out) == 0 {
				continue
			}
			return &Content{Bytes: out, MIME: mime, Extension: extForMIME(mime)}, nil
		}
	}
	out, err := exec.Command("wl-paste", "--no-newline").Output()
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrEmpty
	}
	return &Content{Bytes: out, MIME: "text/plain", Extension: "txt"}, nil
}

func readX11() (*Content, error) {
	if _, err := exec.LookPath("xclip"); err != nil {
		return nil, &ErrToolMissing{Tool: "xclip", Hint: "install xclip"}
	}
	targets, err := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o").Output()
	if err != nil {
		return nil, ErrEmpty
	}
	avail := string(targets)

	for _, mime := range []string{"image/png", "image/jpeg", "image/webp", "image/heic"} {
		if strings.Contains(avail, mime) {
			out, err := exec.Command("xclip", "-selection", "clipboard", "-t", mime, "-o").Output()
			if err != nil || len(out) == 0 {
				continue
			}
			return &Content{Bytes: out, MIME: mime, Extension: extForMIME(mime)}, nil
		}
	}
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrEmpty
	}
	return &Content{Bytes: out, MIME: "text/plain", Extension: "txt"}, nil
}

func extForMIME(mime string) string {
	switch mime {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/heic":
		return "heic"
	}
	return "bin"
}

func pipeTo(cmdline, text string) error {
	parts := strings.Fields(cmdline)
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}
