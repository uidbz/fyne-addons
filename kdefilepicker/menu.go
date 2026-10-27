// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type menuEntry struct {
	label     string
	icon      fyne.Resource
	checked   bool
	disabled  bool
	separator bool
	group     bool
	action    func()
}

func menuSeparator() menuEntry { return menuEntry{separator: true} }
func menuGroup(label string) menuEntry {
	return menuEntry{label: label, group: true, disabled: true}
}

func checkedEntry(label string, checked bool, action func()) menuEntry {
	return menuEntry{label: label, checked: checked, action: action}
}

// breezeMenu is a small Breeze-styled context menu. Fyne's stock menu is
// intentionally avoided so the filter/sort/options dropdowns match the dialog.
type breezeMenu struct {
	widget.BaseWidget

	entries []menuEntry
	hover   int
	close   func()

	width float32
	rows  []*menuRow
	bg    *canvas.Rectangle
}

type menuRow struct {
	label *canvas.Text
	icon  *canvas.Image
	check *canvas.Image
	hover *canvas.Rectangle
	line  *canvas.Rectangle
}

const (
	menuItemH = 36
	menuSepH  = 9
	menuPad   = 6
)

func newBreezeMenu(entries []menuEntry) *breezeMenu {
	m := &breezeMenu{entries: entries, hover: -1}
	m.width = 230
	for _, e := range entries {
		if e.group {
			continue
		}
		w := fyne.MeasureText(e.label, textSize, fyne.TextStyle{}).Width + 70
		if w > m.width {
			m.width = w
		}
	}
	if m.width > 360 {
		m.width = 360
	}
	m.ExtendBaseWidget(m)
	return m
}

func (m *breezeMenu) CreateRenderer() fyne.WidgetRenderer {
	m.bg = newRect(colHeader, colBorder, 1, 5)
	return &breezeMenuRenderer{m: m}
}

func (m *breezeMenu) MinSize() fyne.Size {
	h := float32(menuPad * 2)
	for _, e := range m.entries {
		if e.separator {
			h += menuSepH
		} else {
			h += menuItemH
		}
	}
	return fyne.NewSize(m.width, h)
}

func (m *breezeMenu) rowAt(y float32) int {
	pos := float32(menuPad)
	for i, e := range m.entries {
		h := float32(menuItemH)
		if e.separator {
			h = menuSepH
		}
		if y >= pos && y < pos+h {
			return i
		}
		pos += h
	}
	return -1
}

func (m *breezeMenu) Tapped(e *fyne.PointEvent) {
	idx := m.rowAt(e.Position.Y)
	if idx < 0 {
		return
	}
	entry := m.entries[idx]
	if entry.disabled || entry.separator || entry.group {
		return
	}
	if m.close != nil {
		m.close()
	}
	if entry.action != nil {
		entry.action()
	}
}

func (m *breezeMenu) MouseMoved(e *desktop.MouseEvent) {
	idx := m.rowAt(e.Position.Y)
	if idx >= 0 && (m.entries[idx].disabled || m.entries[idx].separator || m.entries[idx].group) {
		idx = -1
	}
	if idx != m.hover {
		m.hover = idx
		m.Refresh()
	}
}

func (m *breezeMenu) MouseOut() {
	m.hover = -1
	m.Refresh()
}

func (m *breezeMenu) Cursor() desktop.Cursor { return desktop.PointerCursor }

type breezeMenuRenderer struct {
	m    *breezeMenu
	objs []fyne.CanvasObject
}

func (r *breezeMenuRenderer) Layout(size fyne.Size) {
	m := r.m
	m.bg.Resize(size)
	if len(m.rows) != len(m.entries) {
		m.rows = make([]*menuRow, len(m.entries))
		for i := range m.rows {
			row := &menuRow{}
			row.hover = newRect(colHover, colTransparent, 0, 4)
			row.hover.Hide()
			row.check = iconImage("checkmark", colText, 16)
			row.check.Hide()
			row.icon = iconImage("folder", colText, 16)
			row.icon.Hide()
			row.label = canvas.NewText("", colText)
			row.label.TextSize = textSize
			row.line = newRect(colBorder, colTransparent, 0, 0)
			row.line.Hide()
			m.rows[i] = row
		}
	}
	y := float32(menuPad)
	objs := []fyne.CanvasObject{m.bg}
	for i, e := range m.entries {
		row := m.rows[i]
		if e.separator {
			row.hover.Hide()
			row.check.Hide()
			row.icon.Hide()
			row.label.Hide()
			row.line.Show()
			row.line.Move(fyne.NewPos(8, y+menuSepH/2))
			row.line.Resize(fyne.NewSize(size.Width-16, 1))
			y += menuSepH
			objs = append(objs, row.line)
			continue
		}
		row.line.Hide()
		row.label.Show()
		row.label.Text = elide(e.label, m.width-58, textSize)
		row.label.Color = colText
		row.label.TextStyle = fyne.TextStyle{}
		if e.disabled {
			row.label.Color = colTextMuted
		}
		if e.group {
			row.label.TextStyle = fyne.TextStyle{Bold: true}
		}
		textX := float32(14)
		if e.checked {
			row.check.Show()
			row.check.Move(fyne.NewPos(12, y+(menuItemH-16)/2))
			row.check.Resize(fyne.NewSize(16, 16))
			row.icon.Hide()
			textX = 36
		} else if e.icon != nil {
			row.icon.Show()
			row.icon.Resource = e.icon
			row.icon.Refresh()
			row.icon.Move(fyne.NewPos(12, y+(menuItemH-16)/2))
			row.icon.Resize(fyne.NewSize(16, 16))
			row.check.Hide()
			textX = 36
		} else {
			row.check.Hide()
			row.icon.Hide()
		}
		row.label.Move(fyne.NewPos(textX, y-2))
		row.label.Resize(fyne.NewSize(m.width-textX-10, 24))
		if i == m.hover {
			row.hover.Show()
			row.hover.Move(fyne.NewPos(4, y+2))
			row.hover.Resize(fyne.NewSize(size.Width-8, menuItemH-4))
		} else {
			row.hover.Hide()
		}
		objs = append(objs, row.hover, row.check, row.icon, row.label)
		y += menuItemH
	}
	r.objs = objs
}

func (r *breezeMenuRenderer) MinSize() fyne.Size { return r.m.MinSize() }

func (r *breezeMenuRenderer) Refresh() {
	r.Layout(r.m.Size())
	canvas.Refresh(r.m)
}

func (r *breezeMenuRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *breezeMenuRenderer) Destroy()                     {}

// showBreezeMenu opens a menu relative to obj. The menu is placed below obj,
// or above it when above is true, and is kept inside the canvas.
func (d *Dialog) showBreezeMenu(entries []menuEntry, obj fyne.CanvasObject, above bool) {
	c := d.menuCanvas()
	if c == nil || len(entries) == 0 {
		return
	}
	panel := newBreezeMenu(entries)
	popup := widget.NewPopUp(panel, c)
	panel.close = popup.Hide

	abs := fyne.CurrentApp().Driver().AbsolutePositionForObject(obj)
	size := panel.MinSize()
	x := abs.X
	y := abs.Y + obj.Size().Height + 2
	if above {
		y = abs.Y - size.Height - 6
	}
	cs := c.Size()
	if x+size.Width > cs.Width-8 {
		x = cs.Width - size.Width - 8
	}
	if x < 8 {
		x = 8
	}
	if y+size.Height > cs.Height-8 {
		y = cs.Height - size.Height - 8
	}
	if y < 8 {
		y = 8
	}
	popup.ShowAtPosition(fyne.NewPos(x, y))
}
