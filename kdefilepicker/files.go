// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type sortField int

const (
	sortByName sortField = iota
	sortBySize
	sortByDate
	sortByType
)

type fileEntry struct {
	name      string
	path      string
	isDir     bool
	isLink    bool
	size      int64
	modTime   time.Time
	itemCount int
}

type fileView struct {
	widget.BaseWidget

	dir          string
	entries      []fileEntry
	filtered     []fileEntry
	selected     int
	hover        int
	offset       float32
	focused      bool
	showHidden   bool
	sortField    sortField
	ascending    bool
	foldersFirst bool
	filterText   string
	patterns     []string

	viewMode   int // 0 details, 1 compact, 2 icons
	iconSize   float32
	onSelect   func(fileEntry)
	onOpenDir  func(string)
	onActivate func(fileEntry)
	onEscape   func()

	bg       *canvas.Rectangle
	rows     []*fileRow
	hdrBg    *canvas.Rectangle
	hdrTop   *canvas.Rectangle
	hdrDiv1  *canvas.Rectangle
	hdrDiv2  *canvas.Rectangle
	hdrName  *canvas.Text
	hdrSize  *canvas.Text
	hdrDate  *canvas.Text
	sortIcon *canvas.Image
	scrollBg *canvas.Rectangle
	scroll   *canvas.Rectangle
}

type fileRow struct {
	bg   *canvas.Rectangle
	sel  *canvas.Rectangle
	exp  *canvas.Image
	icon *canvas.Image
	name *canvas.Text
	size *canvas.Text
	date *canvas.Text
}

func newFileView(dir string) *fileView {
	v := &fileView{
		dir:          dir,
		selected:     -1,
		hover:        -1,
		ascending:    true,
		foldersFirst: true,
		iconSize:     16,
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *fileView) CreateRenderer() fyne.WidgetRenderer {
	v.bg = canvas.NewRectangle(colView)
	v.hdrBg = canvas.NewRectangle(colHeader)
	v.hdrTop = newRect(colBorder, colTransparent, 0, 0)
	v.hdrDiv1 = newRect(colBorder, colTransparent, 0, 0)
	v.hdrDiv2 = newRect(colBorder, colTransparent, 0, 0)
	v.hdrName = canvas.NewText("Name", colText)
	v.hdrSize = canvas.NewText("Size", colText)
	v.hdrDate = canvas.NewText("Date", colText)
	for _, t := range []*canvas.Text{v.hdrName, v.hdrSize, v.hdrDate} {
		t.TextSize = textSize
	}
	v.sortIcon = iconImage("go-up", colTextMuted, 12)
	v.scrollBg = canvas.NewRectangle(colTransparent)
	v.scroll = newRect(colScroll, colTransparent, 0, 3.5)
	r := &fileViewRenderer{v: v}
	r.Refresh()
	return r
}

func (v *fileView) MinSize() fyne.Size { return fyne.NewSize(500, 300) }

func (v *fileView) SetDir(dir string, entries []fileEntry) {
	v.dir = dir
	v.entries = entries
	v.selected = -1
	v.hover = -1
	v.offset = 0
	v.applyFilter()
}

func (v *fileView) SetShowHidden(show bool) {
	v.showHidden = show
	v.applyFilter()
}

func (v *fileView) SetPatterns(patterns []string) {
	v.patterns = patterns
	v.applyFilter()
}

func (v *fileView) SetFilter(text string) {
	v.filterText = strings.ToLower(text)
	v.applyFilter()
}

func (v *fileView) SetSort(field sortField, ascending bool) {
	v.sortField = field
	v.ascending = ascending
	v.applyFilter()
}

func (v *fileView) applyFilter() {
	filtered := make([]fileEntry, 0, len(v.entries))
	for _, e := range v.entries {
		if !v.showHidden && strings.HasPrefix(e.name, ".") {
			continue
		}
		if v.filterText != "" && !strings.Contains(strings.ToLower(e.name), v.filterText) {
			continue
		}
		if !entryMatches(e, v.patterns) {
			continue
		}
		filtered = append(filtered, e)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		if v.foldersFirst && a.isDir != b.isDir {
			return a.isDir
		}
		var less bool
		switch v.sortField {
		case sortBySize:
			less = a.size < b.size
		case sortByDate:
			less = a.modTime.Before(b.modTime)
		case sortByType:
			less = strings.ToLower(filepath.Ext(a.name)) < strings.ToLower(filepath.Ext(b.name))
		default:
			less = strings.ToLower(a.name) < strings.ToLower(b.name)
		}
		if v.ascending {
			return less
		}
		return !less
	})
	v.filtered = filtered
	if v.selected >= len(v.filtered) {
		v.selected = -1
	}
	v.clampOffset()
	v.Refresh()
}

