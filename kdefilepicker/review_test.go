// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestUnreadableStartDirectoryDoesNotPanic(t *testing.T) {
	test.NewApp()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("permission checks are bypassed (running as root)")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewOpenDialog panicked: %v", r)
		}
	}()
	d := NewOpenDialog(dir)
	if d.msg == nil || d.msgBg == nil {
		t.Fatal("error banner was not created")
	}
	if !d.msg.Visible() || !d.msgBg.Visible() {
		t.Fatal("error banner should be visible for an unreadable directory")
	}
	w := test.NewWindow(d)
	w.Resize(fyne.NewSize(900, 600))
	w.Close()
	if !d.msg.Visible() || !d.msgBg.Visible() {
		t.Fatal("renderer creation hid the existing error banner")
	}
}

func TestSelectionSurvivesFilterAndClearsWhenFilteredOut(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	if !d.SelectFile("alpha.txt") {
		t.Fatal("could not select alpha.txt")
	}
	d.SetFilters([]Filter{AllFiles, {Name: "Text", Patterns: []string{"*.txt"}}})
	d.SetSelectedFilter(1)
	sel, ok := d.files.selectedEntry()
	if !ok || sel.name != "alpha.txt" {
		t.Fatalf("selection was not preserved: %#v, %v", sel, ok)
	}
	if d.bottom.name.Text() != "alpha.txt" {
		t.Fatalf("name field = %q", d.bottom.name.Text())
	}

	d.SetSelectedFilter(0)
	d.SetFilters([]Filter{AllFiles, {Name: "PNG", Patterns: []string{"*.png"}}})
	d.SetSelectedFilter(1)
	if _, ok := d.files.selectedEntry(); ok {
		t.Fatal("selection should be cleared when the entry is filtered out")
	}
	if d.files.selected != -1 {
		t.Fatalf("selected index = %d, want -1", d.files.selected)
	}
}

func TestSortComparatorDescendingIsDeterministic(t *testing.T) {
	test.NewApp()
	v := newFileView()
	v.entries = []fileEntry{
		{name: "b.txt", size: 10},
		{name: "a.txt", size: 10},
		{name: "c.png", size: 10},
	}
	v.SetSort(sortBySize, false)
	got := []string{v.filtered[0].name, v.filtered[1].name, v.filtered[2].name}
	want := []string{"c.png", "b.txt", "a.txt"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("size desc = %v, want %v", got, want)
	}

	v.SetSort(sortByType, false)
	got = []string{v.filtered[0].name, v.filtered[1].name, v.filtered[2].name}
	// Descending extension order puts .txt before .png.
	want = []string{"b.txt", "a.txt", "c.png"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("type desc = %v, want %v", got, want)
	}
}

func TestIconGridHoverUsesActualColumn(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))
	d.SetViewMode(ViewIcons)
	test.WidgetRenderer(d.files).Layout(d.files.Size())

	cols, cellW, cellH := d.files.grid(d.files.Size())
	if cols < 2 {
		t.Fatalf("icon view has %d columns, want at least 2", cols)
	}
	if len(d.files.filtered) < 2 {
		t.Fatalf("need at least two entries, got %d", len(d.files.filtered))
	}
	d.files.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{
		Position: fyne.NewPos(cellW+cellW/2, cellH/2),
	}})
	if d.files.hover != 1 {
		t.Fatalf("hover = %d, want 1", d.files.hover)
	}
}

func TestSetFiltersResetsSelectionAndInvalidViewModeIsIgnored(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	d.SetFilters([]Filter{AllFiles, {Name: "Text", Patterns: []string{"*.txt"}}})
	d.SetSelectedFilter(1)
	d.SetFilters([]Filter{AllFiles, {Name: "Images", Patterns: []string{"*.png"}}})
	if d.SelectedFilter() != 0 {
		t.Fatalf("filter index = %d, want 0", d.SelectedFilter())
	}
	if d.bottom.filter.value != "All Files" {
		t.Fatalf("filter label = %q", d.bottom.filter.value)
	}

	d.SetViewMode(ViewDetails)
	d.SetViewMode(ViewMode(99))
	if d.ViewMode() != ViewDetails {
		t.Fatalf("view mode = %v, want details", d.ViewMode())
	}
}

