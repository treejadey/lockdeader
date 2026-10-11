package main

import (
	"context"
	"log/slog"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/network"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

// clipboardBridge puts links and images on the system clipboard for QML,
// which has no clipboard access of its own. QML asks for a copy by writing a
// URL to one of its keys:
//
//	copyLink:  copies the URL itself, as text
//	copyImage: copies the image at the URL
//
// Once a copy is done, "message" says how it went (e.g. "Image copied") and
// "messageId" goes up by one, so QML can tell a repeated message is new.
//
// All methods must be called on the Qt main thread.
type clipboardBridge struct {
	*qml.QQmlPropertyMap

	// The disk cache that QML loads images into (see installImageCache), so
	// images on screen can be copied without downloading them again.
	cache  *network.QNetworkDiskCache
	client *banana.Client

	messages int
}

func newClipboardBridge(imageCacheDir string, client *banana.Client) *clipboardBridge {
	b := &clipboardBridge{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		cache:           network.NewQNetworkDiskCache(),
		client:          client,
	}

	b.cache.SetCacheDirectory(imageCacheDir)

	b.Insert("copyLink", qt.NewQVariant14(""))
	b.Insert("copyImage", qt.NewQVariant14(""))
	b.Insert("message", qt.NewQVariant14(""))
	b.Insert("messageId", qt.NewQVariant4(0))

	// Only fires for changes made from QML, not for our own Inserts.
	b.OnValueChanged(func(key string, value *qt.QVariant) {
		url := value.ToString()
		if url == "" {
			return
		}

		// Clear the key, so asking for the same copy again is still a change.
		b.Insert(key, qt.NewQVariant14(""))

		switch key {
		case "copyLink":
			b.CopyLink(url)
		case "copyImage":
			b.CopyImage(url)
		}
	})

	return b
}

func (b *clipboardBridge) CopyLink(url string) {
	qt.QGuiApplication_Clipboard().SetText(url)
	b.notify("Link copied")
}

func (b *clipboardBridge) CopyImage(url string) {
	if data := b.cached(url); data != nil {
		b.setImage(url, data)

		return
	}

	go func() {
		data, err := b.client.Download(context.Background(), url)

		mainthread.Start(func() {
			if err != nil {
				slog.Error("failed to download image to copy", "url", url, "err", err)
				b.notify("Couldn't copy the image")

				return
			}

			b.setImage(url, data)
		})
	}()
}

func (b *clipboardBridge) cached(url string) []byte {
	device := b.cache.Data(qt.NewQUrl3(url))
	if device == nil {
		return nil
	}

	defer device.Delete()

	return device.ReadAll()
}

func (b *clipboardBridge) setImage(url string, data []byte) {
	img := qt.QImage_FromDataWithData(data)
	if img.IsNull() {
		slog.Error("failed to decode image to copy", "url", url)
		b.notify("Couldn't copy the image")

		return
	}

	qt.QGuiApplication_Clipboard().SetImage(img)
	b.notify("Image copied")
}

func (b *clipboardBridge) notify(message string) {
	b.messages++
	b.Insert("message", qt.NewQVariant14(message))
	b.Insert("messageId", qt.NewQVariant4(b.messages))
}