func (v *fileView) entryAt(y float32) int {
	return v.entryAtPoint(0, y)
}

func (v *fileView) entryAtPoint(x, y float32) int {
	if v.viewMode == 2 {
		cols, cellW, cellH := v.grid(sizeOf(v))
		if cols < 1 {
			return -1
		}
		row := int((y + v.offset) / cellH)
		col := int(x / cellW)
		if col < 0 || col >= cols {
			return -1
		}
		idx := row*cols + col
		if idx < 0 || idx >= len(v.filtered) {
			return -1
		}
		return idx
	}
	hdr := float32(0)
	if v.viewMode == 0 {
		hdr = headerH
	}
	if y < hdr {
		return -1
	}
	rh := v.rowHeight()
	idx := int((y - hdr + v.offset) / rh)
	if idx < 0 || idx >= len(v.filtered) {
		return -1
	}
	return idx
}

func (v *fileView) rowHeight() float32 {
	if v.viewMode == 1 {
		return 26
	}
	return rowH
}

func sizeOf(v *fileView) fyne.Size { return v.Size() }

func (v *fileView) grid(sz fyne.Size) (int, float32, float32) {
	contentW := sz.Width - scrollbarW
	minW := v.iconSize + 28
	if minW < 88 {
		minW = 88
	}
	cols := int(contentW / minW)
	if cols < 1 {
		cols = 1
	}
	cellW := contentW / float32(cols)
	cellH := v.iconSize + 42
	return cols, cellW, cellH
}

func (v *fileView) Tapped(e *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(v); c != nil {
		c.Focus(v)
	}
	idx := v.entryAtPoint(e.Position.X, e.Position.Y)
	if idx < 0 {
		return
	}
	v.selected = idx
	if v.onSelect != nil {
		v.onSelect(v.filtered[idx])
	}
	v.Refresh()
}

func (v *fileView) DoubleTapped(e *fyne.PointEvent) {
	idx := v.entryAtPoint(e.Position.X, e.Position.Y)
	if idx < 0 {
		return
	}
	v.selected = idx
	e2 := v.filtered[idx]
	if e2.isDir {
		if v.onOpenDir != nil {
			v.onOpenDir(e2.path)
		}
		return
	}
	if v.onSelect != nil {
		v.onSelect(e2)
	}
	if v.onActivate != nil {
		v.onActivate(e2)
	}
}

func (v *fileView) MouseMoved(e *desktop.MouseEvent) {
	idx := v.entryAt(e.Position.Y)
	if idx != v.hover {
		v.hover = idx
		v.Refresh()
	}
}

func (v *fileView) MouseOut() {
	v.hover = -1
	v.Refresh()
}

func (v *fileView) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (v *fileView) TypedRune(rune) {}

func (v *fileView) FocusGained() { v.focused = true; v.Refresh() }
func (v *fileView) FocusLost()   { v.focused = false; v.Refresh() }

