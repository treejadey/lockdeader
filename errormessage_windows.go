package main

import (
	"errors"
	"syscall"
)

// Windows reports a full disk with its own error codes, which syscall.ENOSPC
// doesn't match.
const (
	errorHandleDiskFull syscall.Errno = 39
	errorDiskFull       syscall.Errno = 112
)

func isDiskFull(err error) bool {
	var errno syscall.Errno

	return errors.As(err, &errno) && (errno == errorHandleDiskFull || errno == errorDiskFull || errno == syscall.ENOSPC)
}
