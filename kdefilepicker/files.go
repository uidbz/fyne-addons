// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

type elideCache struct {
	mode byte    // view the width belongs to (details/compact/icons)
	w    float32 // column width the elided text was computed for
	out  string
}

type fileEntry struct {
	name      string
	path      string
	isDir     bool
	size      int64
	modTime   time.Time
	itemCount int
	depth     int
	el        elideCache // cached elide(name) keyed by view+width
}

type fileView struct {
	widget.BaseWidget

	entries      []fileEntry
	filtered     []fileEntry
	selected     int
	hover        int
	offset       float32
	focused      bool
	showHidden   bool
	sortField    sortField
	sizeColW     float32 // user-set details column widths; 0 = automatic
	dateColW     float32
	colHandle1   *colHandle
	colHandle2   *colHandle
	ascending    bool
	foldersFirst bool
	filterText   string
	patterns     []string
	expanded     map[string]bool
	childCache   map[string][]fileEntry
	singleClick  bool
	scrollDrag   bool
	scrollGrab   float32

	viewMode       int // 0 details, 1 compact, 2 icons
	iconSize       float32
	detailIconSize float32
	onSelect       func(fileEntry)
	onOpenDir      func(string)
	onActivate     func(fileEntry)
	onEscape       func()

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

	// Cached inputs so layout can skip expensive icon lookups, text shaping
	// and repaints when a row is only being re-laid out for hover/selection.
	iconName string
	expName  string
}

