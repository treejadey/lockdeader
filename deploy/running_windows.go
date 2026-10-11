package deploy

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Running reports whether Deadlock is running. It's best effort: when it
// can't tell, treat it as not running, so a check that fails never stops
// mods being applied.
func Running() (bool, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, err
	}

	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), exeName) {
			return true, nil
		}
	}

	if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return false, nil
	}

	return false, err
}
