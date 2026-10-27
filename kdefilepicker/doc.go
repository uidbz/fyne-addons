// SPDX-License-Identifier: MIT

// Package kdefilepicker is a pixel-faithful reproduction of the KDE 6
// (Plasma) file dialog, implemented as a reusable Fyne object.
//
// The widget mirrors KFileWidget from KIO: a places sidebar with capacity
// bars, a breadcrumb navigator, the Breeze toolbar, a details/compact/icon
// file view, an editable name field, a file-type filter, and the Open/Cancel
// button row. Layout metrics, palettes, fonts and icons were taken from KIO
// 6.31 and Breeze Dark on Plasma 6.
//
// A dialog can be embedded directly:
//
//	d := kdefilepicker.NewOpenDialog(startDir)
//	d.SetOnAccept(func(path string) { fmt.Println(path) })
//	d.ShowInWindow(window)
//
// or used as a plain fyne.CanvasObject:
//
//	window.SetContent(kdefilepicker.NewOpenDialog(startDir))
//
// The package is desktop-oriented. Devices are discovered with lsblk on
// Linux; other operating systems simply omit the Devices group.
package kdefilepicker
