// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type crumb struct {
	label string
	path  string
	x, w  float32
	tw    float32 // measured text width, cached at path change
}

type breadcrumb struct {
	widget.BaseWidget

	path       string
	home       string
	crumbs     []crumb
	display    []crumb
	hover      int
	onNavigate func(string)
	onDropdown func()
	onEdit     func()

	bg        *canvas.Rectangle
	arrow     *canvas.Image
	seps      []*canvas.Image
	texts     []*canvas.Text
	hoverRect *canvas.Rectangle
}

func newBreadcrumb() *breadcrumb {
	b := &breadcrumb{hover: -1}
	b.home, _ = os.UserHomeDir()
	b.ExtendBaseWidget(b)
	return b
}

func (b *breadcrumb) CreateRenderer() fyne.WidgetRenderer {
	b.bg = newRect(colViewAlternate, colBorder, 1, 5)
	b.arrow = iconImage("go-next", colTextMuted, 12)
	b.hoverRect = newRect(colHover, colTransparent, 0, 3)
	return &breadcrumbRenderer{b: b}
}

func (b *breadcrumb) MinSize() fyne.Size { return fyne.NewSize(200, 37) }

func (b *breadcrumb) SetPath(path string) {
	b.path = path
	b.hover = -1
	b.rebuild()
	b.measure()
	b.Refresh()
}

// measure caches each segment's text width so hover refreshes and relayouts
// never re-measure text (the old per-layout measuring made hovering over a
// long path expensive, especially on Windows).
func (b *breadcrumb) measure() {
	for i := range b.crumbs {
		style := fyne.TextStyle{Bold: i == len(b.crumbs)-1}
		b.crumbs[i].tw = fyne.MeasureText(b.crumbs[i].label, textSize, style).Width
	}
}

func (b *breadcrumb) rebuild() {
	b.crumbs = nil
	path := filepath.Clean(b.path)
	var parts []string
	var prefix []string
	if b.home != "" {
		if r, err := filepath.Rel(b.home, path); err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			if r == "." {
				b.crumbs = []crumb{{label: "Home", path: b.home}}
				return
			}
			parts = strings.Split(r, string(filepath.Separator))
			prefix = []string{b.home}
			b.crumbs = []crumb{{label: "Home", path: b.home}}
			for _, p := range parts {
				if p == "" {
					continue
				}
				prefix = append(prefix, p)
				b.crumbs = append(b.crumbs, crumb{label: p, path: filepath.Join(prefix...)})
			}
			return
		}
	}
	vol := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, vol)
	parts = strings.Split(strings.Trim(rest, string(filepath.Separator)), string(filepath.Separator))
	prefix = []string{vol + string(filepath.Separator)}
	for _, p := range parts {
		if p == "" {
			continue
		}
		prefix = append(prefix, p)
		b.crumbs = append(b.crumbs, crumb{label: p, path: filepath.Join(prefix...)})
	}
	if len(b.crumbs) == 0 {
		b.crumbs = []crumb{{label: "/", path: "/"}}
	}
}

func (b *breadcrumb) Tapped(e *fyne.PointEvent) {
	// Crumbs take precedence over the chevron: the first hit box starts close
	// to the dropdown arrow and must remain clickable.
	for _, c := range b.display {
		if e.Position.X >= c.x && e.Position.X <= c.x+c.w {
			if b.onNavigate != nil {
				b.onNavigate(c.path)
			}
			return
		}
	}
	if e.Position.X < 22 && b.onDropdown != nil {
		b.onDropdown()
		return
	}
	// Like KDE: a click on the location bar that hits no segment switches
	// the bar to an editable path field.
	if b.onEdit != nil {
		b.onEdit()
	}
}

func (b *breadcrumb) MouseIn(*desktop.MouseEvent) {}

func (b *breadcrumb) MouseMoved(e *desktop.MouseEvent) {
	idx := -1
	for i, c := range b.display {
		if e.Position.X >= c.x && e.Position.X <= c.x+c.w {
			idx = i
			break
		}
	}
	if idx != b.hover {
		b.hover = idx
		b.Refresh()
	}
}

func (b *breadcrumb) MouseOut() {
	b.hover = -1
	b.Refresh()
}

type breadcrumbRenderer struct {
	b    *breadcrumb
	objs []fyne.CanvasObject
}