func newFileView() *fileView {
	v := &fileView{
		selected:       -1,
		hover:          -1,
		ascending:      true,
		foldersFirst:   true,
		iconSize:       16,
		detailIconSize: 16,
		expanded:       map[string]bool{},
		childCache:     map[string][]fileEntry{},
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
	v.colHandle1 = newColHandle(v, 0)
	v.colHandle2 = newColHandle(v, 1)
	v.scrollBg = canvas.NewRectangle(colTransparent)
	v.scroll = newRect(colScroll, colTransparent, 0, 3.5)
	r := &fileViewRenderer{v: v}
	r.Refresh()
	return r
}

func (v *fileView) MinSize() fyne.Size { return fyne.NewSize(500, 300) }

// szW/dtW return the effective size/date column widths (user dragged them
// via the header separators, otherwise the defaults apply). The name column
// always absorbs the remainder.
func (v *fileView) szW() float32 {
	if v.sizeColW > 0 {
		return v.sizeColW
	}
	return sizeColumnW
}

func (v *fileView) dtW() float32 {
	if v.dateColW > 0 {
		return v.dateColW
	}
	return dateColumnW
}

func (v *fileView) nameEnd(total float32) float32 {
	ne := total - scrollbarW - v.szW() - v.dtW()
	if ne < 120 {
		ne = 120
	}
	return ne
}

// elidedName returns e.name shortened for width w, cached on the entry so
// refreshes (selection, hover, scroll) do not re-measure text. The cache is
// keyed by view mode and column width.
func (v *fileView) elidedName(idx int, w float32, mode byte) string {
	if idx < 0 || idx >= len(v.filtered) {
		return ""
	}
	e := &v.filtered[idx]
	if q := float32(int(w/4)) * 4; q > 0 {
		w = q // quantise so column drags reuse the cache instead of re-measuring every pixel
	}
	if e.el.mode == mode && e.el.w == w && e.el.out != "" {
		return e.el.out
	}
	e.el.mode, e.el.w = mode, w
	e.el.out = elide(e.name, w, textSize)
	return e.el.out
}

// colHandle is the thin drag target over a details-view column separator.
// Dragging resizes the column to its right; double-click resets it.
type colHandle struct {
	widget.BaseWidget

	view   *fileView
	which  int // 0: name|size separator, 1: size|date separator
	armed  bool
	acc    float32
	baseSz float32
	baseDt float32
}

func newColHandle(v *fileView, which int) *colHandle {
	h := &colHandle{view: v, which: which}
	h.ExtendBaseWidget(h)
	return h
}

func (h *colHandle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(newRect(colTransparent, colTransparent, 0, 0))
}

func (h *colHandle) MinSize() fyne.Size { return fyne.NewSize(9, headerH) }

func (h *colHandle) DragStart(*fyne.DragEvent) { h.arm() }

// arm captures the column widths the drag started from. The desktop driver
// never calls DragStart, so Dragged arms lazily; accumulating per-event
// deltas (instead of tracking absolute positions) also keeps the drag
// stable while the handle itself moves with the separator.
func (h *colHandle) arm() {
	if h.armed {
		return
	}
	h.acc, h.baseSz, h.baseDt = 0, h.view.szW(), h.view.dtW()
	h.armed = true
}

func (h *colHandle) Dragged(ev *fyne.DragEvent) {
	h.arm()
	h.acc += ev.Dragged.DX
	switch h.which {
	case 0:
		h.view.sizeColW = clamp(h.baseSz-h.acc, 50, 600)
	case 1:
		h.view.dateColW = clamp(h.baseDt-h.acc, 90, 600)
	}
	h.view.Refresh()
}

func (h *colHandle) DragEnd() { h.armed, h.acc = false, 0 }

func (h *colHandle) DoubleTapped(*fyne.PointEvent) {
	if h.which == 0 {
		h.view.sizeColW = 0
	} else {
		h.view.dateColW = 0
	}
	h.view.Refresh()
}

func (v *fileView) SetDir(entries []fileEntry) {
	v.entries = entries
	v.selected = -1
	v.hover = -1
	v.offset = 0
	v.expanded = map[string]bool{}
	v.childCache = map[string][]fileEntry{}
	v.applyFilter()
}

func (v *fileView) SetShowHidden(show bool) {
	v.showHidden = show
	v.applyFilter()
}

func (v *fileView) SetPatterns(patterns []string) {
	v.patterns = compilePatterns(patterns)
	v.applyFilter()
}

// compilePatterns flattens the Filter.Patterns spelling (single globs or
// space/semicolon separated groups) once instead of on every entry.
func compilePatterns(groups []string) []string {
	var out []string
	for _, group := range groups {
		for _, pattern := range strings.FieldsFunc(group, func(r rune) bool { return r == ';' || r == ' ' }) {
			out = append(out, strings.ToLower(pattern))
		}
	}
	return out
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
	selectedPath := ""
	if entry, ok := v.selectedEntry(); ok {
		selectedPath = entry.path
	}
	base := v.filterSort(v.entries)
	out := make([]fileEntry, 0, len(base))
	var add func(list []fileEntry, depth int)
	add = func(list []fileEntry, depth int) {
		for _, e := range list {
			e.depth = depth
			out = append(out, e)
			if v.viewMode == 0 && e.isDir && v.expanded[e.path] {
				children := v.filterSort(v.children(e.path))
				add(children, depth+1)
			}
		}
	}
	add(base, 0)
	v.filtered = out
	v.selected = -1
	if selectedPath != "" {
		v.selectPath(selectedPath)
	}
	v.clampOffset()
	v.Refresh()
}

func (v *fileView) filterSort(entries []fileEntry) []fileEntry {
	filtered := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
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
		cmp := compareEntries(a, b, v.sortField)
		if v.ascending {
			return cmp < 0
		}
		return cmp > 0
	})
	return filtered
}

// compareEntries provides a total order for the supported sort fields. Size,
// date and type ties fall back to the name so that equal keys keep a
// deterministic order in both directions.
func compareEntries(a, b fileEntry, field sortField) int {
	switch field {
	case sortBySize:
		if a.size != b.size {
			if a.size < b.size {
				return -1
			}
			return 1
		}
	case sortByDate:
		if !a.modTime.Equal(b.modTime) {
			if a.modTime.Before(b.modTime) {
				return -1
			}
			return 1
		}
	case sortByType:
		ea := strings.ToLower(filepath.Ext(a.name))
		eb := strings.ToLower(filepath.Ext(b.name))
		if ea != eb {
			if ea < eb {
				return -1
			}
			return 1
		}
	}
	an := strings.ToLower(a.name)
	bn := strings.ToLower(b.name)
	switch {
	case an < bn:
		return -1
	case an > bn:
		return 1
	}
	return 0
}

func (v *fileView) children(path string) []fileEntry {
	if cached, ok := v.childCache[path]; ok {
		return cached
	}
	entries, err := readEntries(path, true)
	if err != nil {
		entries = nil
	}
	v.childCache[path] = entries
	return entries
}

