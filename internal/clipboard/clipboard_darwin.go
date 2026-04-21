//go:build darwin

package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Read captures the macOS clipboard.
//
// Priority:
//  1. File URL (Finder Cmd+C, drag-drop) — read the actual file with its
//     real extension. Takes precedence because the clipboard may also
//     expose a rendered image representation of the file, which is
//     usually not what the user wants.
//  2. Image (pngpaste).
//  3. Text (pbpaste).
//
// When the image helper is missing AND text is empty, surface the tool-
// missing hint rather than "clipboard is empty" — the image is probably
// there, we just can't read it.
func Read() (*Content, error) {
	if f, err := readFileURL(); err == nil {
		return f, nil
	} else if !errors.Is(err, ErrEmpty) {
		return nil, err
	}

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

// readFileURL checks whether the clipboard holds a file reference (Finder
// Cmd+C, right-click Copy, drag-drop) and, if so, reads that file.
// Returns ErrEmpty when the clipboard does not hold a file URL.
//
// Finder's Copy writes a «class furl» (public.file-url) item to the
// pasteboard; a programmatic `POSIX file` sets an `alias` item. We check
// `clipboard info` first for either class name — AppleScript's coercions
// are too permissive (they will fabricate a fake path from plain text
// like "just some words" → "/just some words"), so an explicit class
// probe is the only reliable gate.
func readFileURL() (*Content, error) {
	info, err := exec.Command("osascript", "-e", "clipboard info").Output()
	if err != nil {
		return nil, ErrEmpty
	}
	infoStr := string(info)
	hasFurl := strings.Contains(infoStr, "furl")
	hasAlias := strings.Contains(infoStr, "alis")
	if !hasFurl && !hasAlias {
		return nil, ErrEmpty
	}

	const script = `try
	return POSIX path of (the clipboard as «class furl»)
on error
	try
		return POSIX path of (the clipboard as alias)
	on error
		return ""
	end try
end try`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, ErrEmpty
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return nil, ErrEmpty
	}
	data, err := os.ReadFile(path) // #nosec G304 - path comes from the user's own clipboard
	if err != nil {
		return nil, fmt.Errorf("read clipboard file %s: %w", path, err)
	}
	base := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	if ext == "" {
		ext = "bin"
	}
	stem := strings.TrimSuffix(base, "."+ext)
	return &Content{
		Bytes:     data,
		MIME:      "application/octet-stream",
		Extension: ext,
		Basename:  stem,
	}, nil
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
