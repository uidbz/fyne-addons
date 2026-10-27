// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"encoding/xml"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type place struct {
	name    string
	path    string
	icon    string
	device  bool
	used    uint64
	size    uint64
	mounted bool
}

type placeGroup struct {
	title  string
	places []place
}

type placesModel struct {
	groups []placeGroup
	flat   []place
}

func newPlacesModel() *placesModel {
	m := &placesModel{}
	m.reload()
	return m
}

func (m *placesModel) reload() {
	bookmarks := loadBookmarks()
	devices := loadDevices()
	if len(bookmarks) == 0 {
		bookmarks = defaultPlaces()
	}

	groups := []placeGroup{}
	if len(bookmarks) > 0 {
		groups = append(groups, placeGroup{title: "Places", places: bookmarks})
	}
	if len(devices) > 0 {
		groups = append(groups, placeGroup{title: "Devices", places: devices})
	}
	if len(groups) == 0 {
		groups = append(groups, placeGroup{title: "Places", places: defaultPlaces()})
	}
	for _, g := range groups {
		m.flat = append(m.flat, g.places...)
	}
	m.groups = groups
}

func defaultPlaces() []place {
	home, _ := os.UserHomeDir()
	out := []place{{name: "Home", path: home, icon: "folder"}}
	for _, name := range []string{"Desktop", "Documents", "Downloads", "Music", "Pictures", "Videos"} {
		p := filepath.Join(home, name)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, place{name: name, path: p, icon: "folder"})
		}
	}
	return out
}

type xbelRoot struct {
	Items []xbelItem `xml:",any"`
}

type xbelItem struct {
	XMLName xml.Name
	Href    string   `xml:"href,attr"`
	Title   string   `xml:"title"`
	Info    xbelInfo `xml:"info"`
}

type xbelInfo struct {
	Metadata []xbelMetadata `xml:"metadata"`
}

type xbelMetadata struct {
	System bool         `xml:"isSystemItem"`
	Hidden string       `xml:"IsHidden"`
	Icon   xbelIconInfo `xml:"icon"`
}

type xbelIconInfo struct {
	Name string `xml:"name,attr"`
}

func loadBookmarks() []place {
	home, _ := os.UserHomeDir()
	xbelPath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "user-places.xbel")
	if os.Getenv("XDG_DATA_HOME") == "" {
		xbelPath = filepath.Join(home, ".local", "share", "user-places.xbel")
	}
	data, err := os.ReadFile(xbelPath)
	if err != nil {
		return nil
	}
	// The on-disk XBEL produced by KDE uses the bookmark: prefix without
	// declaring it. Add the declaration so encoding/xml can parse it.
	text := strings.Replace(string(data), "<xbel>", `<xbel xmlns:bookmark="http://freedesktop.org">`, 1)
	var root xbelRoot
	if err := xml.Unmarshal([]byte(text), &root); err != nil {
		return nil
	}

	seen := map[string]bool{}
	var out []place
	haveHome := false
	for _, item := range root.Items {
		if item.XMLName.Local != "bookmark" {
			continue
		}
		if item.Href == "" || item.Title == "" || hiddenXBEL(item) {
			continue
		}
		u, err := url.Parse(item.Href)
		if err != nil || u.Scheme != "file" {
			continue
		}
		p := u.Path
		if u.Host != "" && u.Host != "localhost" {
			continue
		}
		if p == "" {
			continue
		}
		if seen[p] {
			continue
		}
		icon := "folder"
		for _, md := range item.Info.Metadata {
			if md.Icon.Name != "" {
				icon = md.Icon.Name
			}
		}
		icon = folderIconName(icon)
		seen[p] = true
		if filepath.Clean(p) == filepath.Clean(home) {
			haveHome = true
			out = append(out, place{name: "Home", path: p, icon: icon})
			continue
		}
		out = append(out, place{name: item.Title, path: p, icon: icon})
	}
	if !haveHome && home != "" {
		// Insert Home in the same place KFilePlacesModel does when the
		// XBEL has no explicit entry: after user bookmarks.
		pos := 0
		for i, p := range out {
			if p.name == "Home" {
				pos = i
				break
			}
			pos = i + 1
		}
		out = append(out, place{})
		copy(out[pos+1:], out[pos:])
		out[pos] = place{name: "Home", path: home, icon: "folder"}
	}
	return out
}

