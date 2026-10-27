// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"bytes"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

var (
	iconCacheMu sync.Mutex
	iconCache   = map[string]fyne.Resource{}
)

// iconResource returns a Breeze icon whose "currentColor" placeholders are
// replaced by clr. Icons are read from the embedded Breeze Dark icon set.
func iconResource(name string, clr color.Color) fyne.Resource {
	r, g, b, _ := clr.RGBA()
	key := fmt.Sprintf("%s-%02x%02x%02x", name, r>>8, g>>8, b>>8)

	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()
	if res, ok := iconCache[key]; ok {
		return res
	}

	raw, err := assets.ReadFile("assets/icons/" + name + ".svg")
	if err != nil {
		raw, err = assets.ReadFile("assets/icons/text-x-generic.svg")
		if err != nil {
			return nil
		}
	}
	hex := fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	raw = bytes.ReplaceAll(raw, []byte("currentColor"), []byte(hex))
	res := fyne.NewStaticResource(key+".svg", raw)
	iconCache[key] = res
	return res
}

func iconImage(name string, clr color.Color, size float32) *canvas.Image {
	img := canvas.NewImageFromResource(iconResource(name, clr))
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	img.SetMinSize(fyne.NewSize(size, size))
	img.Resize(fyne.NewSize(size, size))
	return img
}

var mimeIcons = map[string]string{
	".7z":   "application-x-archive",
	".avi":  "video-x-generic",
	".bmp":  "image-x-generic",
	".bz2":  "application-x-archive",
	".c":    "text-x-csrc",
	".cpp":  "text-x-csrc",
	".css":  "text-x-generic",
	".csv":  "x-office-spreadsheet",
	".doc":  "x-office-document",
	".docx": "x-office-document",
	".flac": "audio-x-generic",
	".gif":  "image-x-generic",
	".go":   "text-x-csrc",
	".gz":   "application-x-archive",
	".h":    "text-x-csrc",
	".htm":  "text-html",
	".html": "text-html",
	".jpeg": "image-x-generic",
	".jpg":  "image-x-generic",
	".js":   "text-x-script",
	".json": "application-json",
	".m4a":  "audio-x-generic",
	".md":   "text-x-generic",
	".mkv":  "video-x-generic",
	".mp3":  "audio-x-generic",
	".mp4":  "video-x-generic",
	".odp":  "x-office-presentation",
	".ods":  "x-office-spreadsheet",
	".odt":  "x-office-document",
	".ogg":  "audio-x-generic",
	".opus": "audio-x-generic",
	".pdf":  "application-pdf",
	".png":  "image-x-generic",
	".ppt":  "x-office-presentation",
	".pptx": "x-office-presentation",
	".py":   "text-x-script",
	".rar":  "application-x-archive",
	".sh":   "text-x-script",
	".so":   "application-x-sharedlib",
	".svg":  "image-x-generic",
	".tar":  "application-x-archive",
	".tif":  "image-x-generic",
	".tiff": "image-x-generic",
	".txt":  "text-x-generic",
	".wav":  "audio-x-generic",
	".webm": "video-x-generic",
	".webp": "image-x-generic",
	".xls":  "x-office-spreadsheet",
	".xlsx": "x-office-spreadsheet",
	".xml":  "application-xml",
	".xz":   "application-x-archive",
	".zip":  "application-x-archive",
	".zst":  "application-x-archive",
}

func fileIconName(name string, isDir bool) string {
	if isDir {
		return "folder"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if icon, ok := mimeIcons[ext]; ok {
		return icon
	}
	return "text-x-generic"
}

func folderIconName(base string) string {
	switch base {
	case "folder-red", "folder-blue", "folder-green", "folder-yellow", "folder-grey":
		return base
	}
	return "folder"
}

func placeColor(icon string) color.Color {
	switch icon {
	case "folder-red":
		return color.NRGBA{R: 0xda, G: 0x44, B: 0x53, A: 0xff}
	case "folder-blue":
		return color.NRGBA{R: 0x3d, G: 0xae, B: 0xe9, A: 0xff}
	case "folder-green":
		return color.NRGBA{R: 0x27, G: 0xae, B: 0x60, A: 0xff}
	case "folder-yellow":
		return color.NRGBA{R: 0xfd, G: 0xbc, B: 0x4b, A: 0xff}
	}
	return colText
}
