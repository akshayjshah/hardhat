// Package hhlog is a printf-style logging package. It accepts both standard
// and debugging messages; standard messages are written immediately to
// console, but debugging messages are buffered internally. If the application
// encounters an error, it can annotate the error with the accumulated debug
// logs.
package hhlog

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

// Logger is a simple printf-style logger.
type Logger struct {
	mu  sync.Mutex
	buf *bytes.Buffer
	out io.Writer
}

// New returns a logger that writes to standard out.
func New() *Logger {
	return &Logger{
		buf: bytes.NewBuffer(nil),
		out: os.Stdout,
	}
}

// NewNop returns a logger that never writes any output.
func NewNop() *Logger {
	return &Logger{out: ioutil.Discard}
}

// Debugf buffers a debugging message.
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.buf == nil {
		return
	}
	l.mu.Lock()
	l.buf.WriteString("[DEBUG] ")
	fmt.Fprintf(l.buf, format, args...)
	l.buf.WriteString("\n")
	l.mu.Unlock()
}

// Printf writes out the supplied message immediately.
func (l *Logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	fmt.Fprintf(l.out, format, args...)
	fmt.Fprintf(l.out, "\n")
	l.mu.Unlock()
}

// Annotate decorates the supplied error with any debugging information
// accumulated on the logger.
func (l *Logger) Annotate(err error) error {
	return fmt.Errorf("%s\n\n%s", err.Error(), l.buf.String())
}
