package deploy

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// link puts a file into our folder without copying it; os.Link, other than
// in tests. A hard link needs the store and the game on the same volume,
// so when it fails, the file is copied instead.
var link = os.Link

// On Windows, antivirus or Steam can hold a file open for a moment, which
// makes renaming or deleting it fail. Each is tried again a few times, with
// a short wait in between, as Deadlock Mod Manager does. Linux never needs
// the retries, but running the same code everywhere is one less thing to
// get wrong.
var retryDelays = []time.Duration{50 * time.Millisecond, 100 * time.Millisecond,
	200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond}

func retry(op func() error) error {
	var err error

	for i := 0; ; i++ {
		if err = op(); err == nil || i == len(retryDelays) {
			return err
		}

		time.Sleep(retryDelays[i])
	}
}

// RemoveFile deletes a file, trying again if something has it open for a
// moment (see retry), and is happy if it's already gone.
func RemoveFile(path string) error {
	return removeFile(path)
}

// removeFile deletes a file, and is happy if it's already gone.
func removeFile(path string) error {
	err := retry(func() error {
		err := os.Remove(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	})
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return nil
}

// removeEntry deletes a file, or a folder and everything in it.
func removeEntry(path string) error {
	err := retry(func() error {
		return os.RemoveAll(path)
	})
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return nil
}

// placeFile puts the file at src at dst, by a hard link if it can, or a copy.
func placeFile(src, dst string, size int64) error {
	if err := link(src, dst); err == nil {
		return nil
	}

	return copyFile(src, dst, size)
}

func copyFile(src, dst string, size int64) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("deploy: %w", closeErr)
		}
	}()

	// io.Copy between files uses the system's own copying, which on btrfs
	// and XFS shares the data rather than duplicating it.
	n, err := io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	if n != size {
		return fmt.Errorf("deploy: %s is %d bytes, not %d", src, n, size)
	}

	// Flushed before it's renamed into place, so a power cut can't leave a
	// slot that's empty under its final name.
	if err := out.Sync(); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return nil
}

// writeFileAtomic writes data to a temporary file next to path, flushes it,
// then renames it over path: a crash leaves the old file or the new one,
// never half of each. A read-only path, as some guides tell Windows users
// to make gameinfo.gi, is made writable first.
func writeFileAtomic(path string, data []byte, perm fs.FileMode) (err error) {
	tmp := path + ".lockdeader" + tempSuffix

	if err := removeFile(tmp); err != nil {
		return err
	}

	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm|0o200)
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()

	if _, err := out.Write(data); err != nil {
		out.Close()

		return fmt.Errorf("deploy: %w", err)
	}

	if err := out.Sync(); err != nil {
		out.Close()

		return fmt.Errorf("deploy: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	// On Windows, Chmod with the write bit clears the read-only attribute;
	// without it, the rename over the file fails.
	if err := os.Chmod(path, perm|0o200); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("deploy: %w", err)
	}

	if err := retry(func() error { return os.Rename(tmp, path) }); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return nil
}
