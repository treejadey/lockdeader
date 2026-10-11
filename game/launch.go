package game

import (
	"errors"
	"net/url"
	"os/exec"
	"strings"
)

// ErrNoSteam is returned by Launch when Steam's executable can't be found.
// SteamURL can still start the game, through whatever opens steam:// links.
var ErrNoSteam = errors.New("game: Steam isn't installed")

// Launch starts Deadlock through Steam, with args added to its command line.
// Steam is started too if it isn't running. Launch doesn't wait for either.
func Launch(args []string) error {
	steam := steamCommand()
	if steam == nil {
		return ErrNoSteam
	}

	cmd := exec.Command(steam[0], append(append(steam[1:], "-applaunch", AppID), args...)...)
	detach(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}

	// Steam may keep running for as long as the user is logged in.
	go cmd.Wait()

	return nil
}

// SteamURL starts Deadlock with args, when opened.
func SteamURL(args []string) string {
	return "steam://run/" + AppID + "//" + url.PathEscape(strings.Join(args, " ")) + "/"
}

// SplitOptions splits launch options into arguments as a shell would, at
// spaces outside quotes: `-novid +exec "my config.cfg"` is three.
func SplitOptions(options string) []string {
	var args []string
	var arg strings.Builder

	inArg := false
	var quote rune

	for _, r := range options {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			arg.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
			inArg = true
		case r == ' ' || r == '\t' || r == '\n':
			if inArg {
				args = append(args, arg.String())
				arg.Reset()
				inArg = false
			}
		default:
			arg.WriteRune(r)
			inArg = true
		}
	}

	if inArg {
		args = append(args, arg.String())
	}

	return args
}