func (v *fileView) toggleExpanded(e fileEntry) {
	if !e.isDir || v.viewMode != 0 {
		return
	}
	if v.expanded[e.path] {
		delete(v.expanded, e.path)
	} else {
		v.expanded[e.path] = true
	}
	v.applyFilter()
}

func (v *fileView) selectPath(path string) {
	if path == "" {
		return
	}
	for i, e := range v.filtered {
		if e.path == path {
			v.selected = i
			return
		}
	}
	v.selected = -1
}

func (v *fileView) SetViewMode(mode int) {
	v.viewMode = mode
	v.applyFilter()
}

func (v *fileView) entryAtPoint(x, y float32) int {
	if v.viewMode == 2 {
		cols, cellW, cellH := v.grid(v.Size())
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
	if v.viewMode == 0 {
		s := clamp(v.detailIconSize, 16, 64)
		rh := s + 15
		if rh < rowH {
			rh = rowH
		}
		return rh
	}
	if v.viewMode == 1 {
		s := clamp(v.detailIconSize, 16, 48)
		rh := s + 10
		if rh < 26 {
			rh = 26
		}
		return rh
	}
	return rowH
}

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
	if v.inScrollbar(e.Position.X, e.Position.Y) {
		v.scrollToY(e.Position.Y)
		return
	}
	// Tapping a details-view header sorts by that column; the drag handles
	// sit on the separators and swallow events there instead.
	if v.viewMode == 0 && e.Position.Y >= 0 && e.Position.Y <= headerH {
		ne := v.nameEnd(v.Size().Width)
		var field sortField
		switch {
		case e.Position.X < ne-4:
			field = sortByName
		case e.Position.X < ne+v.szW()-4:
			field = sortBySize
		default:
			field = sortByDate
		}
		asc := true
		if v.sortField == field {
			asc = !v.ascending
		}
		v.SetSort(field, asc)
		return
	}
	idx := v.entryAtPoint(e.Position.X, e.Position.Y)
	if idx < 0 {
		return
	}
	entry := v.filtered[idx]
	if v.viewMode == 0 && entry.isDir && e.Position.X >= float32(4+entry.depth*16) && e.Position.X <= float32(22+entry.depth*16) {
		v.toggleExpanded(entry)
		return
	}
	v.selected = idx
	if v.onSelect != nil {
		v.onSelect(entry)
	}
	if v.singleClick && entry.isDir && v.onOpenDir != nil {
		v.onOpenDir(entry.path)
		return
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

func (v *fileView) MouseIn(*desktop.MouseEvent) {}

func (v *fileView) MouseMoved(e *desktop.MouseEvent) {
	if v.scrollDrag {
		v.scrollToY(e.Position.Y)
		return
	}
	idx := v.entryAtPoint(e.Position.X, e.Position.Y)
	if idx != v.hover {
		v.hover = idx
		v.Refresh()
	}
}

func (v *fileView) MouseOut() {
	v.hover = -1
	v.Refresh()
}

func (v *fileView) inScrollbar(x, y float32) bool {
	if v.maxOffset() <= 0 {
		return false
	}
	top := float32(0)
	if v.viewMode == 0 {
		top = headerH
	}
	return x >= v.Size().Width-scrollbarW && y >= top
}

func (v *fileView) scrollbarTrack() (top, height, thumb float32) {
	top = 0
	if v.viewMode == 0 {
		top = headerH
	}
	height = v.Size().Height - top - 8
	if height < 1 {
		height = 1
	}
	content := v.maxOffset() + height
	thumb = height * height / content
	if thumb < 24 {
		thumb = 24
	}
	return
}

func (v *fileView) scrollToY(y float32) {
	top, height, thumb := v.scrollbarTrack()
	track := height - thumb
	if track <= 0 {
		return
	}
	frac := (y - top - v.scrollGrab) / track
	v.offset = clamp(frac, 0, 1) * v.maxOffset()
	v.clampOffset()
	v.Refresh()
}

func (v *fileView) scrollThumbTop() float32 {
	top, height, thumb := v.scrollbarTrack()
	track := height - thumb
	if track <= 0 {
		return top
	}
	return top + (v.offset/v.maxOffset())*track
}

func (v *fileView) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary || !v.inScrollbar(e.Position.X, e.Position.Y) {
		return
	}
	_, _, thumb := v.scrollbarTrack()
	thumbTop := v.scrollThumbTop()
	v.scrollGrab = thumb / 2
	if e.Position.Y >= thumbTop && e.Position.Y <= thumbTop+thumb {
		v.scrollGrab = e.Position.Y - thumbTop
	}
	v.scrollDrag = true
	v.scrollToY(e.Position.Y)
}

func (v *fileView) MouseUp(*desktop.MouseEvent) {
	v.scrollDrag = false
}

func (v *fileView) Dragged(e *fyne.DragEvent) {
	if v.scrollDrag {
		v.scrollToY(e.Position.Y)
	}
}

func (v *fileView) DragEnd() {
	v.scrollDrag = false
	v.Refresh()
}

func (v *fileView) TypedRune(rune) {}

func (v *fileView) FocusGained() { v.focused = true; v.Refresh() }
func (v *fileView) FocusLost()   { v.focused = false; v.Refresh() }

func (v *fileView) TypedKey(ev *fyne.KeyEvent) {
	selectedChanged := false
	offsetChanged := false
	switch ev.Name {
	case fyne.KeyDown:
		if v.selected < len(v.filtered)-1 {
			v.selected++
			v.scrollToSelected()
			selectedChanged, offsetChanged = true, true
		}
	case fyne.KeyUp:
		if v.selected > 0 {
			v.selected--
			v.scrollToSelected()
			selectedChanged, offsetChanged = true, true
		}
	case fyne.KeyHome:
		if len(v.filtered) > 0 {
			v.selected = 0
			v.scrollToSelected()
			selectedChanged, offsetChanged = true, true
		}
	case fyne.KeyEnd:
		if len(v.filtered) > 0 {
			v.selected = len(v.filtered) - 1
			v.scrollToSelected()
			selectedChanged, offsetChanged = true, true
		}
	case fyne.KeyPageDown:
		v.offset += float32(v.visibleRows()) * v.rowHeight()
		v.clampOffset()
		offsetChanged = true
	case fyne.KeyPageUp:
		v.offset -= float32(v.visibleRows()) * v.rowHeight()
		v.clampOffset()
		offsetChanged = true
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
	if selectedChanged && v.selected >= 0 && v.selected < len(v.filtered) && v.onSelect != nil {
		v.onSelect(v.filtered[v.selected])
	}
	if selectedChanged || offsetChanged {
		v.Refresh()
	}
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
	v.offset = clamp(v.offset, 0, v.maxOffset())
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

// ---------------------------------------------------------------------------
// Renderer.

type fileViewRenderer struct {
	v    *fileView
	objs []fyne.CanvasObject
}

func (r *fileViewRenderer) rowObjects() []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, len(r.v.rows)*7)
	for _, row := range r.v.rows {
		// Spare/hidden rows stay allocated for reuse but are not handed to
		// the painter every frame.
		if !row.bg.Visible() {
			continue
		}
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

func showObject(o fyne.CanvasObject) {
	if !o.Visible() {
		o.Show()
	}
}

func hideObject(o fyne.CanvasObject) {
	if o.Visible() {
		o.Hide()
	}
}

func setFill(r *canvas.Rectangle, clr color.Color) {
	if r.FillColor != clr {
		r.FillColor = clr
		r.Refresh()
	}
}

func setRowText(t *canvas.Text, text string, align fyne.TextAlign) {
	if t.Text != text {
		t.Text = text
		t.Refresh()
	}
	if t.Alignment != align {
		t.Alignment = align
		t.Refresh()
	}
}

func setRowIcon(row *fileRow, name string) {
	if row.iconName == name {
		return
	}
	row.iconName = name
	row.icon.Resource = iconResource(name, colText)
	row.icon.Refresh()
}

func setRowExpandedIcon(row *fileRow, name string) {
	if row.expName == name {
		return
	}
	row.expName = name
	row.exp.Resource = iconResource(name, colTextMuted)
	row.exp.Refresh()
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
	needed := 0
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
		needed = count
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
		needed = count
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
	// Do not keep a stale, oversized row pool after a view/zoom change that
	// needs fewer cells; a small spare avoids re-allocating rows while
	// scrolling.
	maxKeep := needed + 8
	if maxKeep < 16 {
		maxKeep = 16
	}
	if len(v.rows) > maxKeep {
		v.rows = v.rows[:maxKeep]
	}

	if v.maxOffset() <= 0 {
		hideObject(v.scrollBg)
		hideObject(v.scroll)
	} else {
		showObject(v.scrollBg)
		v.scrollBg.Resize(fyne.NewSize(scrollbarW, visibleH))
		v.scrollBg.Move(fyne.NewPos(size.Width-scrollbarW, header))
		v.scrollBg.FillColor = colView
		showObject(v.scroll)
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
			v.colHandle1, v.colHandle2,
		)
	}
	r.objs = append(r.objs, v.scrollBg, v.scroll)
}

func (r *fileViewRenderer) layoutHeader(size fyne.Size) {
	v := r.v
	for _, o := range []fyne.CanvasObject{v.hdrBg, v.hdrTop, v.hdrDiv1, v.hdrDiv2, v.hdrName, v.hdrSize, v.hdrDate, v.sortIcon, v.colHandle1, v.colHandle2} {
		showObject(o)
	}
	headerW := size.Width - scrollbarW
	ne := v.nameEnd(size.Width)
	sz := v.szW()
	v.hdrBg.Resize(fyne.NewSize(headerW, headerH))
	v.hdrBg.Move(fyne.NewPos(0, 0))
	v.hdrTop.Resize(fyne.NewSize(headerW, 1))
	v.hdrTop.Move(fyne.NewPos(0, 0))
	v.hdrDiv1.Resize(fyne.NewSize(1, headerH))
	v.hdrDiv1.Move(fyne.NewPos(ne, 0))
	v.hdrDiv2.Resize(fyne.NewSize(1, headerH))
	v.hdrDiv2.Move(fyne.NewPos(ne+sz, 0))
	hdrY := centerY(headerH, measureHeight())
	v.hdrName.Move(fyne.NewPos(3, hdrY+1))
	v.hdrSize.Move(fyne.NewPos(ne+6, hdrY+1))
	v.hdrDate.Move(fyne.NewPos(ne+sz+7, hdrY+1))
	v.hdrName.Resize(fyne.NewSize(ne-40, 18))
	v.hdrSize.Resize(fyne.NewSize(sz, 18))
	v.hdrDate.Resize(fyne.NewSize(100, 18))
	v.sortIcon.Resize(fyne.NewSize(12, 12))
	v.sortIcon.Move(fyne.NewPos(ne-22, (headerH-12)/2))
	v.colHandle1.Resize(fyne.NewSize(9, headerH))
	v.colHandle1.Move(fyne.NewPos(ne-4, 0))
	v.colHandle2.Resize(fyne.NewSize(9, headerH))
	v.colHandle2.Move(fyne.NewPos(ne+sz-4, 0))
}

func (r *fileViewRenderer) hideHeader() {
	hideObject(r.v.hdrBg)
	hideObject(r.v.hdrTop)
	hideObject(r.v.hdrDiv1)
	hideObject(r.v.hdrDiv2)
	hideObject(r.v.hdrName)
	hideObject(r.v.hdrSize)
	hideObject(r.v.hdrDate)
	hideObject(r.v.sortIcon)
	hideObject(r.v.colHandle1)
	hideObject(r.v.colHandle2)
}

func (r *fileViewRenderer) layoutCompactRow(row *fileRow, idx int, e fileEntry, y float32, size fyne.Size, visibleH, header float32) {
	v := r.v
	rh := v.rowHeight()
	if y+rh < header || y > header+visibleH {
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
	setFill(row.bg, rowBgColor)
	showObject(row.bg)
	row.bg.Resize(fyne.NewSize(size.Width-scrollbarW, rh))
	row.bg.Move(fyne.NewPos(0, y))
	hideObject(row.sel)
	if idx == v.selected {
		showObject(row.sel)
		row.sel.Move(fyne.NewPos(1, y+2))
		row.sel.Resize(fyne.NewSize(size.Width-scrollbarW-2, rh-4))
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
	hideObject(row.exp)
	iconSize := clamp(v.detailIconSize, 16, 48)
	showObject(row.icon)
	setRowIcon(row, fileIconName(e.name, e.isDir))
	row.icon.Resize(fyne.NewSize(iconSize, iconSize))
	row.icon.Move(fyne.NewPos(8, y+(rh-iconSize)/2))
	showObject(row.name)
	nameX := 8 + iconSize + 6
	nameW := size.Width - scrollbarW - 140 - nameX - 8
	if nameW < 10 {
		nameW = 10
	}
	row.name.Text = v.elidedName(idx, nameW, 1)
	row.name.Alignment = fyne.TextAlignLeading
	row.name.Color = colText
	row.name.Move(fyne.NewPos(nameX, y+(rh-26)/2-2))
	row.name.Resize(fyne.NewSize(nameW, 24))
	showObject(row.size)
	setRowText(row.size, formatSize(e.size, e.isDir, e.itemCount), fyne.TextAlignTrailing)
	row.size.Color = colText
	row.size.Move(fyne.NewPos(size.Width-scrollbarW-140, y+(rh-26)/2-2))
	row.size.Resize(fyne.NewSize(130, 24))
	hideObject(row.date)
}

func (r *fileViewRenderer) layoutIconRow(row *fileRow, idx int, e fileEntry, x, y, cellW, cellH, visibleH, header float32) {
	v := r.v
	iconSize := v.iconSize
	if iconSize < 16 {
		iconSize = 64
	}
	setFill(row.bg, colView)
	showObject(row.bg)
	row.bg.Resize(fyne.NewSize(cellW, cellH))
	row.bg.Move(fyne.NewPos(x, y))

	switch {
	case idx == v.selected && v.focused:
		showObject(row.sel)
		row.sel.FillColor = colAccent
		row.sel.StrokeColor = colAccent
		row.sel.StrokeWidth = 0
		row.sel.Move(fyne.NewPos(x+4, y+2))
		row.sel.Resize(fyne.NewSize(cellW-8, cellH-4))
		row.sel.Refresh()
	case idx == v.selected:
		showObject(row.sel)
		row.sel.FillColor = colSelection
		row.sel.StrokeColor = colSelectionEdge
		row.sel.StrokeWidth = 1
		row.sel.Move(fyne.NewPos(x+4, y+2))
		row.sel.Resize(fyne.NewSize(cellW-8, cellH-4))
		row.sel.Refresh()
	case idx == v.hover:
		showObject(row.sel)
		row.sel.FillColor = colHover
		row.sel.StrokeColor = colTransparent
		row.sel.StrokeWidth = 0
		row.sel.Move(fyne.NewPos(x+4, y+2))
		row.sel.Resize(fyne.NewSize(cellW-8, cellH-4))
		row.sel.Refresh()
	default:
		hideObject(row.sel)
	}

	hideObject(row.exp)
	showObject(row.icon)
	setRowIcon(row, fileIconName(e.name, e.isDir))
	row.icon.Resize(fyne.NewSize(iconSize, iconSize))
	row.icon.Move(fyne.NewPos(x+(cellW-iconSize)/2, y+10))

	showObject(row.name)
	row.name.Text = v.elidedName(idx, cellW-8, 2)
	row.name.Alignment = fyne.TextAlignCenter
	row.name.Color = colText
	row.name.Move(fyne.NewPos(x+4, y+iconSize+14))
	row.name.Resize(fyne.NewSize(cellW-8, 24))
	hideObject(row.size)
	hideObject(row.date)
}

func (r *fileViewRenderer) layoutRow(row *fileRow, idx int, e fileEntry, y float32, size fyne.Size, visibleH float32) {
	v := r.v
	rh := v.rowHeight()
	if y+rh < headerH || y > headerH+visibleH {
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
	setFill(row.bg, rowBgColor)
	showObject(row.bg)
	row.bg.Resize(fyne.NewSize(size.Width-scrollbarW, rh))
	row.bg.Move(fyne.NewPos(0, y))

	hideObject(row.sel)
	if idx == v.selected {
		showObject(row.sel)
		row.sel.Move(fyne.NewPos(1, y+2))
		row.sel.Resize(fyne.NewSize(size.Width-scrollbarW-2, rh-4))
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

	indent := float32(e.depth * 16)
	hideObject(row.exp)
	row.exp.Move(fyne.NewPos(4+indent, y+(rh-14)/2))
	row.exp.Resize(fyne.NewSize(14, 14))
	if e.isDir {
		if v.expanded[e.path] {
			setRowExpandedIcon(row, "go-down")
		} else {
			setRowExpandedIcon(row, "go-next")
		}
		showObject(row.exp)
	}

	iconSize := clamp(v.detailIconSize, 16, 64)
	showObject(row.icon)
	setRowIcon(row, fileIconName(e.name, e.isDir))
	row.icon.Resize(fyne.NewSize(iconSize, iconSize))
	row.icon.Move(fyne.NewPos(23+indent, y+(rh-iconSize)/2-1))

	showObject(row.name)
	showObject(row.size)
	showObject(row.date)
	nameX := 23 + indent + iconSize + 5
	nameW := v.nameEnd(size.Width) - nameX - 16
	if nameW < 10 {
		nameW = 10
	}
	row.name.Text = v.elidedName(idx, nameW, 0)
	row.name.Alignment = fyne.TextAlignLeading
	row.name.Color = colText
	textY := y + (rh-rowH)/2 + 1
	row.name.Move(fyne.NewPos(nameX, textY))
	row.name.Resize(fyne.NewSize(nameW, 24))

	setRowText(row.size, formatSize(e.size, e.isDir, e.itemCount), fyne.TextAlignTrailing)
	row.size.Move(fyne.NewPos(v.nameEnd(size.Width), textY))
	row.size.Resize(fyne.NewSize(v.szW()-4, 24))
	row.size.Color = colText

	setRowText(row.date, formatDate(e.modTime), fyne.TextAlignLeading)
	row.date.Move(fyne.NewPos(v.nameEnd(size.Width)+v.szW()+5, textY))
	row.date.Resize(fyne.NewSize(size.Width-v.nameEnd(size.Width)-v.szW()-scrollbarW-18, 24))
	row.date.Color = colText
}

func (r *fileViewRenderer) hideRow(row *fileRow) {
	hideObject(row.bg)
	hideObject(row.sel)
	hideObject(row.exp)
	hideObject(row.icon)
	hideObject(row.name)
	hideObject(row.size)
	hideObject(row.date)
}

const (
	scrollbarW  float32 = 24
	sizeColumnW float32 = 109
	dateColumnW float32 = 215
)

func (r *fileViewRenderer) MinSize() fyne.Size { return fyne.NewSize(500, 300) }

func (r *fileViewRenderer) Refresh() {
	if r.v.sortIcon != nil {
		if r.v.ascending {
			setIconResource(r.v.sortIcon, "go-up", colTextMuted)
		} else {
			setIconResource(r.v.sortIcon, "arrow-down", colTextMuted)
		}
	}
	setFill(r.v.scroll, colScroll)
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
		out = append(out, fileEntry{
			name:    de.Name(),
			path:    filepath.Join(dir, de.Name()),
			isDir:   target.IsDir(),
			size:    target.Size(),
			modTime: target.ModTime(),
		})
	}
	if countChildren {
		countDirectoryEntries(out)
	}
	return out, nil
}

// countDirectoryEntries fills in the KIO-style "N items" counts. Reading every
// subdirectory is I/O bound, so use a small worker pool instead of walking the
// list serially on the UI goroutine.
func countDirectoryEntries(entries []fileEntry) {
	dirs := make([]int, 0, len(entries))
	for i := range entries {
		if entries[i].isDir {
			dirs = append(dirs, i)
		}
	}
	if len(dirs) == 0 {
		return
	}
	if len(dirs) < 4 {
		for _, i := range dirs {
			if children, err := os.ReadDir(entries[i].path); err == nil {
				entries[i].itemCount = len(children)
			}
		}
		return
	}
	workers := len(dirs)
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for index := range jobs {
				if children, err := os.ReadDir(entries[index].path); err == nil {
					entries[index].itemCount = len(children)
				}
			}
		}()
	}
	for _, index := range dirs {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
}

func entryMatches(e fileEntry, patterns []string) bool {
	if e.isDir || len(patterns) == 0 {
		return true
	}
	name := strings.ToLower(e.name)
	for _, pattern := range patterns {
		if pattern == "*" || pattern == "*.*" {
			return true
		}
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}
