// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// InstallShortcuts binds the standard KDE file-dialog keyboard shortcuts to
// the canvas of w. ShowInWindow calls this automatically.
func (d *Dialog) InstallShortcuts(w fyne.Window) {
	c := w.Canvas()
	if c == nil {
		return
	}
	add := func(key fyne.KeyName, mod fyne.KeyModifier, fn func()) {
		c.AddShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: mod}, func(fyne.Shortcut) { fn() })
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