func (r *breadcrumbRenderer) Layout(size fyne.Size) {
	b := r.b
	b.bg.Resize(size)
	b.bg.Move(fyne.NewPos(0, 0))
	b.arrow.Resize(fyne.NewSize(12, 12))
	b.arrow.Move(fyne.NewPos(10, (size.Height-12)/2))

	b.display = b.fitCrumbs(size.Width)
	// Recreate segment objects as the displayed path changes.
	if len(b.texts) != len(b.display) {
		b.texts = make([]*canvas.Text, len(b.display))
		b.seps = make([]*canvas.Image, len(b.display))
		for i := range b.display {
			t := canvas.NewText("", colText)
			t.TextSize = textSize
			b.texts[i] = t
			s := iconImage("go-next", colTextMuted, 10)
			b.seps[i] = s
		}
	}

	x := float32(22)
	for i, c := range b.display {
		t := b.texts[i]
		t.Text = c.label
		style := fyne.TextStyle{Bold: i == len(b.display)-1}
		t.TextStyle = style
		w := c.tw
		if w == 0 {
			w = fyne.MeasureText(c.label, textSize, style).Width
		}
		b.display[i].x = x
		b.display[i].w = w + 8
		t.Move(fyne.NewPos(x, centerY(size.Height, measureHeight())-1))
		t.Resize(fyne.NewSize(w+4, 18))
		t.Color = colText
		t.Refresh()
		x += w + 8
		if i < len(b.display)-1 {
			sep := b.seps[i]
			showObject(sep)
			sep.Resize(fyne.NewSize(10, 10))
			sep.Move(fyne.NewPos(x, (size.Height-10)/2))
			x += 16
		} else {
			hideObject(b.seps[i])
		}
	}
	if b.hover >= 0 && b.hover < len(b.display) {
		c := b.display[b.hover]
		showObject(b.hoverRect)
		b.hoverRect.FillColor = colHover
		b.hoverRect.Resize(fyne.NewSize(c.w, size.Height-10))
		b.hoverRect.Move(fyne.NewPos(c.x-4, 5))
	} else {
		hideObject(b.hoverRect)
	}

	r.objs = []fyne.CanvasObject{b.bg, b.arrow, b.hoverRect}
	for i := range b.texts {
		r.objs = append(r.objs, b.texts[i])
		if i < len(b.seps) {
			r.objs = append(r.objs, b.seps[i])
		}
	}
}

// fitCrumbs returns the visible crumb list, collapsing leading segments into
// an ellipsis when the path would otherwise overflow the navigator.
func (b *breadcrumb) fitCrumbs(width float32) []crumb {
	if len(b.crumbs) == 0 {
		return nil
	}
	const startX, endPad, sepWidth = float32(22), float32(4), float32(16)
	avail := width - startX - endPad
	if avail < 20 {
		avail = 20
	}
	crumbWidth := func(c crumb, bold bool) float32 {
		return fyne.MeasureText(c.label, textSize, fyne.TextStyle{Bold: bold}).Width + 8
	}
	total := func(from int, ellipsis bool) float32 {
		var w float32
		if ellipsis {
			w += fyne.MeasureText("…", textSize, fyne.TextStyle{}).Width + 8 + sepWidth
		}
		for i := from; i < len(b.crumbs); i++ {
			w += crumbWidth(b.crumbs[i], i == len(b.crumbs)-1)
			if i < len(b.crumbs)-1 {
				w += sepWidth
			}
		}
		return w
	}
	start := 0
	for start < len(b.crumbs)-1 && total(start, start > 0) > avail {
		start++
	}
	display := make([]crumb, 0, len(b.crumbs)-start+1)
	if start > 0 {
		display = append(display, crumb{label: "…", path: b.crumbs[start-1].path})
	}
	display = append(display, b.crumbs[start:]...)

	// Even a single trailing crumb can be wider than the navigator; elide it.
	var displayWidth float32
	for i, c := range display {
		displayWidth += crumbWidth(c, i == len(display)-1)
		if i < len(display)-1 {
			displayWidth += sepWidth
		}
	}
	if displayWidth > avail && len(display) > 0 {
		last := display[len(display)-1]
		last.label = elide(last.label, avail-16, textSize)
		display = []crumb{last}
	}
	return display
}

func (r *breadcrumbRenderer) MinSize() fyne.Size { return fyne.NewSize(200, 37) }

func (r *breadcrumbRenderer) Refresh() {
	if r.b.hover >= 0 && r.b.hover < len(r.b.texts) {
		r.b.hoverRect.Show()
	} else {
		r.b.hoverRect.Hide()
	}
	r.Layout(r.b.Size())
	canvas.Refresh(r.b)
}

func (r *breadcrumbRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *breadcrumbRenderer) Destroy()                     {}
