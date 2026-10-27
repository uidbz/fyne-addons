// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type toolbar struct {
	widget.BaseWidget

	back, forward, up, reload *iconButton
	icons, compact, details   *iconButton
	preview, sort, bookmarks  *iconButton
	zoomOut, zoomIn           *iconButton
	newFolder, options        *iconButton
	search                    *iconButton
	slider                    *zoomSlider
	activeView                int

	bg         *canvas.Rectangle
	separators []*canvas.Rectangle
}

func newToolbar() *toolbar {
	t := &toolbar{activeView: 0}
	t.bg = canvas.NewRectangle(colWindow)
	t.back = newIconButton("go-previous", "Back", nil)
	t.forward = newIconButton("go-next", "Forward", nil)
	t.up = newIconButton("go-up", "Up", nil)
	t.reload = newIconButton("view-refresh", "Reload", nil)
	t.icons = newIconButton("view-list-icons", "Icons", nil)
	t.compact = newIconButton("view-list-details", "Compact", nil)
	t.details = newIconButton("view-list-tree", "Details", nil)
	t.preview = newIconButton("view-preview", "Show Preview", nil)
	t.sort = newIconButton("view-sort", "Sort", nil)
	t.bookmarks = newIconButton("bookmarks", "Bookmarks", nil)
	t.zoomOut = newIconButton("zoom-out", "Zoom Out", nil)
	t.zoomIn = newIconButton("zoom-in", "Zoom In", nil)
	t.newFolder = newIconButton("folder-new", "New Folder", nil)
	t.options = newIconButton("configure", "Options", nil)
	t.search = newIconButton("edit-find", "Search", nil)
	t.slider = newZoomSlider(nil)
	t.separators = []*canvas.Rectangle{
		canvas.NewRectangle(colBorder),
		canvas.NewRectangle(colBorder),
		canvas.NewRectangle(colBorder),
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *toolbar) CreateRenderer() fyne.WidgetRenderer {
	r := &toolbarRenderer{t: t}
	r.Refresh()
	return r
}

func (t *toolbar) MinSize() fyne.Size { return fyne.NewSize(700, toolbarH) }

func (t *toolbar) SetAction(name string, fn func()) {
	switch name {
	case "back":
		t.back.tap = fn
	case "forward":
		t.forward.tap = fn
	case "up":
		t.up.tap = fn
	case "reload":
		t.reload.tap = fn
	case "icons":
		t.icons.tap = fn
	case "compact":
		t.compact.tap = fn
	case "details":
		t.details.tap = fn
	case "preview":
		t.preview.tap = fn
	case "sort":
		t.sort.tap = fn
	case "bookmarks":
		t.bookmarks.tap = fn
	case "zoomout":
		t.zoomOut.tap = fn
	case "zoomin":
		t.zoomIn.tap = fn
	case "newfolder":
		t.newFolder.tap = fn
	case "options":
		t.options.tap = fn
	case "search":
		t.search.tap = fn
	}
}

func (t *toolbar) SetView(mode int) {
	t.activeView = mode
	t.icons.SetChecked(mode == 2)
	t.compact.SetChecked(mode == 1)
	t.details.SetChecked(mode == 0)
}

func (t *toolbar) SetNav(canBack, canForward, canUp bool) {
	t.back.SetEnabled(canBack)
	t.forward.SetEnabled(canForward)
	t.up.SetEnabled(canUp)
}

type toolbarRenderer struct {
	t    *toolbar
	objs []fyne.CanvasObject
}

func (r *toolbarRenderer) Layout(size fyne.Size) {
	r.t.bg.Resize(size)
	// Left group: four navigation buttons at a 34 px pitch.
	x := float32(5)
	place := func(b *iconButton) {
		b.Resize(fyne.NewSize(32, 30))
		b.Move(fyne.NewPos(x, 8))
		x += 34
	}
	place(r.t.back)
	place(r.t.forward)
	place(r.t.up)
	place(r.t.reload)
	r.t.separators[0].Resize(fyne.NewSize(1, 26))
	r.t.separators[0].Move(fyne.NewPos(x+3, 10))
	x += 8
	place(r.t.icons)
	place(r.t.compact)
	place(r.t.details)
	r.t.separators[1].Resize(fyne.NewSize(1, 26))
	r.t.separators[1].Move(fyne.NewPos(x+3, 10))
	x += 8
	place(r.t.preview)
	place(r.t.sort)
	place(r.t.bookmarks)

	// Right group, anchored to the trailing edge.
	optionsX := size.Width - 39
	r.t.options.Resize(fyne.NewSize(32, 30))
	r.t.options.Move(fyne.NewPos(optionsX, 8))
	r.t.separators[2].Resize(fyne.NewSize(1, 26))
	r.t.separators[2].Move(fyne.NewPos(optionsX-5, 10))
	zoomInX := optionsX - 42
	r.t.zoomIn.Resize(fyne.NewSize(32, 30))
	r.t.zoomIn.Move(fyne.NewPos(zoomInX, 8))
	sliderX := zoomInX - 90
	r.t.slider.Resize(fyne.NewSize(90, 30))
	r.t.slider.Move(fyne.NewPos(sliderX, 8))
	zoomOutX := sliderX - 27
	r.t.zoomOut.Resize(fyne.NewSize(30, 30))
	r.t.zoomOut.Move(fyne.NewPos(zoomOutX, 8))
	r.t.search.Hide()
	r.t.newFolder.Hide()
}
func (r *toolbarRenderer) MinSize() fyne.Size { return fyne.NewSize(700, toolbarH) }

func (r *toolbarRenderer) Refresh() {
	r.objs = []fyne.CanvasObject{
		r.t.bg,
		r.t.back, r.t.forward, r.t.up, r.t.reload,
		r.t.separators[0],
		r.t.icons, r.t.compact, r.t.details,
		r.t.separators[1],
		r.t.preview, r.t.sort, r.t.bookmarks,
		r.t.zoomOut, r.t.slider, r.t.zoomIn,
		r.t.separators[2], r.t.options,
	}
	r.Layout(r.t.Size())
	canvas.Refresh(r.t)
}

func (r *toolbarRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *toolbarRenderer) Destroy()                     {}

var _ = desktop.PointerCursor
