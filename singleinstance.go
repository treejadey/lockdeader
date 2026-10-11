package main

import (
	"crypto/sha256"
	"fmt"
	"log/slog"

	"github.com/mappu/miqt/qt6/network"
)

// claimInstance makes this the only copy of the app running with dataDir, so
// two never share a database. If another copy already has it, that copy is
// asked to show its window, and claimInstance returns false. Otherwise show
// is called whenever another copy starts.
//
// QLocalServer is a Unix domain socket on Linux and a named pipe on Windows.
func claimInstance(dataDir string, show func()) bool {
	// Per data directory, so a development copy with its own
	// LOCKDEADER_DATA_DIR can run alongside the installed one.
	sum := sha256.Sum256([]byte(dataDir))
	name := fmt.Sprintf("lockdeader-%x", sum[:8])

	// Connecting is all it takes to ask the other copy to show its window.
	socket := network.NewQLocalSocket()
	defer socket.Delete()

	socket.ConnectToServerWithName(name)
	if socket.WaitForConnectedWithMsecs(1000) {
		socket.DisconnectFromServer()

		return false
	}

	server := network.NewQLocalServer()
	server.SetSocketOptions(network.QLocalServer__UserAccessOption)

	// A copy that crashed leaves its socket behind on Unix, which would stop
	// this one listening.
	network.QLocalServer_RemoveServer(name)

	if !server.Listen(name) {
		slog.Warn("failed to listen for other copies of the app starting", "err", server.ErrorString())

		return true
	}

	server.OnNewConnection(func() {
		for conn := server.NextPendingConnection(); conn != nil; conn = server.NextPendingConnection() {
			conn.Abort()
			conn.DeleteLater()
		}

		show()
	})

	return true
}
