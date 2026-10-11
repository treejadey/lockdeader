package main

import (
	_ "embed"
	"fmt"

	qt "github.com/mappu/miqt/qt6"
)

//go:embed assets/logo.svg
var logoSVG []byte

// QIcon only keeps an SVG scalable when it reads it from a file, so the
// embedded logo is drawn at each size the desktop may ask for instead.
var logoSizes = []int{16, 24, 32, 48, 64, 128, 256}

// setWindowIcon must be called after the application is created and before
// any window opens.
func setWindowIcon() error {
	icon := qt.NewQIcon()

	for _, size := range logoSizes {
		image, err := drawLogo(size)
		if err != nil {
			return err
		}

		icon.AddPixmap(qt.QPixmap_FromImage(image))
	}

	qt.QGuiApplication_SetWindowIcon(icon)

	return nil
}

func drawLogo(size int) (*qt.QImage, error) {
	buffer := qt.NewQBuffer()
	defer buffer.Delete()

	buffer.SetData(logoSVG)
	buffer.Open(qt.QIODeviceBase__ReadOnly)

	reader := qt.NewQImageReader4(buffer.QIODevice, []byte("svg"))
	defer reader.Delete()

	reader.SetScaledSize(qt.NewQSize2(size, size))

	image := reader.Read()
	if image.IsNull() {
		return nil, fmt.Errorf("drawing the logo at %d px: %s", size, reader.ErrorString())
	}

	return image, nil
}