func TestNewFolderUsesNextFreeName(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	for _, name := range []string{"New Folder", "New Folder 1"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	d := NewOpenDialog(dir)
	d.newFolder()
	if _, err := os.Stat(filepath.Join(dir, "New Folder 2")); err != nil {
		t.Fatalf("New Folder 2 was not created: %v", err)
	}
	if d.bottom.name.Text() != "New Folder 2" {
		t.Fatalf("name field = %q", d.bottom.name.Text())
	}
}

func TestPlacesReloadDoesNotDuplicateRows(t *testing.T) {
	test.NewApp()
	m := &placesModel{}
	m.reload()
	first := len(m.flat)
	if first == 0 {
		t.Fatal("no places loaded")
	}
	m.reload()
	if len(m.flat) != first {
		t.Fatalf("flat places grew from %d to %d on reload", first, len(m.flat))
	}
}

func TestBookmarksInNestedFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	if err := os.MkdirAll(filepath.Join(home, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	xbel := `<?xml version="1.0" encoding="UTF-8"?>
<xbel version="1.0">
 <folder>
  <title>Grouped</title>
  <bookmark href="file:///tmp/nested">
   <title>Nested</title>
   <info><metadata owner="http://freedesktop.org"><bookmark:icon name="folder-blue"/></metadata></info>
  </bookmark>
 </folder>
</xbel>`
	if err := os.WriteFile(filepath.Join(home, "data", "user-places.xbel"), []byte(xbel), 0o644); err != nil {
		t.Fatal(err)
	}
	places := loadBookmarks()
	if len(places) == 0 || places[0].name != "Nested" || places[0].path != "/tmp/nested" {
		t.Fatalf("nested bookmark = %#v", places)
	}
	if places[0].icon != "folder-blue" {
		t.Fatalf("icon = %q, want folder-blue", places[0].icon)
	}
	if places[len(places)-1].name != "Home" {
		t.Fatalf("missing Home fallback: %#v", places)
	}
}

// visibleText exercises the renderer's windowing logic with a detached
// line edit; the production entry point is lineEditRenderer.visibleText.
func visibleText(full string, cursor int, avail float32) (string, int) {
	e := newLineEdit()
	e.SetText(full)
	e.cursor = cursor
	e.anchor = cursor
	return (&lineEditRenderer{e: e}).visibleText(avail)
}

func TestElideAndVisibleText(t *testing.T) {
	test.NewApp()
	if got := elide("short", 500, textSize); got != "short" {
		t.Fatalf("elide short = %q", got)
	}
	if got := elide("a very long file name.txt", 40, textSize); got == "" || !strings.HasSuffix(got, "…") {
		t.Fatalf("elide long = %q", got)
	}
	long := "a/very/long/path/that/does/not/fit"
	display, start := visibleText(long, len([]rune(long)), 60)
	if !strings.HasSuffix(long, display) || start == 0 {
		t.Fatalf("visibleText tail = %q start %d", display, start)
	}
	display, start = visibleText(long, 0, 60)
	if !strings.HasPrefix(long, display) || start != 0 {
		t.Fatalf("visibleText head = %q start %d", display, start)
	}
	if display, start := visibleText("tiny", 2, 500); display != "tiny" || start != 0 {
		t.Fatalf("visibleText fit = %q start %d", display, start)
	}
}

func TestPreviewSVGFallback(t *testing.T) {
	test.NewApp()
	// An SVG that cannot be read must fall back to the MIME icon instead of
	// leaving a stale image on screen.
	path := filepath.Join(t.TempDir(), "missing.svg")
	p := newPreviewPane()
	test.WidgetRenderer(p)
	p.SetFile(path, fileEntry{name: "missing.svg", path: path})
	if !p.icon.Visible() {
		t.Fatal("icon fallback should be visible")
	}
	if p.img.Visible() {
		t.Fatal("image preview should be hidden")
	}
}

func TestBreadcrumbFitsAndFirstCrumbWins(t *testing.T) {
	test.NewApp()
	b := newBreadcrumb()
	b.SetPath("/usr/share/doc/package/deep/sub/dir")
	b.Resize(fyne.NewSize(200, 37))
	test.WidgetRenderer(b).Layout(b.Size())
	if len(b.display) == 0 {
		t.Fatal("no crumbs displayed")
	}
	last := b.display[len(b.display)-1]
	if last.x+last.w > 200 {
		t.Fatalf("last crumb overflows: x=%v w=%v", last.x, last.w)
	}

	b = newBreadcrumb()
	b.SetPath("/tmp/a/b")
	b.Resize(fyne.NewSize(500, 37))
	test.WidgetRenderer(b).Layout(b.Size())
	first := b.display[0]
	navigated := ""
	b.onNavigate = func(path string) { navigated = path }
	b.onDropdown = func() { navigated = "dropdown" }
	b.Tapped(&fyne.PointEvent{Position: fyne.NewPos(first.x+1, 10)})
	if navigated != first.path {
		t.Fatalf("first crumb tapped %q, want %q", navigated, first.path)
	}
}

func TestRefreshDirPreservesNameSelectionAndOffset(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	for i := 0; i < 60; i++ {
		if err := os.WriteFile(filepath.Join(dir, "extra-"+string(rune('a'+i%26))+"-"+string(rune('a'+i/26))+".txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 600))

	if !d.SelectFile("alpha.txt") {
		t.Fatal("could not select alpha.txt")
	}
	d.SetFileName("typed.txt")
	d.files.offset = 20
	d.RefreshDir()
	if got := d.bottom.name.Text(); got != "typed.txt" {
		t.Fatalf("name after refresh = %q", got)
	}
	if sel, ok := d.files.selectedEntry(); !ok || sel.name != "alpha.txt" {
		t.Fatalf("selection after refresh = %#v, %v", sel, ok)
	}
	if d.files.offset != 20 {
		t.Fatalf("offset after refresh = %v, want 20", d.files.offset)
	}
}

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	if got := resolvePath("~/alpha.txt", "/tmp"); got != filepath.Join(home, "alpha.txt") {
		t.Fatalf("~ path = %q", got)
	}
	if got := resolvePath("sub/alpha.txt", "/tmp/base"); got != filepath.Join("/tmp/base", "sub", "alpha.txt") {
		t.Fatalf("relative path = %q", got)
	}
	if got := resolvePath("/absolute/alpha.txt", "/tmp"); got != "/absolute/alpha.txt" {
		t.Fatalf("absolute path = %q", got)
	}
}

func TestThemeHonoursMonospaceAndSymbolStyles(t *testing.T) {
	for _, style := range []fyne.TextStyle{{Monospace: true}, {Symbol: true}} {
		got := NewTheme().Font(style)
		want := theme.DefaultTheme().Font(style)
		if got == nil || want == nil || got.Name() != want.Name() {
			t.Fatalf("style %#v returned %v, want %v", style, got, want)
		}
	}
}

func TestParseHumanSize(t *testing.T) {
	cases := map[string]uint64{
		"":       0,
		"0B":     0,
		"512B":   512,
		"1K":     1 << 10,
		"1.5K":   1536,
		"2G":     2 << 30,
		"1.5GiB": 3 << 29,
		"junk":   0,
	}
	for input, want := range cases {
		if got := parseHumanSize(input); got != want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestReadEntriesCountsChildDirectories(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(sub, "f"+string(rune('a'+i))), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := readEntries(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].itemCount != 3 {
		t.Fatalf("entries = %#v", entries)
	}
	entries, err = readEntries(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].itemCount != 0 {
		t.Fatalf("uncounted entries = %#v", entries)
	}
}

func TestRowPoolShrinksAfterViewChange(t *testing.T) {
	test.NewApp()
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f"+string(rune('a'+i))+".txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d := NewOpenDialog(dir)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	d.files.iconSize = 16
	d.SetViewMode(ViewIcons)
	test.WidgetRenderer(d.files).Layout(d.files.Size())
	iconRows := len(d.files.rows)

	d.SetViewMode(ViewDetails)
	test.WidgetRenderer(d.files).Layout(d.files.Size())
	if len(d.files.rows) >= iconRows {
		t.Fatalf("row pool did not shrink: icons %d, details %d", iconRows, len(d.files.rows))
	}
	if len(d.files.rows) > 49 {
		t.Fatalf("details row pool = %d, want at most 49", len(d.files.rows))
	}
}