func (v *fileView) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyDown:
		if v.selected < len(v.filtered)-1 {
			v.selected++
			v.scrollToSelected()
		}
	case fyne.KeyUp:
		if v.selected > 0 {
			v.selected--
			v.scrollToSelected()
		}
	case fyne.KeyHome:
		v.selected = 0
		v.scrollToSelected()
	case fyne.KeyEnd:
		v.selected = len(v.filtered) - 1
		v.scrollToSelected()
	case fyne.KeyPageDown:
		v.offset += float32(v.visibleRows()) * v.rowHeight()
		v.clampOffset()
	case fyne.KeyPageUp:
		v.offset -= float32(v.visibleRows()) * v.rowHeight()
		v.clampOffset()
	case fyne.KeyEscape:
		if v.onEscape != nil {
			v.onEscape()
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if v.selected >= 0 && v.selected < len(v.filtered) {
			e := v.filtered[v.selected]
			if e.isDir {
				if v.onOpenDir != nil {
					v.onOpenDir(e.path)
				}
			} else if v.onActivate != nil {
				v.onActivate(e)
			}
		}
	}
	if v.selected >= 0 && v.selected < len(v.filtered) && v.onSelect != nil {
		v.onSelect(v.filtered[v.selected])
	}
	v.Refresh()
}

func (v *fileView) Scrolled(ev *fyne.ScrollEvent) {
	v.offset -= ev.Scrolled.DY
	v.clampOffset()
	v.Refresh()
}

func (v *fileView) visibleRows() int {
	h := v.Size().Height - 8
	if v.viewMode == 0 {
		h -= headerH
	}
	if v.viewMode == 2 {
		_, _, cellH := v.grid(v.Size())
		if cellH < 1 {
			return 0
		}
		return int(h / cellH)
	}
	rh := v.rowHeight()
	if h < rh {
		return 0
	}
	return int(h / rh)
}

func (v *fileView) floorOffset() float32 { return 0 }
func (v *fileView) maxOffset() float32 {
	viewH := v.Size().Height - 8
	if v.viewMode == 0 {
		viewH -= headerH
	}
	var content float32
	var rh float32
	if v.viewMode == 2 {
		cols, _, cellH := v.grid(v.Size())
		if cols < 1 {
			cols = 1
		}
		rows := (len(v.filtered) + cols - 1) / cols
		content = float32(rows) * cellH
	} else {
		rh = v.rowHeight()
		content = float32(len(v.filtered)) * rh
	}
	m := content - viewH
	if m < 0 {
		return 0
	}
	return m
}

func (v *fileView) clampOffset() {
	v.offset = clamp(v.offset, v.floorOffset(), v.maxOffset())
}

func (v *fileView) scrollToSelected() {
	if v.selected < 0 {
		return
	}
	viewH := v.Size().Height - 8
	if v.viewMode == 0 {
		viewH -= headerH
	}
	var top, height float32
	if v.viewMode == 2 {
		cols, _, cellH := v.grid(v.Size())
		if cols < 1 {
			cols = 1
		}
		row := v.selected / cols
		top = float32(row) * cellH
		height = cellH
	} else {
		height = v.rowHeight()
		top = float32(v.selected) * height
	}
	if top < v.offset {
		v.offset = top
	}
	if top+height > v.offset+viewH {
		v.offset = top + height - viewH
	}
	v.clampOffset()
}

// selectName selects the entry with the given name, if present.
func (v *fileView) selectName(name string) {
	for i, e := range v.filtered {
		if e.name == name {
			v.selected = i
			v.scrollToSelected()
			v.Refresh()
			return
		}
	}
}

func (v *fileView) selectedEntry() (fileEntry, bool) {
	if v.selected < 0 || v.selected >= len(v.filtered) {
		return fileEntry{}, false
	}
	return v.filtered[v.selected], true
}

func (v *fileView) clearSelection() {
	v.selected = -1
	v.Refresh()
}

// ---------------------------------------------------------------------------
// Renderer.

type fileViewRenderer struct {
	v    *fileView
	objs []fyne.CanvasObject
}

func (r *fileViewRenderer) rowObjects() []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, len(r.v.rows)*7)
	for _, row := range r.v.rows {
		out = append(out, row.bg, row.sel, row.exp, row.icon, row.name, row.size, row.date)
	}
	return out
}

