package utils

import (
	"io"
	"log"
	"os"
	"time"
)

const (
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorWhite  = "\033[39m"
	colorReset  = "\033[0m"
	timeFormat  = "15:04:05"
)

// *L* is for regular logs
// *F* is for failure logs
// *S* is for success logs
// *D* is for debug logs
var L, F, S, D *log.Logger

type logWriter struct {
	output io.Writer
}
type failWriter struct {
	output io.Writer
}
type successWriter struct {
	output io.Writer
}
type debugWriter struct {
	output io.Writer
}

func init() {
	L = log.New(&logWriter{output: os.Stdout}, "", 0)
	F = log.New(&failWriter{output: os.Stdout}, "", 0)
	S = log.New(&successWriter{output: os.Stdout}, "", 0)
	D = log.New(&debugWriter{output: os.Stdout}, "", 0)
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	now := time.Now().Format(timeFormat)
	prefix := colorCyan + "[HBCD @ " + now + "] " + colorReset
	content := colorGray + string(p) + colorReset

	_, err = w.output.Write([]byte(prefix + content))

	return len(p), nil
}

func (w *failWriter) Write(p []byte) (n int, err error) {
	now := time.Now().Format(timeFormat)
	prefix := colorRed + "[HBCD @ " + now + "] " + colorReset
	content := colorGray + string(p) + colorReset

	_, err = w.output.Write([]byte(prefix + content))

	return len(p), nil
}

func (w *successWriter) Write(p []byte) (n int, err error) {
	now := time.Now().Format(timeFormat)
	prefix := colorGreen + "[HBCD @ " + now + "] " + colorReset
	content := colorWhite + string(p) + colorReset

	_, err = w.output.Write([]byte(prefix + content))

	return len(p), nil
}

func (w *debugWriter) Write(p []byte) (n int, err error) {
	now := time.Now().Format(timeFormat)
	prefix := colorYellow + "[DEBUG: HBCD @ " + now + "] " + colorReset
	content := colorWhite + string(p) + colorReset

	_, err = w.output.Write([]byte(prefix + content))

	return len(p), nil
}
