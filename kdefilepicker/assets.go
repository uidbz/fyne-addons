// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"embed"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed assets/icons/*.svg assets/fonts/*.ttf
var assets embed.FS

var (
	fontRegular    fyne.Resource
	fontBold       fyne.Resource
	fontItalic     fyne.Resource
	fontBoldItalic fyne.Resource
	fontOnce       sync.Once
)

func loadFonts() {
	fontOnce.Do(func() {
		fontRegular = readAsset("assets/fonts/NotoSans-Regular.ttf")
		fontBold = readAsset("assets/fonts/NotoSans-Bold.ttf")
		fontItalic = readAsset("assets/fonts/NotoSans-Italic.ttf")
		fontBoldItalic = readAsset("assets/fonts/NotoSans-BoldItalic.ttf")
	})
}

func readAsset(name string) fyne.Resource {
	b, err := assets.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("kdefilepicker: embedded asset %q missing: %v", name, err))
	}
	return fyne.NewStaticResource(name, b)
}
