// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func makeTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".secret"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDetailsSelectionAndAccept(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	idx := -1
	for i, e := range d.files.filtered {
		if e.name == "alpha.txt" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("alpha.txt not listed")
	}
	test.TapAt(d.files, fyne.NewPos(120, headerH+rowH*float32(idx)+10))
	if got := d.bottom.name.Text(); got != "alpha.txt" {
		t.Fatalf("name entry = %q, want alpha.txt", got)
	}
	if !d.bottom.open.enabled {
		t.Fatal("Open button should be enabled after selecting a file")
	}

	var accepted string
	d.SetOnAccept(func(path string) { accepted = path })
	d.Accept()
	if accepted != filepath.Join(dir, "alpha.txt") {
		t.Fatalf("accepted %q", accepted)
	}
}

func TestDoubleClickFolderNavigates(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	idx := -1
	for i, e := range d.files.filtered {
		if e.name == "folder" {
			idx = i
			break
		}
	}
	d.files.DoubleTapped(&fyne.PointEvent{Position: fyne.NewPos(100, headerH+rowH*float32(idx)+10)})
	if d.CurrentDir() != filepath.Join(dir, "folder") {
		t.Fatalf("dir = %q", d.CurrentDir())
	}
}

func TestSaveModeAcceptsNewName(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewSaveDialog(dir, "new.txt")
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	var accepted string
	d.SetOnAccept(func(path string) { accepted = path })
	d.Accept()
	if accepted != filepath.Join(dir, "new.txt") {
		t.Fatalf("accepted %q", accepted)
	}
	if d.openLabel() != "Save" {
		t.Fatalf("button label = %q", d.openLabel())
	}
}

func TestFiltersAndHidden(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	d.SetFilters([]Filter{AllFiles, {Name: "Text", Patterns: []string{"*.txt"}}})
	d.SetSelectedFilter(1)
	for _, e := range d.files.filtered {
		if e.name == "beta.png" {
			t.Fatal("beta.png should be filtered out")
		}
		if e.name == ".secret" {
			t.Fatal(".secret should be hidden by default")
		}
	}
	d.SetSelectedFilter(0)
	d.SetShowHidden(true)
	found := false
	for _, e := range d.files.filtered {
		if e.name == ".secret" {
			found = true
		}
	}
	if !found {
		t.Fatal(".secret should be visible when hidden files are shown")
	}
}

func TestViewModes(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	for _, mode := range []ViewMode{ViewCompact, ViewIcons, ViewDetails} {
		d.SetViewMode(mode)
		if d.ViewMode() != mode {
			t.Fatalf("view mode = %v, want %v", d.ViewMode(), mode)
		}
		if r := test.WidgetRenderer(d.files); r == nil {
			t.Fatal("no renderer")
		}
	}
}

func TestBookmarksParsing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := os.MkdirAll(filepath.Join(home, ".local", "share"), 0o755); err != nil {
		t.Fatal(err)
	}
	xbel := `<?xml version="1.0" encoding="UTF-8"?>
<xbel>
 <bookmark href="file:///tmp">
  <title>Temp</title>
  <info><metadata owner="http://www.kde.org"><IsHidden>false</IsHidden></metadata></info>
 </bookmark>
 <bookmark href="file:///secret">
  <title>Secret</title>
  <info><metadata owner="http://www.kde.org"><IsHidden>true</IsHidden></metadata></info>
 </bookmark>
 <separator href="file:///dev">
  <title>Ignored device</title>
 </separator>
 <bookmark href="remote:/">
  <title>Network</title>
 </bookmark>
</xbel>`
	if err := os.WriteFile(filepath.Join(home, ".local", "share", "user-places.xbel"), []byte(xbel), 0o644); err != nil {
		t.Fatal(err)
	}
	places := loadBookmarks()
	if len(places) < 1 {
		t.Fatalf("places = %#v", places)
	}
	if places[0].name != "Temp" || places[0].path != "/tmp" {
		t.Fatalf("place = %#v", places[0])
	}
	if places[len(places)-1].name != "Home" {
		t.Fatalf("missing Home fallback: %#v", places)
	}
}

func TestFormatting(t *testing.T) {
	if got := formatSize(0, true, 0); got != "0 items" {
		t.Errorf("formatSize dir = %q", got)
	}
	if got := formatSize(1536, false, 0); got != "1.5 KiB" {
		t.Errorf("formatSize file = %q", got)
	}
	ts := time.Date(2026, 10, 22, 19, 24, 0, 0, time.Local)
	if got := formatDate(ts); got != "10/22/26 at 7:24 PM" {
		t.Errorf("formatDate = %q", got)
	}
}

