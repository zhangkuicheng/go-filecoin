package testhelpers

import (
	"io"
	"strings"
	"sync"
)

// Output manages running, inprocess, a filecoin command.
type Output struct {
	lk sync.Mutex
	// Input is the the raw input we got.
	Input string
	// Args is the cleaned up version of the input.
	Args []string
	// Code is the unix style exit code, set after the command exited.
	Code int
	// Error is the error returned from the command, after it exited.
	Error  error
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	stdout []byte
	Stderr io.ReadCloser
	stderr []byte
}

// Close will close the output
func (o *Output) Close(code int, err error) {
	o.lk.Lock()
	defer o.lk.Unlock()

	o.Code = code
	o.Error = err
}

// ReadStderr does that
func (o *Output) ReadStderr() string {
	o.lk.Lock()
	defer o.lk.Unlock()

	return string(o.stderr)
}

// ReadStdout does that
func (o *Output) ReadStdout() string {
	o.lk.Lock()
	defer o.lk.Unlock()

	return string(o.stdout)
}

// ReadStdoutTrimNewlines does what it's called
func (o *Output) ReadStdoutTrimNewlines() string {
	return strings.Trim(o.ReadStdout(), "\n")
}
