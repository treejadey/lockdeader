// Package install downloads mods from GameBanana and keeps the VPKs taken out
// of them in a store in the data directory (Cache/Mods), named by their
// SHA-256. See docs/mod-support-plan.md.
package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// VPKs are named by their SHA-256, e.g. "3a7bd3e2….vpk", so two mods with the
// same VPK share one copy.
//
// Files only get their final name once they're complete, so a crash leaves
// at worst a temporary file behind (see CleanUp), never a broken VPK.
type Store struct {
	Dir string
}

// Temporary files: VPKs being added, and archives being downloaded.
const (
	tempSuffix    = ".tmp"
	partialSuffix = ".partial"
)

// The first four bytes of every VPK, read as little-endian.
const vpkMagic = 0x55AA1234

var ErrNotVPK = errors.New("install: not a VPK file")

func (s Store) Path(sha256 string) string {
	return filepath.Join(s.Dir, sha256+".vpk")
}

// Each download gets a file of its own, even of the same archive.
func (s Store) createPartial(fileID int) (*os.File, error) {
	return os.CreateTemp(s.Dir, fmt.Sprintf("%d-*%s", fileID, partialSuffix))
}

// Add fails with ErrNotVPK if r doesn't start like a VPK. Adding a VPK the
// store already has changes nothing.
func (s Store) Add(r io.Reader) (sha string, size int64, err error) {
	var header [8]byte

	if _, err := io.ReadFull(r, header[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return "", 0, ErrNotVPK
		}

		return "", 0, fmt.Errorf("install: %w", err)
	}

	if binary.LittleEndian.Uint32(header[:4]) != vpkMagic {
		return "", 0, ErrNotVPK
	}

	tmp, err := os.CreateTemp(s.Dir, "*"+tempSuffix)
	if err != nil {
		return "", 0, fmt.Errorf("install: %w", err)
	}

	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	hash := sha256.New()

	size, err = io.Copy(io.MultiWriter(tmp, hash), io.MultiReader(bytes.NewReader(header[:]), r))
	if err != nil {
		return "", 0, fmt.Errorf("install: %w", err)
	}

	// Flushed before it's renamed, so a power cut can't leave an empty file
	// under the final name.
	if err = tmp.Sync(); err != nil {
		return "", 0, fmt.Errorf("install: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return "", 0, fmt.Errorf("install: %w", err)
	}

	sha = hex.EncodeToString(hash.Sum(nil))
	final := s.Path(sha)

	if _, statErr := os.Stat(final); statErr == nil {
		// Already there: the same contents, as the name says. Replacing it
		// could fail anyway on Windows, if the game has it open.
		os.Remove(tmp.Name())

		return sha, size, nil
	}

	if err = os.Rename(tmp.Name(), final); err != nil {
		return "", 0, fmt.Errorf("install: %w", err)
	}

	return sha, size, nil
}

// Removing a VPK that isn't there isn't an error.
func (s Store) Remove(sha256 string) error {
	if err := os.Remove(s.Path(sha256)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

func (s Store) VPKs() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	var shas []string

	for _, e := range entries {
		if sha, ok := strings.CutSuffix(e.Name(), ".vpk"); ok && e.Type().IsRegular() {
			shas = append(shas, sha)
		}
	}

	return shas, nil
}

// CleanUp deletes what unfinished installs left behind. Nothing may be
// installing while it runs.
func (s Store) CleanUp() error {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}

	var errs []error

	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, tempSuffix) || strings.HasSuffix(name, partialSuffix) {
			if err := os.Remove(filepath.Join(s.Dir, name)); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	return nil
}
