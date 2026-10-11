package main

import (
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/url"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/audio"
	"github.com/treejadey/lockdeader/banana"
	"github.com/treejadey/lockdeader/deploy"
	"github.com/treejadey/lockdeader/install"
)

// errorTexts gives errorMessage(err) and err.Error(), or two "" for nil.
func errorTexts(err error) (message, detail string) {
	if err == nil {
		return "", ""
	}

	return errorMessage(err), err.Error()
}

// insertError puts err in a bridge as "error", in plain words, and
// "errorDetail", as Go wrote it, for the UI to show on hover.
func insertError(bridge *qml.QQmlPropertyMap, err error) {
	message, detail := errorTexts(err)

	bridge.Insert("error", qt.NewQVariant14(message))
	bridge.Insert("errorDetail", qt.NewQVariant14(detail))
}

// errorMessage says in plain words why something failed, and what to do about
// it. The UI puts it after what failed ("Couldn’t load mods"), and shows
// err.Error() itself only as a detail, on hover.
func errorMessage(err error) string {
	var apiErr *banana.APIError

	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == http.StatusNotFound:
			return tr("It’s no longer on GameBanana.")
		case apiErr.StatusCode == http.StatusTooManyRequests:
			return tr("GameBanana is busy. Try again in a minute.")
		case apiErr.StatusCode >= 500:
			return tr("GameBanana isn’t working right now. Try again later.")
		}

		return tr("Something went wrong.")
	}

	// A request that never got an answer, whether from DNS, the connection
	// or a timeout, comes back as a *url.Error. Not net.Error: syscall.Errno
	// is one too, so a full disk would match.
	var urlErr *url.Error
	var opErr *net.OpError
	var dnsErr *net.DNSError

	if errors.As(err, &urlErr) || errors.As(err, &opErr) || errors.As(err, &dnsErr) || errors.Is(err, install.ErrStalled) {
		return tr("GameBanana can’t be reached. Check your internet connection and try again.")
	}

	switch {
	case errors.Is(err, install.ErrChecksum):
		return tr("The download was damaged. Try installing it again.")
	case errors.Is(err, install.ErrUnknownFormat):
		return tr("The mod’s file isn’t an archive Lockdeader can open.")
	case errors.Is(err, install.ErrNoVPKs), errors.Is(err, install.ErrNotVPK):
		return tr("The mod has no files the game can use.")
	case errors.Is(err, install.ErrChunkedVPK):
		return tr("The mod is split into parts, which isn’t supported yet.")
	case errors.Is(err, install.ErrTooLarge):
		return tr("The mod is too big to pack.")
	case errors.Is(err, deploy.ErrGameRunning):
		return tr("Deadlock is running. Close it and try again.")
	case errors.Is(err, deploy.ErrTooMany):
		return tr("Too many of the mods that are on change the same files. Turn some off.")
	case errors.Is(err, deploy.ErrMissing):
		return tr("Some mods’ files are missing. Reinstall them.")
	case errors.Is(err, deploy.ErrNoSearchPaths):
		return tr("Deadlock’s gameinfo.gi isn’t one Lockdeader understands. Verify the game files in Steam.")
	case isDiskFull(err):
		return tr("There isn’t enough disk space.")
	case errors.Is(err, errGameFolder) && errors.Is(err, fs.ErrPermission):
		return tr("Lockdeader isn’t allowed to change files in Deadlock’s folder.")
	case errors.Is(err, fs.ErrPermission):
		return tr("Lockdeader isn’t allowed to save files in its data folder.")
	case errors.Is(err, audio.ErrDevice):
		return tr("No sound output is available.")
	case errors.Is(err, audio.ErrDecode):
		return tr("This sound can’t be played.")
	}

	return tr("Something went wrong.")
}
