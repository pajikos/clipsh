// Package template renders remote-path templates with named placeholders.
//
// A template is a string that may contain {placeholder} tokens. Supported
// placeholders:
//
//	{timestamp}  Unix seconds (integer)
//	{ext}        File extension (e.g. "png", "txt"; no leading dot)
//	{basename}   Base filename without extension (for explicit-file uploads)
//	{hostname}   Local hostname
//	{user}       Local $USER / $LOGNAME
//	{random}     8 hex chars (cryptographically random)
//
// Unknown placeholders produce an error rather than silently expanding to
// empty — this protects users from typos in config.
package template

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

// Context holds substitution values. Fields that depend on the input (Ext,
// Basename) must be set explicitly by the caller; the rest have sensible
// defaults via Auto.
type Context struct {
	Timestamp int64
	Ext       string
	Basename  string
	Hostname  string
	User      string
	Random    string
}

// Auto fills in environment-derived defaults: Timestamp, Hostname, User,
// Random. Leaves Ext and Basename untouched so callers can set them.
func (c *Context) Auto() {
	if c.Timestamp == 0 {
		c.Timestamp = time.Now().Unix()
	}
	if c.Hostname == "" {
		if h, err := os.Hostname(); err == nil {
			c.Hostname = h
		}
	}
	if c.User == "" {
		if u := os.Getenv("USER"); u != "" {
			c.User = u
		} else {
			c.User = os.Getenv("LOGNAME")
		}
	}
	if c.Random == "" {
		var b [4]byte
		if _, err := rand.Read(b[:]); err == nil {
			c.Random = hex.EncodeToString(b[:])
		}
	}
}

var placeholder = regexp.MustCompile(`\{(\w+)\}`)

// Render expands placeholders in tmpl using ctx. Returns an error on unknown
// placeholders.
func Render(tmpl string, ctx Context) (string, error) {
	var firstErr error
	result := placeholder.ReplaceAllStringFunc(tmpl, func(match string) string {
		name := match[1 : len(match)-1]
		switch name {
		case "timestamp":
			return strconv.FormatInt(ctx.Timestamp, 10)
		case "ext":
			return ctx.Ext
		case "basename":
			return ctx.Basename
		case "hostname":
			return ctx.Hostname
		case "user":
			return ctx.User
		case "random":
			return ctx.Random
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("unknown placeholder: {%s}", name)
		}
		return match
	})
	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
}
