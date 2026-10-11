//go:build !linux

package filedialog

// ChooseFolder asks the user for a folder, starting in start if it's not "".
// It returns "" if they cancel. It blocks until they're done, so call it from
// a goroutine.
func ChooseFolder(parent Window, title, start string) (string, error) {
	return chooseWithZenity(parent, title, start)
}
