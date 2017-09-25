package log

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

// New returns a logger that writes to standard error.
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

// Debugf buffers a debugging message. Before any non-zero exit, call
// FlushDebug to write out the accumulated debug logs.
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

// FlushDebug writes out any accumulated debug logs.
func (l *Logger) FlushDebug() {
	l.out.Write(l.buf.Bytes())
}
