// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// InstallShortcuts binds the standard KDE file-dialog keyboard shortcuts to
// the canvas of w. ShowInWindow and ShowOverlay call this automatically.
// Registrations are tracked so RemoveShortcuts can tear them down again;
// calling InstallShortcuts twice does not stack duplicate handlers.
func (d *Dialog) InstallShortcuts(w fyne.Window) {
	c := w.Canvas()
	if c == nil {
		return
	}
	d.RemoveShortcuts()
	add := func(key fyne.KeyName, mod fyne.KeyModifier, fn func()) {
		sc := &desktop.CustomShortcut{KeyName: key, Modifier: mod}
		c.AddShortcut(sc, func(fyne.Shortcut) { fn() })
		d.installed = append(d.installed, installedShortcut{canvas: c, shortcut: sc})
	}
	add(fyne.KeyL, fyne.KeyModifierControl, d.beginPathEdit)
	add(fyne.KeyI, fyne.KeyModifierControl, func() { d.setQuickFilter(!d.quickVisible) })
	add(fyne.KeyF, fyne.KeyModifierControl, func() { d.setQuickFilter(true) })
	add(fyne.KeyH, fyne.KeyModifierControl, func() { d.SetShowHidden(!d.showHidden) })
	add(fyne.KeyN, fyne.KeyModifierControl|fyne.KeyModifierShift, d.newFolder)
	add(fyne.KeyF5, 0, func() { d.RefreshDir() })
	add(fyne.KeyF9, 0, func() {
		d.showPlaces = !d.showPlaces
		d.Refresh()
	})
	add(fyne.KeyEscape, 0, func() {
		switch {
		case d.editing:
			d.endPathEdit()
		case d.quickVisible:
			d.setQuickFilter(false)
		default:
			d.cancel()
		}
	})
	add(fyne.KeyLeft, fyne.KeyModifierAlt, d.back)
	add(fyne.KeyRight, fyne.KeyModifierAlt, d.forward)
	add(fyne.KeyUp, fyne.KeyModifierAlt, d.up)
}

// installedShortcut remembers one canvas shortcut registration for teardown.
type installedShortcut struct {
	canvas   fyne.Canvas
	shortcut fyne.Shortcut
}

// RemoveShortcuts uninstalls the keyboard shortcuts registered by
// InstallShortcuts so they do not leak into the host window after the
// dialog is closed.
func (d *Dialog) RemoveShortcuts() {
	for _, in := range d.installed {
		in.canvas.RemoveShortcut(in.shortcut)
	}
	d.installed = nil
}