func TestMenusQuickFilterAndPreview(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))
	d.window = w
	d.noDelayedFocus = true

	// Menus use the canvas and absolute object positions.
	d.showFilterMenu()
	d.showSortMenu()
	d.showBookmarksMenu()
	d.showOptionsMenu()
	d.showPathMenu()

	d.setQuickFilter(true)
	if !d.quickVisible {
		t.Fatal("quick filter row not shown")
	}
	d.quick.SetText("alpha")
	if len(d.files.filtered) != 1 || d.files.filtered[0].name != "alpha.txt" {
		t.Fatalf("quick filter = %#v", d.files.filtered)
	}
	d.setQuickFilter(false)

	d.beginPathEdit()
	test.WidgetRenderer(d).Layout(d.Size())
	test.WidgetRenderer(d).Layout(d.Size())
	if !d.editing {
		t.Fatal("path edit did not start")
	}
	d.pathEdit.onSubmitted(filepath.Join(dir, "folder"))
	if d.CurrentDir() != filepath.Join(dir, "folder") {
		t.Fatalf("path edit navigated to %q", d.CurrentDir())
	}

	d.ShowPreview(true)
	if !d.previewVisible {
		t.Fatal("preview not visible")
	}
	d.SelectFile("")
	d.SetViewMode(ViewIcons)
	d.SetViewMode(ViewDetails)
}

func TestTreeExpansionAndSingleClick(t *testing.T) {
	test.NewApp()
	dir := t.TempDir()
	folder := filepath.Join(dir, "folder")
	if err := os.MkdirAll(filepath.Join(folder, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "nested", "deep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	idx := -1
	for i, e := range d.files.filtered {
		if e.name == "folder" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("folder not listed")
	}
	y := headerH + rowH*float32(idx) + 10
	test.TapAt(d.files, fyne.NewPos(10, y))
	found := false
	for _, e := range d.files.filtered {
		if e.name == "nested" && e.depth == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("folder did not expand")
	}
	test.TapAt(d.files, fyne.NewPos(10, y))
	for _, e := range d.files.filtered {
		if e.name == "nested" {
			t.Fatal("folder did not collapse")
		}
	}

	// The folder is still the first row after collapsing.
	d.SetSingleClickDirectories(true)
	test.TapAt(d.files, fyne.NewPos(200, y))
	if d.CurrentDir() != folder {
		t.Fatalf("single-click did not enter folder: %q", d.CurrentDir())
	}
}

func TestScrollbarAndSliderDrag(t *testing.T) {
	test.NewApp()
	dir := t.TempDir()
	for i := 0; i < 80; i++ {
		name := filepath.Join(dir, "file-"+strconv.Itoa(i)+".txt")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	if d.files.maxOffset() <= 0 {
		t.Fatal("test directory should be scrollable")
	}
	x := d.files.Size().Width - 10
	d.files.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(x, headerH+80)}})
	if !d.files.scrollDrag {
		t.Fatal("scrollbar drag did not start")
	}
	d.files.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(x, headerH+300)}})
	d.files.DragEnd()
	if d.files.offset <= 0 {
		t.Fatal("scrollbar drag did not scroll")
	}

	d.SetViewMode(ViewIcons)
	slider := d.toolbar.slider
	slider.Resize(fyne.NewSize(90, 30))
	before := slider.value
	slider.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(15, 15)}})
	slider.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(75, 15)}})
	slider.MouseUp(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(75, 15)}})
	if slider.value <= before {
		t.Fatalf("slider did not move: %v -> %v", before, slider.value)
	}
	if d.files.iconSize <= 16 {
		t.Fatalf("slider did not change icon size: %v", d.files.iconSize)
	}

	d.SetViewMode(ViewDetails)
	if d.files.rowHeight() != rowH {
		t.Fatalf("details row height should start at %v, got %v", rowH, d.files.rowHeight())
	}
	slider.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(55, 15)}})
	slider.MouseUp(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(55, 15)}})
	if d.files.detailIconSize <= 16 {
		t.Fatalf("details zoom did not change icon size: %v", d.files.detailIconSize)
	}
	if d.files.rowHeight() <= rowH {
		t.Fatalf("details rows did not grow: %v", d.files.rowHeight())
	}

	// The renderer must reposition the handle when the value changes.
	test.WidgetRenderer(slider) // ensure sliderRenderer and its circles exist
	slider.setValue(1)
	if slider.outer.Position().X < 60 {
		t.Fatalf("slider handle did not move: %v", slider.outer.Position())
	}

	// A large icon must push the text to the right instead of overlapping it.
	test.WidgetRenderer(d.files).Layout(d.files.Size())
	if len(d.files.rows) == 0 {
		t.Fatal("no file rows laid out")
	}
	row := d.files.rows[0]
	if row.name.Position().X < row.icon.Position().X+row.icon.Size().Width {
		t.Fatalf("name overlaps icon: icon %v width %v name x %v", row.icon.Position(), row.icon.Size().Width, row.name.Position().X)
	}
}