func (r *fileViewRenderer) ensureRows(n int) {
	v := r.v
	for len(v.rows) < n {
		row := &fileRow{}
		row.bg = canvas.NewRectangle(colView)
		row.sel = newRect(colTransparent, colTransparent, 0, 4)
		row.exp = iconImage("go-next", colTextMuted, 10)
		row.icon = iconImage("folder", colText, 16)
		row.name = canvas.NewText("", colText)
		row.name.TextSize = textSize
		row.size = canvas.NewText("", colText)
		row.size.TextSize = textSize
		row.date = canvas.NewText("", colText)
		row.date.TextSize = textSize
		v.rows = append(v.rows, row)
	}
}

func (r *fileViewRenderer) Layout(size fyne.Size) {
	v := r.v
	v.bg.Resize(size)
	v.bg.Move(fyne.NewPos(0, 0))

	header := float32(0)
	if v.viewMode == 0 {
		header = headerH
		r.layoutHeader(size)
	} else {
		r.hideHeader()
	}

	visibleH := size.Height - header - 8
	if visibleH < 1 {
		visibleH = 1
	}

	slot := 0
	if v.viewMode == 2 {
		cols, cellW, cellH := v.grid(size)
		if cols < 1 {
			cols = 1
		}
		firstRow := int(v.offset / cellH)
		if firstRow < 0 {
			firstRow = 0
		}
		y := header - (v.offset - float32(firstRow)*cellH)
		rows := int(visibleH/cellH) + 3
		count := cols * rows
		if count < 1 {
			count = 1
		}
		r.ensureRows(count)
		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				idx := (firstRow+row)*cols + col
				rr := v.rows[slot]
				if idx >= len(v.filtered) {
					r.hideRow(rr)
					slot++
					continue
				}
				x := float32(col) * cellW
				ry := y + float32(row)*cellH
				r.layoutIconRow(rr, idx, v.filtered[idx], x, ry, cellW, cellH, visibleH, header)
				slot++
			}
		}
	} else {
		rh := v.rowHeight()
		first := int(v.offset / rh)
		if first < 0 {
			first = 0
		}
		y := header - (v.offset - float32(first)*rh)
		count := int(visibleH/rh) + 3
		if count < 1 {
			count = 1
		}
		r.ensureRows(count)
		for idx := first; idx < len(v.filtered) && slot < count; idx, slot = idx+1, slot+1 {
			row := v.rows[slot]
			rowY := y + float32(slot)*rh
			if v.viewMode == 1 {
				r.layoutCompactRow(row, idx, v.filtered[idx], rowY, size, visibleH, header)
			} else {
				r.layoutRow(row, idx, v.filtered[idx], rowY, size, visibleH)
			}
		}
	}
	for ; slot < len(v.rows); slot++ {
		r.hideRow(v.rows[slot])
	}

	if v.maxOffset() <= 0 {
		v.scrollBg.Hide()
		v.scroll.Hide()
	} else {
		v.scrollBg.Show()
		v.scrollBg.Resize(fyne.NewSize(scrollbarW, visibleH))
		v.scrollBg.Move(fyne.NewPos(size.Width-scrollbarW, header))
		v.scrollBg.FillColor = colView
		v.scroll.Show()
		content := v.maxOffset() + visibleH
		thumbH := visibleH * visibleH / content
		if thumbH < 24 {
			thumbH = 24
		}
		frac := v.offset / v.maxOffset()
		thumbY := header + frac*(visibleH-thumbH)
		v.scroll.Resize(fyne.NewSize(7, thumbH))
		v.scroll.Move(fyne.NewPos(size.Width-16, thumbY))
	}

	r.objs = []fyne.CanvasObject{v.bg}
	r.objs = append(r.objs, r.rowObjects()...)
	if v.viewMode == 0 {
		r.objs = append(r.objs,
			v.hdrBg, v.hdrTop, v.hdrDiv1, v.hdrDiv2,
			v.hdrName, v.hdrSize, v.hdrDate, v.sortIcon,
		)
	}
	r.objs = append(r.objs, v.scrollBg, v.scroll)
}

