package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunning(t *testing.T) {
	proc := t.TempDir()

	write := func(pid, name, data string) {
		t.Helper()
		os.MkdirAll(filepath.Join(proc, pid), 0o755)

		if err := os.WriteFile(filepath.Join(proc, pid, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("789", "comm", "bash\n")
	write("789", "cmdline", "/bin/bash\x00")
	write("790", "comm", "steam\n")
	os.WriteFile(filepath.Join(proc, "version"), []byte("not a process"), 0o644)
	os.MkdirAll(filepath.Join(proc, "self"), 0o755)

	if got, err := running(proc); got || err != nil {
		t.Errorf("running = %v, %v with no game", got, err)
	}

	// Under Proton, the name is the exe's, and the command line a Windows path.
	write("456", "cmdline", "Z:\\SteamLibrary\\steamapps\\common\\Deadlock\\game\\bin\\win64\\deadlock.exe\x00-novid\x00")

	if got, err := running(proc); !got || err != nil {
		t.Errorf("running = %v, %v with the game's command line", got, err)
	}

	os.RemoveAll(filepath.Join(proc, "456"))
	write("123", "comm", "deadlock.exe\n")

	if got, err := running(proc); !got || err != nil {
		t.Errorf("running = %v, %v with the game's comm", got, err)
	}

	if _, err := running(filepath.Join(proc, "missing")); err == nil {
		t.Error("no error for a proc that can't be read")
	}
}
