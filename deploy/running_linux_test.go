package deploy

import (
	"os"
	"os/exec"
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

func TestPgrep(t *testing.T) {
	// pgrep's exit codes: 0 found, 1 not found, more for its own errors.
	for code, want := range map[string]bool{"0": true, "1": false} {
		if got, err := pgrep([]string{"sh", "-c", "exit " + code, "sh"}); got != want || err != nil {
			t.Errorf("exit %s: %v, %v; want %v", code, got, err, want)
		}
	}

	if _, err := pgrep([]string{"sh", "-c", "exit 2", "sh"}); err == nil {
		t.Error("pgrep's own error isn't one")
	}

	if _, err := pgrep([]string{"sh", "-c", "sleep 5", "sh"}); err == nil {
		t.Error("a pgrep that doesn't finish in time isn't an error")
	}

	if _, err := pgrep([]string{"no-such-command-lockdeader"}); err == nil {
		t.Error("a missing command isn't an error")
	}

	// The real pgrep finds a process named like the game.
	if got, err := pgrepRunning("pgrep"); got || err != nil {
		t.Fatalf("pgrep with no game running: %v, %v", got, err)
	}

	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("no pgrep")
	}

	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep")
	}

	data, err := os.ReadFile(sleep)
	if err != nil {
		t.Fatal(err)
	}

	game := filepath.Join(t.TempDir(), exeName)

	if err := os.WriteFile(game, data, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(game, "10")

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Waiting reaps it, or it would linger as a zombie with the game's name.
	defer cmd.Wait()
	defer cmd.Process.Kill()

	if got, err := pgrepRunning("pgrep"); !got || err != nil {
		t.Errorf("pgrep with the game running: %v, %v", got, err)
	}
}

// Under Wine, the game's process soon has another name, but its command line
// still starts with the exe's Windows path.
func TestPgrepCommandLine(t *testing.T) {
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("no pgrep")
	}

	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep")
	}

	start := func(argv0 string) {
		t.Helper()

		cmd := exec.Command(sleep, "10")
		cmd.Args[0] = argv0

		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			cmd.Process.Kill()
			cmd.Wait()
		})
	}

	// Neither is the game, though both mention it.
	start(`Z:\Games\Deadlock\game\bin\win64\deadlock.exe.bak`)
	start(`/usr/bin/reaper Z:\Games\Deadlock\game\bin\win64\deadlock.exe`)

	if got, err := pgrepRunning("pgrep"); got || err != nil {
		t.Fatalf("pgrep with no game running: %v, %v", got, err)
	}

	start(`Z:\Steam Library\steamapps\common\Deadlock\game\bin\win64\DEADLOCK.EXE`)

	if got, err := pgrepRunning("pgrep"); !got || err != nil {
		t.Errorf("pgrep with the game running: %v, %v", got, err)
	}
}