func (r *fileViewRenderer) layoutHeader(size fyne.Size) {
	v := r.v
	for _, o := range []fyne.CanvasObject{v.hdrBg, v.hdrTop, v.hdrDiv1, v.hdrDiv2, v.hdrName, v.hdrSize, v.hdrDate, v.sortIcon} {
		o.Show()
	}
	headerW := size.Width - scrollbarW
	v.hdrBg.Resize(fyne.NewSize(headerW, headerH))
	v.hdrBg.Move(fyne.NewPos(0, 0))
	v.hdrTop.Resize(fyne.NewSize(headerW, 1))
	v.hdrTop.Move(fyne.NewPos(0, 0))
	v.hdrDiv1.Resize(fyne.NewSize(1, headerH))
	v.hdrDiv1.Move(fyne.NewPos(nameColumnW(size.Width), 0))
	v.hdrDiv2.Resize(fyne.NewSize(1, headerH))
	v.hdrDiv2.Move(fyne.NewPos(nameColumnW(size.Width)+sizeColumnW, 0))
	hdrY := centerY(headerH, measureHeight())
	v.hdrName.Move(fyne.NewPos(3, hdrY+1))
	v.hdrSize.Move(fyne.NewPos(nameColumnW(size.Width)+6, hdrY+1))
	v.hdrDate.Move(fyne.NewPos(nameColumnW(size.Width)+sizeColumnW+7, hdrY+1))
	v.hdrName.Resize(fyne.NewSize(nameColumnW(size.Width)-40, 18))
	v.hdrSize.Resize(fyne.NewSize(sizeColumnW, 18))
	v.hdrDate.Resize(fyne.NewSize(100, 18))
	v.sortIcon.Resize(fyne.NewSize(12, 12))
	v.sortIcon.Move(fyne.NewPos(nameColumnW(size.Width)-22, (headerH-12)/2))
}

func (r *fileViewRenderer) hideHeader() {
	r.v.hdrBg.Hide()
	r.v.hdrTop.Hide()
	r.v.hdrDiv1.Hide()
	r.v.hdrDiv2.Hide()
	r.v.hdrName.Hide()
	r.v.hdrSize.Hide()
	r.v.hdrDate.Hide()
	r.v.sortIcon.Hide()
}

func (r *fileViewRenderer) layoutCompactRow(row *fileRow, idx int, e fileEntry, y float32, size fyne.Size, visibleH, header float32) {
	v := r.v
	if y+26 < header || y > header+visibleH {
		r.hideRow(row)
		return
	}
	rowBgColor := colView
	if idx%2 == 1 {
		rowBgColor = colViewAlternate
	}
	if idx == v.hover && idx != v.selected {
		rowBgColor = colHover
	}
	row.bg.FillColor = rowBgColor
	row.bg.Show()
	row.bg.Resize(fyne.NewSize(size.Width-scrollbarW, 26))
	row.bg.Move(fyne.NewPos(0, y))
	row.sel.Hide()
	if idx == v.selected {
		row.sel.Show()
		row.sel.Move(fyne.NewPos(1, y+2))
		row.sel.Resize(fyne.NewSize(size.Width-scrollbarW-2, 22))
		if v.focused {
			row.sel.FillColor = colAccent
			row.sel.StrokeWidth = 0
		} else {
			row.sel.FillColor = colSelection
			row.sel.StrokeColor = colSelectionEdge
			row.sel.StrokeWidth = 1
		}
		row.sel.Refresh()
	}
	row.exp.Hide()
	row.icon.Show()
	row.icon.Resource = iconResource(fileIconName(e.name, e.isDir), colText)
	row.icon.Refresh()
	row.icon.Resize(fyne.NewSize(16, 16))
	row.icon.Move(fyne.NewPos(8, y+5))
	row.name.Show()
	row.name.Text = elide(e.name, size.Width-scrollbarW-160, textSize)
	row.name.Color = colText
	row.name.Alignment = fyne.TextAlignLeading
	row.name.Move(fyne.NewPos(30, y-2))
	row.name.Resize(fyne.NewSize(size.Width-scrollbarW-150, 24))
	row.name.Refresh()
	row.size.Show()
	row.size.Text = formatSize(e.size, e.isDir, e.itemCount)
	row.size.Alignment = fyne.TextAlignTrailing
	row.size.Color = colText
	row.size.Move(fyne.NewPos(size.Width-scrollbarW-140, y-2))
	row.size.Resize(fyne.NewSize(130, 24))
	row.size.Refresh()
	row.date.Hide()
}

