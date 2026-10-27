// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

func (d *Dialog) menuCanvas() fyne.Canvas {
	if d.window != nil {
		return d.window.Canvas()
	}
	return fyne.CurrentApp().Driver().CanvasForObject(d)
}

func (d *Dialog) showFilterMenu() {
	entries := make([]menuEntry, 0, len(d.filters))
	for i, f := range d.filters {
		idx := i
		entries = append(entries, checkedEntry(f.Name, i == d.filterIdx, func() {
			d.SetSelectedFilter(idx)
		}))
	}
	d.showBreezeMenu(entries, d.bottom.filter, true)
}

func (d *Dialog) showPathMenu() {
	entries := make([]menuEntry, 0, len(d.nav.crumbs)+2)
	for _, c := range d.nav.crumbs {
		path := c.path
		entries = append(entries, menuEntry{
			label:  c.label,
			icon:   iconResource("folder", colText),
			action: func() { d.openDir(path) },
		})
	}
	entries = append(entries, menuSeparator())
	entries = append(entries, menuEntry{label: "Edit Location…", action: d.beginPathEdit})
	d.showBreezeMenu(entries, d.nav, false)
}

func (d *Dialog) showSortMenu() {
	field := d.files.sortField
	asc := d.files.ascending
	entries := []menuEntry{
		checkedEntry("Name", field == sortByName, func() { d.files.SetSort(sortByName, asc) }),
		checkedEntry("Size", field == sortBySize, func() { d.files.SetSort(sortBySize, asc) }),
		checkedEntry("Date", field == sortByDate, func() { d.files.SetSort(sortByDate, asc) }),
		checkedEntry("Type", field == sortByType, func() { d.files.SetSort(sortByType, asc) }),
		menuSeparator(),
		checkedEntry("Ascending", asc, func() { d.files.SetSort(field, true) }),
		checkedEntry("Descending", !asc, func() { d.files.SetSort(field, false) }),
		menuSeparator(),
		checkedEntry("Folders First", d.files.foldersFirst, func() {
			d.files.foldersFirst = !d.files.foldersFirst
			d.files.applyFilter()
		}),
	}
	d.showBreezeMenu(entries, d.toolbar.sort, false)
}

func (d *Dialog) showBookmarksMenu() {
	var entries []menuEntry
	for _, g := range d.places.model.groups {
		if len(entries) > 0 {
			entries = append(entries, menuSeparator())
		}
		entries = append(entries, menuGroup(g.title))
		for _, p := range g.places {
			if p.path == "" {
				continue
			}
			path := p.path
			entries = append(entries, menuEntry{
				label:  p.name,
				icon:   iconResource(folderIconName(p.icon), placeColor(p.icon)),
				action: func() { d.openDir(path) },
			})
		}
	}
	d.showBreezeMenu(entries, d.toolbar.bookmarks, false)
}

func (d *Dialog) showOptionsMenu() {
	entries := []menuEntry{
		{label: "New Folder", icon: iconResource("folder-new", colText), action: d.newFolder},
		menuSeparator(),
		checkedEntry("Show Hidden Files", d.showHidden, func() {
			d.SetShowHidden(!d.showHidden)
		}),
		checkedEntry("Show Places Panel", d.showPlaces, func() {
			d.showPlaces = !d.showPlaces
			canvasRefresh(d)
		}),
		checkedEntry("Show Quick Filter", d.quickVisible, func() {
			d.setQuickFilter(!d.quickVisible)
		}),
		checkedEntry("Show Preview", d.previewVisible, func() {
			d.setPreview(!d.previewVisible)
		}),
	}
	d.showBreezeMenu(entries, d.toolbar.options, false)
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
