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
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
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

// CaptureStderr hijacks os.Stderr and returns a buffer and a cleanup function.
// It handles synchronization to prevent race conditions during test assertions.
func CaptureStderr(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	output := new(bytes.Buffer)

	done := make(chan struct{})

	go func() {
		io.Copy(output, r)
		close(done)
	}()

	cleanup := func() {
		w.Close()
		<-done
		os.Stderr = old
	}

	return output, cleanup
}
