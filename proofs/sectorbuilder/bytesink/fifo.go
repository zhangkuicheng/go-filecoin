// +build !windows

package bytesink

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

type fifoByteSink struct {
	file *os.File
	path string
}

var _ ByteSink = (*fifoByteSink)(nil)

// Open prepares the sink for writing by opening the backing FIFO file. Open
// will block until someone opens the FIFO file for reading.
func (s *fifoByteSink) Open() error {
	file, err := os.OpenFile(s.path, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return errors.Wrap(err, "failed to open pipe")
	}

	s.file = file

	return nil
}

// Remove deletes the underlying FIFO file.
func (s *fifoByteSink) Remove() error {
	return os.Remove(s.path)
}

// Write writes the provided buffer to the underlying file.
func (s *fifoByteSink) Write(buf []byte) (int, error) {
	return s.file.Write(buf)
}

// Close ensures that the underlying file is closed.
func (s *fifoByteSink) Close() (retErr error) {
	return s.file.Close()
}

// NewFifo creates a FIFO pipe and returns the address of a fifoByteSink, which
// satisfies the ByteSink interface. The FIFO pipe is used to stream bytes to
// rust-fil-proofs from Go during the piece-adding flow. Writes to the pipe are
// buffered automatically by the OS; the size of the buffer varies.
func NewFifo(path string) (ByteSink, error) {
	err := syscall.Mkfifo(path, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "mkfifo failed")
	}

	return &fifoByteSink{
		path: path,
	}, nil
}
