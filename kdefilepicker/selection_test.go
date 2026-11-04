// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func newTestLineEdit(t *testing.T, text string) (*lineEdit, fyne.Window) {
	t.Helper()
	w := test.NewWindow(nil)
	w.Resize(fyne.NewSize(600, 100))
	e := newLineEdit()
	e.SetText(text)
	e.MinSize()
	e.Resize(fyne.NewSize(500, 37))
	e.Move(fyne.NewPos(0, 0))
	w.SetContent(e)
	return e, w
}

func TestLineEditSelectAllAndReplace(t *testing.T) {
	test.NewApp()
	e, w := newTestLineEdit(t, "hello world")
	defer w.Close()

	e.selectAll()
	if got := e.SelectedText(); got != "hello world" {
		t.Fatalf("SelectedText = %q", got)
	}
	e.TypedRune('x')
	if e.Text() != "x" {
		t.Fatalf("typing replaced selection to %q", e.Text())
	}
	if e.hasSelection() {
		t.Fatal("selection should collapse after typing")
	}
}

func TestLineEditDragSelection(t *testing.T) {
	test.NewApp()
	e, w := newTestLineEdit(t, "abcdefg")
	defer w.Close()

	if len(e.widths) != len([]rune("abcdefg"))+1 {
		t.Fatal("width table not built")
	}
	xAt := func(i int) float32 { return editPadX + e.widths[i] }
	dragTo := func(i int, extend bool) {
		ev := &fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(xAt(i), 10)}}
		if extend {
			e.Dragged(ev)
		} else {
			e.DragStart(ev)
		}
	}

	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	dragTo(0, false)
	dragTo(3, true)
	if e.cursor != 3 || e.anchor != 0 {
		t.Fatalf("drag selection = [%d,%d]", e.anchor, e.cursor)
	}
	if got := e.SelectedText(); got != "abc" {
		t.Fatalf("SelectedText after drag = %q", got)
	}
	// A second drag re-anchors.
	dragTo(5, false)
	dragTo(7, true)
	if got := e.SelectedText(); got != "fg" {
		t.Fatalf("SelectedText after second drag = %q", got)
	}
}

func TestLineEditBackspaceCollapsesAndDeletes(t *testing.T) {
	test.NewApp()
	e, w := newTestLineEdit(t, "abcdef")
	defer w.Close()

	e.selectAll()
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e.Text() != "" {
		t.Fatalf("backspace over selection left %q", e.Text())
	}

	e.SetText("abcdef")
	e.selectAll()
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft}) // collapse to start
	if e.cursor != 0 || e.hasSelection() {
		t.Fatalf("arrow collapsed to cursor=%d anchor=%d", e.cursor, e.anchor)
	}
}

func TestLineEditCopySelection(t *testing.T) {
	test.NewApp()
	e, w := newTestLineEdit(t, "copy me please")
	defer w.Close()

	e.selectAll()
	e.TypedShortcut(&fyne.ShortcutCopy{})
	if got := fyne.CurrentApp().Clipboard().Content(); got != "copy me please" {
		t.Fatalf("clipboard after copy-all = %q", got)
	}
	// select just "please" via explicit anchor/cursor:
	e.SetText("copy me please")
	e.cursor, e.anchor = len([]rune("copy me ")), len([]rune("copy me please"))
	e.TypedShortcut(&fyne.ShortcutCopy{})
	if got := fyne.CurrentApp().Clipboard().Content(); got != "please" {
		t.Fatalf("clipboard after copy-selection = %q", got)
	}
	e.TypedShortcut(&fyne.ShortcutCut{})
	if e.Text() != "copy me " {
		t.Fatalf("cut left %q", e.Text())
	}
}

func TestBreadcrumbClickEmptyStartsPathEdit(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	d.noDelayedFocus = true
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))
	d.window = w

	// Layout so crumb geometry exists, then click past the last crumb.
	test.WidgetRenderer(d).Layout(d.Size())
	navSize := d.nav.Size()
	d.nav.Tapped(&fyne.PointEvent{Position: fyne.NewPos(navSize.Width-10, navSize.Height/2)})
	if !d.editing {
		t.Fatal("clicking the empty location bar did not switch to path editing")
	}
	if d.pathEdit.anchor != 0 || d.pathEdit.cursor != len([]rune(d.pathEdit.Text())) {
		t.Fatal("path text should start fully selected")
	}
	d.pathEdit.onEscape()
	if d.editing {
		t.Fatal("escape did not end path editing")
	}
}

func TestDetailsColumnResizeAndSort(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	d := NewOpenDialog(dir)
	d.noDelayedFocus = true
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))
	test.WidgetRenderer(d).Layout(d.Size())
	test.WidgetRenderer(d.files).Layout(d.files.Size())

	v := d.files
	if v.sizeColW != 0 || v.szW() != sizeColumnW {
		t.Fatalf("default size column = %v", v.szW())
	}
	h1 := v.colHandle1
	h1.DragStart(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(500, 10)}})
	h1.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(550, 10)}})
	if want := sizeColumnW - 50; v.szW() != want {
		t.Fatalf("after drag size column = %v, want %v", v.szW(), want)
	}
	h1.DoubleTapped(&fyne.PointEvent{})
	if v.sizeColW != 0 {
		t.Fatal("double-click should reset the column")
	}

	// Header click sorts; clicking Name again flips the direction.
	v.Tapped(&fyne.PointEvent{Position: fyne.NewPos(v.nameEnd(v.Size().Width)+20, 5)})
	if v.sortField != sortBySize {
		t.Fatalf("header click set sortField = %v", v.sortField)
	}
	v.Tapped(&fyne.PointEvent{Position: fyne.NewPos(v.nameEnd(v.Size().Width)+20, 5)})
	if v.ascending {
		t.Fatal("second header click should sort descending")
	}
}

func TestPreviewImageIsAsync(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	// A genuine (tiny) PNG so the background decode can succeed.
	pngPath := filepath.Join(dir, "beta.png")
	imgFile, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(imgFile, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	imgFile.Close()

	d := NewOpenDialog(dir)
	d.noDelayedFocus = true
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 869))

	d.ShowPreview(true)
	if !d.SelectFile("beta.png") {
		t.Fatal("beta.png not selectable")
	}

	// SetFile returns with the icon placeholder visible; the decoded image
	// appears once the background load lands.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if d.preview.img.Visible() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("preview image never arrived")
}
