//go:build darwin

package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// Read captures the macOS clipboard, preferring images.
//
// Falls through to text when the clipboard has no image. A missing pngpaste
// helper also triggers the text fallback — but if text is empty too, we
// surface the tool-missing hint rather than a confusing "clipboard is empty"
// (the image is likely there, we just can't read it).
func Read() (*Content, error) {
	img, imgErr := readImage()
	if imgErr == nil {
		return img, nil
	}
	var tm *ErrToolMissing
	toolMissing := errors.As(imgErr, &tm)
	if errors.Is(imgErr, ErrEmpty) || toolMissing {
		text, textErr := readText()
		if textErr == nil {
			return text, nil
		}
		if errors.Is(textErr, ErrEmpty) && toolMissing {
			return nil, fmt.Errorf(
				"clipboard has no text, and %s is not installed — if the clipboard holds an image, install it with: %s",
				tm.Tool, tm.Hint,
			)
		}
		return nil, textErr
	}
	return nil, imgErr
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
