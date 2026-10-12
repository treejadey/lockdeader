package main

import (
	"bytes"
	"io/fs"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

// Qt reads every file in qml/ back as it was (TestMain registers them).
func TestQMLFiles(t *testing.T) {
	count := 0

	err := fs.WalkDir(qmlFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		want, err := fs.ReadFile(qmlFiles, path)
		if err != nil {
			return err
		}

		count++

		mainthread.Wait(func() {
			file := qt.NewQFile2(":/" + path)
			defer file.Delete()

			if !file.Open(qt.QIODeviceBase__ReadOnly) {
				t.Errorf("Qt can't open :/%s", path)

				return
			}

			if got := file.ReadAll(); !bytes.Equal(got, want) {
				t.Errorf(":/%s has %d bytes, want %d", path, len(got), len(want))
			}
		})

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if count < 50 {
		t.Errorf("only %d files embedded", count)
	}
}
