package transport

import "os"

// stderrWrite is a tiny indirection that keeps the testable parts of this
// package free of direct os.Stderr references.
func stderrWrite(p []byte) (int, error) {
	return os.Stderr.Write(p)
}
