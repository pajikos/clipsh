//go:build darwin

package clipboard

import (
	"bytes"
	"errors"
	"os/exec"
)

// Read captures the macOS clipboard, preferring images.
//
// Falls through to text when the clipboard has no image OR when pngpaste is
// not installed — text is always usable, so a missing optional helper must
// not block the common case.
func Read() (*Content, error) {
	img, err := readImage()
	if err == nil {
		return img, nil
	}
	var tm *ErrToolMissing
	if errors.Is(err, ErrEmpty) || errors.As(err, &tm) {
		return readText()
	}
	return nil, err
}

// Copy writes text to the macOS clipboard.
func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}

func readImage() (*Content, error) {
	if _, err := exec.LookPath("pngpaste"); err != nil {
		return nil, &ErrToolMissing{Tool: "pngpaste", Hint: "brew install pngpaste"}
	}
	// pngpaste writes PNG to stdout when given "-", and exits non-zero when
	// the clipboard has no image.
	out, err := exec.Command("pngpaste", "-").Output()
	if err != nil {
		return nil, ErrEmpty
	}
	if len(out) == 0 {
		return nil, ErrEmpty
	}
	return &Content{Bytes: out, MIME: "image/png", Extension: "png"}, nil
}

func readText() (*Content, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrEmpty
	}
	return &Content{Bytes: out, MIME: "text/plain", Extension: "txt"}, nil
}
