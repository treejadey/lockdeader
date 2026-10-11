//go:build !windows

package main

import (
	"fmt"
	"io/fs"
	"syscall"
	"testing"
)

func TestErrorMessageDiskFull(t *testing.T) {
	err := fmt.Errorf("install: %w", &fs.PathError{Op: "write", Path: "/x", Err: syscall.ENOSPC})

	if got, want := errorMessage(err), "There isn’t enough disk space."; got != want {
		t.Errorf("errorMessage(%v) = %q, want %q", err, got, want)
	}
}
