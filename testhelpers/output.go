package testhelpers

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

	test testing.TB
}

func (o *Output) Close(code int, err error) {
	o.lk.Lock()
	defer o.lk.Unlock()

	o.Code = code
	o.Error = err
}

func (o *Output) ReadStderr() string {
	o.lk.Lock()
	defer o.lk.Unlock()

	return string(o.stderr)
}

func (o *Output) ReadStdout() string {
	o.lk.Lock()
	defer o.lk.Unlock()

	return string(o.stdout)
}

// ReadStdoutTrimNewlines does what it's called
func (o *Output) ReadStdoutTrimNewlines() string {
	return strings.Trim(o.ReadStdout(), "\n")
}

func (o *Output) AssertSuccess() *Output {
	o.test.Helper()
	assert.NoError(o.test, o.Error)
	oErr := o.ReadStderr()

	assert.Equal(o.test, o.Code, 0, oErr)
	assert.NotContains(o.test, oErr, "CRITICAL")
	assert.NotContains(o.test, oErr, "ERROR")
	assert.NotContains(o.test, oErr, "WARNING")
	return o

}

func (o *Output) AssertFail(err string) *Output {
	o.test.Helper()
	assert.NoError(o.test, o.Error)
	assert.Equal(o.test, 1, o.Code)
	assert.Empty(o.test, o.ReadStdout())
	assert.Contains(o.test, o.ReadStderr(), err)
	return o
}
