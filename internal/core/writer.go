package core

import (
	"fmt"
	"io"
	"os"
)

// StdOutputWriter writes output to stdout and diagnostics to stderr.
type StdOutputWriter struct {
	stdout io.Writer
	stderr io.Writer
}

// NewStdOutputWriter creates a writer using os.Stdout and os.Stderr.
func NewStdOutputWriter() *StdOutputWriter {
	return &StdOutputWriter{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// WriteOutput writes data to stdout.
func (w *StdOutputWriter) WriteOutput(data []byte) (int, error) {
	return w.stdout.Write(data)
}

// WriteDiagnostic writes diagnostic info to stderr.
func (w *StdOutputWriter) WriteDiagnostic(data []byte) (int, error) {
	return w.stderr.Write(data)
}

// WriteFormatted writes formatted output to stdout.
func (w *StdOutputWriter) WriteFormatted(format string, args ...any) error {
	_, err := fmt.Fprintf(w.stdout, format, args...)
	return err
}

// OutputWriter returns the underlying stdout writer.
func (w *StdOutputWriter) OutputWriter() io.Writer {
	return w.stdout
}

// DiagnosticWriter returns the underlying stderr writer.
func (w *StdOutputWriter) DiagnosticWriter() io.Writer {
	return w.stderr
}

// BufferedOutputWriter collects output in memory.
type BufferedOutputWriter struct {
	output     []byte
	diagnostic []byte
}

// NewBufferedOutputWriter creates a writer that buffers output in memory.
func NewBufferedOutputWriter() *BufferedOutputWriter {
	return &BufferedOutputWriter{}
}

// WriteOutput appends to the output buffer.
func (w *BufferedOutputWriter) WriteOutput(data []byte) (int, error) {
	w.output = append(w.output, data...)
	return len(data), nil
}

// WriteDiagnostic appends to the diagnostic buffer.
func (w *BufferedOutputWriter) WriteDiagnostic(data []byte) (int, error) {
	w.diagnostic = append(w.diagnostic, data...)
	return len(data), nil
}

// WriteFormatted appends formatted text to the output buffer.
func (w *BufferedOutputWriter) WriteFormatted(format string, args ...any) error {
	w.output = append(w.output, []byte(fmt.Sprintf(format, args...))...)
	return nil
}

// OutputWriter returns a reader for the buffered output.
func (w *BufferedOutputWriter) OutputWriter() io.Writer {
	return &writeCollector{buf: &w.output}
}

// DiagnosticWriter returns a reader for the buffered diagnostics.
func (w *BufferedOutputWriter) DiagnosticWriter() io.Writer {
	return &writeCollector{buf: &w.diagnostic}
}

// Output returns the collected output bytes.
func (w *BufferedOutputWriter) Output() []byte {
	return w.output
}

// Diagnostic returns the collected diagnostic bytes.
func (w *BufferedOutputWriter) Diagnostic() []byte {
	return w.diagnostic
}

type writeCollector struct {
	buf *[]byte
}

func (w *writeCollector) Write(data []byte) (int, error) {
	*w.buf = append(*w.buf, data...)
	return len(data), nil
}
