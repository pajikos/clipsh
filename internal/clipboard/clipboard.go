// Package clipboard reads the local OS clipboard and writes text back to it.
//
// Read prefers image content over text: if the clipboard holds a PNG/JPEG/HEIC/WebP
// image, those bytes are returned with the matching MIME type. Otherwise the text
// contents are returned. Returns ErrEmpty when the clipboard is empty.
package clipboard

import "errors"

// Content is a single clipboard snapshot.
type Content struct {
	Bytes     []byte
	MIME      string // "image/png", "image/jpeg", "text/plain", ...
	Extension string // "png", "jpg", "txt", ...
}

// ErrEmpty is returned by Read when no content is available.
var ErrEmpty = errors.New("clipboard is empty")

// ErrToolMissing wraps missing-dependency errors (e.g. pngpaste, xclip).
type ErrToolMissing struct {
	Tool string
	Hint string
}

func (e *ErrToolMissing) Error() string {
	if e.Hint != "" {
		return "required tool not found: " + e.Tool + " (" + e.Hint + ")"
	}
	return "required tool not found: " + e.Tool
}
