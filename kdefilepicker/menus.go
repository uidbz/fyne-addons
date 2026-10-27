// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

func (d *Dialog) menuCanvas() fyne.Canvas {
	if d.window != nil {
		return d.window.Canvas()
	}
	return fyne.CurrentApp().Driver().CanvasForObject(d)
}

func (d *Dialog) showMenu(menu *fyne.Menu, obj fyne.CanvasObject, above bool) {
	c := d.menuCanvas()
	if c == nil {
		return
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(obj)
	if above {
		pos.Y -= float32(len(menu.Items))*34 + 16
	}
	widget.NewPopUpMenu(menu, c).ShowAtPosition(pos)
}

func (d *Dialog) showFilterMenu() {
	items := make([]*fyne.MenuItem, 0, len(d.filters))
	for i, f := range d.filters {
		idx := i
		it := fyne.NewMenuItem(f.Name, func() {
			d.SetSelectedFilter(idx)
		})
		it.Checked = i == d.filterIdx
		items = append(items, it)
	}
	d.showMenu(fyne.NewMenu("", items...), d.bottom.filter, true)
}

func (d *Dialog) showPathMenu() {
	items := make([]*fyne.MenuItem, 0, len(d.nav.crumbs)+2)
	for _, c := range d.nav.crumbs {
		path := c.path
		it := fyne.NewMenuItem(c.label, func() { d.openDir(path) })
		it.Icon = iconResource("folder", colText)
		items = append(items, it)
	}
	items = append(items, fyne.NewMenuItemSeparator())
	items = append(items, fyne.NewMenuItem("Edit Location…", func() { d.beginPathEdit() }))
	d.showMenu(fyne.NewMenu("", items...), d.nav, false)
}

func (d *Dialog) showSortMenu() {
	field := d.files.sortField
	asc := d.files.ascending
	items := []*fyne.MenuItem{
		checkedItem("Name", field == sortByName, func() { d.files.SetSort(sortByName, asc) }),
		checkedItem("Size", field == sortBySize, func() { d.files.SetSort(sortBySize, asc) }),
		checkedItem("Date", field == sortByDate, func() { d.files.SetSort(sortByDate, asc) }),
		checkedItem("Type", field == sortByType, func() { d.files.SetSort(sortByType, asc) }),
		fyne.NewMenuItemSeparator(),
		checkedItem("Ascending", asc, func() { d.files.SetSort(field, true) }),
		checkedItem("Descending", !asc, func() { d.files.SetSort(field, false) }),
		fyne.NewMenuItemSeparator(),
		checkedItem("Folders First", d.files.foldersFirst, func() {
			d.files.foldersFirst = !d.files.foldersFirst
			d.files.applyFilter()
		}),
	}
	d.showMenu(fyne.NewMenu("", items...), d.toolbar.sort, false)
}

func checkedItem(label string, checked bool, fn func()) *fyne.MenuItem {
	it := fyne.NewMenuItem(label, fn)
	it.Checked = checked
	return it
}

func (d *Dialog) showBookmarksMenu() {
	var items []*fyne.MenuItem
	for _, g := range d.places.model.groups {
		if len(items) > 0 {
			items = append(items, fyne.NewMenuItemSeparator())
		}
		items = append(items, &fyne.MenuItem{Label: g.title, Disabled: true})
		for _, p := range g.places {
			path := p.path
			if path == "" {
				continue
			}
			it := fyne.NewMenuItem(p.name, func() { d.openDir(path) })
			it.Icon = iconResource(folderIconName(p.icon), placeColor(p.icon))
			items = append(items, it)
		}
	}
	d.showMenu(fyne.NewMenu("", items...), d.toolbar.bookmarks, false)
}

func (d *Dialog) showOptionsMenu() {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("New Folder", func() { d.newFolder() }),
		fyne.NewMenuItemSeparator(),
		checkedItem("Show Hidden Files", d.showHidden, func() {
			d.SetShowHidden(!d.showHidden)
		}),
		checkedItem("Show Places Panel", d.showPlaces, func() {
			d.showPlaces = !d.showPlaces
			canvasRefresh(d)
		}),
		checkedItem("Show Quick Filter", d.quickVisible, func() {
			d.setQuickFilter(!d.quickVisible)
		}),
		checkedItem("Show Preview", d.previewVisible, func() {
			d.setPreview(!d.previewVisible)
		}),
	}
	d.showMenu(fyne.NewMenu("", items...), d.toolbar.options, false)
}

func canvasRefresh(obj fyne.CanvasObject) { canvas.Refresh(obj) }

func (d *Dialog) newFolder() {
	base := "New Folder"
	name := base
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(d.currentDir, name)); os.IsNotExist(err) {
			break
		}
		name = base + " " + strconv.Itoa(i)
	}
	path := filepath.Join(d.currentDir, name)
	if err := os.Mkdir(path, 0o755); err != nil {
		d.showError(err.Error())
		return
	}
	d.loadDir(d.currentDir)
	d.files.selectName(name)
	d.bottom.name.SetText(name)
	d.updateOpenState()
}
