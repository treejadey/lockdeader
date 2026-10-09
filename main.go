package main

import (
	"log/slog"
	"os"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

func main() {
	qt.NewQGuiApplication(os.Args)

	// app holds UI state that QML binds to.
	app := qml.NewQQmlPropertyMap()
	client := banana.NewClient()
	mods := newModListModel(client, app)
	details := newModDetails(client, mods.Lookup)

	retryAction := qt.NewQAction()
	retryAction.OnTriggered(mods.Retry)

	engine := qml.NewQQmlApplicationEngine()
	engine.RootContext().SetContextProperty("app", app.QObject)
	engine.RootContext().SetContextProperty("mods", mods.QObject)
	engine.RootContext().SetContextProperty("details", details.QObject)
	engine.RootContext().SetContextProperty("retryAction", retryAction.QObject)
	engine.Load(qt.QUrl_FromLocalFile("qml/Main.qml"))
	if len(engine.RootObjects()) == 0 {
		slog.Error("failed to load qml/Main.qml")
		os.Exit(1)
	}

	// Load the first page; the grid requests the rest as it scrolls.
	mods.FetchNextPage()

	os.Exit(qt.QGuiApplication_Exec())
}