func hiddenXBEL(item xbelItem) bool {
	if len(item.Info.Metadata) == 0 {
		return false
	}
	for _, md := range item.Info.Metadata {
		if strings.EqualFold(md.Hidden, "true") {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Places view.

type placesView struct {
	widget.BaseWidget

	model   *placesModel
	current string
	onOpen  func(string)

	hover   int
	focused bool
	offset  float32
	rows    []*placeRow
	headers []*canvas.Text

	bg *canvas.Rectangle
}

type placeRow struct {
	sel    *canvas.Rectangle
	icon   *canvas.Image
	text   *canvas.Text
	barBg  *canvas.Rectangle
	barFg  *canvas.Rectangle
	path   string
	height float32
}

func newPlacesView(model *placesModel) *placesView {
	v := &placesView{model: model, hover: -1}
	v.ExtendBaseWidget(v)
	return v
}

func (v *placesView) CreateRenderer() fyne.WidgetRenderer {
	v.bg = canvas.NewRectangle(colWindow)
	n := 0
	for _, g := range v.model.groups {
		n += len(g.places)
	}
	v.rows = make([]*placeRow, n)
	for i := range v.rows {
		row := &placeRow{}
		row.sel = newRect(colTransparent, colTransparent, 0, 4)
		row.icon = iconImage("folder", colText, 16)
		row.text = canvas.NewText("", colText)
		row.text.TextSize = textSize
		row.barBg = newRect(colorWithAlpha(colTicks, 0x55), colTransparent, 0, 1)
		row.barFg = newRect(colAccent, colTransparent, 0, 1)
		v.rows[i] = row
	}
	v.headers = []*canvas.Text{
		canvas.NewText("", colTextMuted),
		canvas.NewText("", colTextMuted),
	}
	for _, h := range v.headers {
		h.TextSize = 22
	}
	objs := []fyne.CanvasObject{v.bg}
	for _, h := range v.headers {
		objs = append(objs, h)
	}
	for _, row := range v.rows {
		objs = append(objs, row.sel, row.icon, row.text, row.barBg, row.barFg)
	}
	return &placesRenderer{v: v, objs: objs}
}

func (v *placesView) MinSize() fyne.Size { return fyne.NewSize(sidebarWidth, 400) }

func (v *placesView) SetCurrent(path string) {
	v.current = filepath.Clean(path)
	v.Refresh()
}

func (v *placesView) Tapped(e *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(v); c != nil {
		c.Focus(v)
	}
	idx := v.indexAt(e.Position.Y)
	if idx >= 0 {
		if v.onOpen != nil {
			v.onOpen(v.flat()[idx].path)
		}
	}
}

func (v *placesView) flat() []place {
	var out []place
	for _, g := range v.model.groups {
		out = append(out, g.places...)
	}
	return out
}

func (v *placesView) indexAt(y float32) int {
	y += v.offset
	pos := float32(41)
	idx := 0
	for gi, g := range v.model.groups {
		if gi > 0 {
			pos += 35
		}
		for range g.places {
			if y >= pos && y < pos+24 {
				return idx
			}
			pos += 24
			idx++
		}
	}
	return -1
}

func (v *placesView) TypedRune(rune)          {}
func (v *placesView) TypedKey(*fyne.KeyEvent) {}

func (v *placesView) FocusGained() { v.focused = true; v.Refresh() }
func (v *placesView) FocusLost()   { v.focused = false; v.Refresh() }

func (v *placesView) MouseMoved(e *desktop.MouseEvent) {
	idx := v.indexAt(e.Position.Y)
	if idx != v.hover {
		v.hover = idx
		v.Refresh()
	}
}

func (v *placesView) MouseOut() {
	v.hover = -1
	v.Refresh()
}

func (v *placesView) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (v *placesView) Scrolled(ev *fyne.ScrollEvent) {
	v.offset -= ev.Scrolled.DY
	v.clampOffset()
	v.Refresh()
}

func (v *placesView) clampOffset() {
	total := float32(41)
	for gi, g := range v.model.groups {
		if gi > 0 {
			total += 35
		}
		total += float32(len(g.places)) * 24
	}
	max := total - v.Size().Height + 8
	if max < 0 {
		max = 0
	}
	v.offset = clamp(v.offset, 0, max)
}

type placesRenderer struct {
	v    *placesView
	objs []fyne.CanvasObject
}

func (r *placesRenderer) Layout(size fyne.Size) {
	r.v.bg.Resize(size)
	pos := float32(41) - r.v.offset
	idx := 0
	headerIdx := 0
	for gi, g := range r.v.model.groups {
		if gi > 0 {
			pos += 35
		}
		h := r.v.headers[headerIdx]
		h.Text = g.title
		h.Resize(fyne.NewSize(180, 24))
		if gi == 0 {
			h.Move(fyne.NewPos(6, 11))
		} else {
			h.Move(fyne.NewPos(6, pos-31))
		}
		h.Show()
		headerIdx++

		for _, p := range g.places {
			row := r.v.rows[idx]
			row.path = p.path
			row.height = 24
			visible := pos+24 > 0 && pos < size.Height
			if !visible {
				r.hideRow(row)
				pos += 24
				idx++
				continue
			}
			row.show()
			row.icon.Resource = iconResource(p.icon, placeColor(p.icon))
			row.icon.Refresh()
			row.icon.Resize(fyne.NewSize(16, 16))
			row.icon.Move(fyne.NewPos(4, pos+4))
			row.text.Text = elide(p.name, 158, textSize)
			row.text.Move(fyne.NewPos(24, pos-2))
			row.text.Resize(fyne.NewSize(165, 24))
			row.sel.Resize(fyne.NewSize(191, 22))
			selected := filepath.Clean(p.path) == r.v.current
			hovered := idx == r.v.hover
			if selected {
				if r.v.focused {
					row.sel.FillColor = colAccent
				} else {
					row.sel.FillColor = colPlaceSelection
				}
				row.sel.Show()
			} else if hovered {
				row.sel.FillColor = colHover
				row.sel.Show()
			} else {
				row.sel.Hide()
			}
			row.sel.Move(fyne.NewPos(4, pos+1))
			row.sel.Resize(fyne.NewSize(191, 22))

			textW := fyne.MeasureText(row.text.Text, textSize, fyne.TextStyle{}).Width
			if p.device && p.size > 0 {
				barY := pos + 18
				row.barBg.Show()
				row.barBg.Move(fyne.NewPos(25, barY))
				row.barBg.Resize(fyne.NewSize(textW, 2))
				row.barFg.Show()
				frac := float32(p.used) / float32(p.size)
				row.barFg.Move(fyne.NewPos(25, barY))
				row.barFg.Resize(fyne.NewSize(textW*clamp(frac, 0, 1), 2))
				if selected {
					row.barFg.FillColor = colText
				} else {
					row.barFg.FillColor = colAccent
				}
			} else {
				row.barBg.Hide()
				row.barFg.Hide()
			}
			pos += 24
			idx++
		}
	}
	for ; headerIdx < len(r.v.headers); headerIdx++ {
		r.v.headers[headerIdx].Hide()
	}
	r.objs = r.objs[:0]
	r.objs = append(r.objs, r.v.bg)
	for _, h := range r.v.headers {
		r.objs = append(r.objs, h)
	}
	for _, row := range r.v.rows {
		r.objs = append(r.objs, row.sel, row.icon, row.text, row.barBg, row.barFg)
	}
}

func (r *placesRenderer) hideRow(row *placeRow) {
	row.sel.Hide()
	row.icon.Hide()
	row.text.Hide()
	row.barBg.Hide()
	row.barFg.Hide()
}

func (row *placeRow) show() {
	row.sel.Show()
	row.icon.Show()
	row.text.Show()
	row.barBg.Hide()
	row.barFg.Hide()
}

func (r *placesRenderer) MinSize() fyne.Size { return fyne.NewSize(sidebarWidth, 400) }

func (r *placesRenderer) Refresh() {
	canvas.Refresh(r.v)
	r.Layout(r.v.Size())
}

func (r *placesRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *placesRenderer) Destroy()                     {}

func elide(s string, max float32, size float32) string {
	if fyne.MeasureText(s, size, fyne.TextStyle{}).Width <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		cand := string(r) + "…"
		if fyne.MeasureText(cand, size, fyne.TextStyle{}).Width <= max {
			return cand
		}
	}
	return "…"
}

func colorWithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}