func (r *fileViewRenderer) layoutIconRow(row *fileRow, idx int, e fileEntry, x, y, cellW, cellH, visibleH, header float32) {
	v := r.v
	iconSize := v.iconSize
	if iconSize < 16 {
		iconSize = 64
	}
	row.bg.FillColor = colView
	row.bg.Show()
	row.bg.Resize(fyne.NewSize(cellW, cellH))
	row.bg.Move(fyne.NewPos(x, y))

	row.sel.Show()
	row.sel.Move(fyne.NewPos(x+4, y+2))
	row.sel.Resize(fyne.NewSize(cellW-8, cellH-4))
	switch {
	case idx == v.selected && v.focused:
		row.sel.FillColor = colAccent
		row.sel.StrokeColor = colAccent
		row.sel.StrokeWidth = 0
	case idx == v.selected:
		row.sel.FillColor = colSelection
		row.sel.StrokeColor = colSelectionEdge
		row.sel.StrokeWidth = 1
	case idx == v.hover:
		row.sel.FillColor = colHover
		row.sel.StrokeColor = colTransparent
		row.sel.StrokeWidth = 0
	default:
		row.sel.Hide()
	}
	row.sel.Refresh()

	row.exp.Hide()
	row.icon.Show()
	row.icon.Resource = iconResource(fileIconName(e.name, e.isDir), colText)
	row.icon.Refresh()
	row.icon.Resize(fyne.NewSize(iconSize, iconSize))
	row.icon.Move(fyne.NewPos(x+(cellW-iconSize)/2, y+10))

	row.name.Show()
	row.name.Text = elide(e.name, cellW-8, textSize)
	row.name.Color = colText
	row.name.Alignment = fyne.TextAlignCenter
	row.name.Move(fyne.NewPos(x+4, y+iconSize+14))
	row.name.Resize(fyne.NewSize(cellW-8, 24))
	row.name.Refresh()
	row.size.Hide()
	row.date.Hide()
}

func (r *fileViewRenderer) layoutRow(row *fileRow, idx int, e fileEntry, y float32, size fyne.Size, visibleH float32) {
	v := r.v
	if y+rowH < headerH || y > headerH+visibleH {
		r.hideRow(row)
		return
	}
	rowBgColor := colView
	if idx%2 == 1 {
		rowBgColor = colViewAlternate
	}
	if idx == v.hover && idx != v.selected {
		rowBgColor = colHover
	}
	row.bg.FillColor = rowBgColor
	row.bg.Show()
	row.bg.Resize(fyne.NewSize(size.Width-scrollbarW, rowH))
	row.bg.Move(fyne.NewPos(0, y))

	row.sel.Hide()
	if idx == v.selected {
		row.sel.Show()
		row.sel.Move(fyne.NewPos(1, y+2))
		row.sel.Resize(fyne.NewSize(size.Width-scrollbarW-2, rowH-4))
		if v.focused {
			row.sel.FillColor = colAccent
			row.sel.StrokeColor = colAccent
			row.sel.StrokeWidth = 0
		} else {
			row.sel.FillColor = colSelection
			row.sel.StrokeColor = colSelectionEdge
			row.sel.StrokeWidth = 1
		}
		row.sel.Refresh()
	}

	row.exp.Hide()
	row.exp.Move(fyne.NewPos(4, y+8))
	row.exp.Resize(fyne.NewSize(14, 14))
	if e.isDir {
		row.exp.Show()
	}
	row.icon.Show()
	row.icon.Resource = iconResource(fileIconName(e.name, e.isDir), colText)
	row.icon.Refresh()
	row.icon.Resize(fyne.NewSize(16, 16))
	row.icon.Move(fyne.NewPos(23, y+(rowH-16)/2-1))
	row.name.Show()
	row.size.Show()
	row.date.Show()
	row.name.Alignment = fyne.TextAlignLeading
	row.name.Text = elide(e.name, nameColumnW(size.Width)-60, textSize)
	row.name.Color = colText
	row.name.Move(fyne.NewPos(44, y+1))
	row.name.Resize(fyne.NewSize(nameColumnW(size.Width)-60, 24))
	row.name.Refresh()

	row.size.Text = formatSize(e.size, e.isDir, e.itemCount)
	row.size.Move(fyne.NewPos(nameColumnW(size.Width), y+1))
	row.size.Resize(fyne.NewSize(sizeColumnW-4, 24))
	row.size.Alignment = fyne.TextAlignTrailing
	row.size.Color = colText
	row.size.Refresh()

	row.date.Text = formatDate(e.modTime)
	row.date.Move(fyne.NewPos(nameColumnW(size.Width)+sizeColumnW+5, y+1))
	row.date.Resize(fyne.NewSize(size.Width-nameColumnW(size.Width)-sizeColumnW-scrollbarW-18, 24))
	row.date.Color = colText
	row.date.Refresh()
}

