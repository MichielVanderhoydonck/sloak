package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureOutput hijacks os.Stdout and returns a buffer and a cleanup function.
// It handles synchronization to prevent race conditions during test assertions.
func CaptureOutput(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper() // Marks this function as a test helper for better failure logs

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output := new(bytes.Buffer)

	// Channel to signal when copying is complete
	done := make(chan struct{})

	// Start the copy in a goroutine
	go func() {
		io.Copy(output, r)
		close(done)
	}()

	// The cleanup function
	cleanup := func() {
		w.Close()
		<-done // Wait for the copy to finish
		os.Stdout = old
	}

	return output, cleanup
}
