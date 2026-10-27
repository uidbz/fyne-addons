// SPDX-License-Identifier: MIT

package kdefilepicker

// ViewMode selects how the file list is presented.
type ViewMode int

const (
	// ViewDetails is the multi-column details view (the default).
	ViewDetails ViewMode = iota
	// ViewCompact is a dense list with a small icon.
	ViewCompact
	// ViewIcons is an icon grid whose zoom is controlled by the slider.
	ViewIcons
)

// SetViewMode changes the file list view.
func (d *Dialog) SetViewMode(mode ViewMode) { d.setView(int(mode)) }

// ViewMode returns the current file list view.
func (d *Dialog) ViewMode() ViewMode { return ViewMode(d.files.viewMode) }

// SelectFile selects the entry with the given file name in the current
// directory. It returns false when no such entry is visible.
func (d *Dialog) SelectFile(name string) bool {
	d.files.selectName(name)
	e, ok := d.files.selectedEntry()
	if !ok || e.name != name {
		return false
	}
	d.bottom.name.SetText(e.name)
	d.updateOpenState()
	d.updatePreview()
	return true
}