func (r *fileViewRenderer) hideRow(row *fileRow) {
	row.bg.Hide()
	row.sel.Hide()
	row.exp.Hide()
	row.icon.Hide()
	row.name.Hide()
	row.size.Hide()
	row.date.Hide()
}

const (
	scrollbarW  float32 = 24
	sizeColumnW float32 = 109
	dateColumnW float32 = 215
)

func nameColumnW(total float32) float32 {
	return total - scrollbarW - sizeColumnW - dateColumnW
}

func (r *fileViewRenderer) MinSize() fyne.Size { return fyne.NewSize(500, 300) }

func (r *fileViewRenderer) Refresh() {
	if r.v.sortIcon != nil {
		if r.v.ascending {
			r.v.sortIcon.Resource = iconResource("go-up", colTextMuted)
		} else {
			r.v.sortIcon.Resource = iconResource("arrow-down", colTextMuted)
		}
		r.v.sortIcon.Refresh()
	}
	if r.v.focused {
		r.v.scroll.FillColor = colScroll
	} else {
		r.v.scroll.FillColor = colScroll
	}
	r.Layout(r.v.Size())
	canvas.Refresh(r.v)
}

func (r *fileViewRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *fileViewRenderer) Destroy()                     {}

// readEntries lists a directory and gathers the metadata shown by the details
// view. Directory item counts are computed eagerly, like KIO does.
func readEntries(dir string, countChildren bool) ([]fileEntry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]fileEntry, 0, len(des))
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		isLink := info.Mode()&os.ModeSymlink != 0
		target := info
		if isLink {
			if st, err := os.Stat(filepath.Join(dir, de.Name())); err == nil {
				target = st
			}
		}
		e := fileEntry{
			name:    de.Name(),
			path:    filepath.Join(dir, de.Name()),
			isDir:   target.IsDir(),
			isLink:  isLink,
			size:    target.Size(),
			modTime: target.ModTime(),
		}
		if e.isDir && countChildren {
			if children, err := os.ReadDir(e.path); err == nil {
				e.itemCount = len(children)
				if e.itemCount == 0 {
					e.itemCount = 0
				}
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func entryMatches(e fileEntry, patterns []string) bool {
	if e.isDir || len(patterns) == 0 {
		return true
	}
	for _, group := range patterns {
		for _, pat := range strings.FieldsFunc(group, func(r rune) bool { return r == ';' || r == ' ' }) {
			if pat == "*" || pat == "*.*" {
				return true
			}
			if ok, _ := filepath.Match(strings.ToLower(pat), strings.ToLower(e.name)); ok {
				return true
			}
		}
	}
	return false
}
