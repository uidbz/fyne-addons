// SPDX-License-Identifier: MIT

// Command kdefilepicker-demo opens the KDE 6 file dialog replica.
package main

import (
	"flag"
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"

	"github.com/uidbz/fyne-addons/kdefilepicker"
)

func main() {
	var (
		dir        = flag.String("dir", "", "directory to open (default: home)")
		save       = flag.Bool("save", false, "open a save dialog")
		folder     = flag.Bool("folder", false, "open a folder-selection dialog")
		preview    = flag.Bool("preview", false, "show the preview panel")
		view       = flag.String("view", "details", "details, compact or icons")
		selectFile = flag.String("select", "", "select this file name on startup")
	)
	flag.Parse()

	start := *dir
	if start == "" {
		start, _ = os.UserHomeDir()
	}

	a := app.NewWithID("org.uidbz.fyneaddons.kdefilepicker")
	a.Settings().SetTheme(kdefilepicker.NewTheme())

	var d *kdefilepicker.Dialog
	switch {
	case *save:
		d = kdefilepicker.NewSaveDialog(start, "Untitled.txt")
	case *folder:
		d = kdefilepicker.NewDirectoryDialog(start)
	default:
		d = kdefilepicker.NewOpenDialog(start)
	}
	d.SetFilters([]kdefilepicker.Filter{
		kdefilepicker.AllFiles,
		{Name: "Images", Patterns: []string{"*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.svg"}},
		{Name: "Text Files", Patterns: []string{"*.txt", "*.md", "*.go", "*.c", "*.h"}},
		{Name: "Audio", Patterns: []string{"*.mp3", "*.flac", "*.ogg", "*.wav", "*.opus", "*.m4a"}},
		{Name: "Video", Patterns: []string{"*.mp4", "*.mkv", "*.webm", "*.avi", "*.mov"}},
	})

	w := a.NewWindow("Open File")
	d.SetOnAccept(func(path string) {
		fmt.Println(path)
		w.Close()
	})
	d.SetOnCancel(func() {
		fmt.Println("cancelled")
		w.Close()
	})
	if *view == "icons" {
		d.SetViewMode(kdefilepicker.ViewIcons)
	} else if *view == "compact" {
		d.SetViewMode(kdefilepicker.ViewCompact)
	}
	d.ShowInWindow(w)
	if *preview {
		d.ShowPreview(true)
	}
	if *selectFile != "" {
		d.SelectFile(*selectFile)
	}
	w.ShowAndRun()
}
