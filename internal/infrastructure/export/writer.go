package export

import (
	"bytes"
	"errors"
)

// BufferWriter collects exported content in memory.
type BufferWriter struct {
	buffer bytes.Buffer
}

func NewBufferWriter() *BufferWriter { return &BufferWriter{} }

// Write appends bytes and always succeeds.
func (w *BufferWriter) Write(p []byte) (int, error) {
	return w.buffer.Write(p)
}

// Bytes returns the collected content.
func (w *BufferWriter) Bytes() []byte { return w.buffer.Bytes() }

// Reset clears the buffer.
func (w *BufferWriter) Reset() { w.buffer.Reset() }

var ErrEmptyExport = errors.New("export buffer is empty")
