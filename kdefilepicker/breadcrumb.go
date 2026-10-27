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
}

type breadcrumb struct {
	widget.BaseWidget

	path       string
	home       string
	crumbs     []crumb
	hover      int
	onNavigate func(string)
	onDropdown func()

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
	b.rebuild()
	b.Refresh()
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
	if e.Position.X < 32 {
		if b.onDropdown != nil {
			b.onDropdown()
		}
		return
	}
	for i, c := range b.crumbs {
		if e.Position.X >= c.x && e.Position.X <= c.x+c.w {
			if b.onNavigate != nil {
				b.onNavigate(c.path)
			}
			_ = i
			return
		}
	}
}

func (b *breadcrumb) MouseIn(*desktop.MouseEvent) {}

func (b *breadcrumb) MouseMoved(e *desktop.MouseEvent) {
	idx := -1
	for i, c := range b.crumbs {
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

func (b *breadcrumb) Cursor() desktop.Cursor { return desktop.PointerCursor }

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

	// Recreate segment objects as the path changes.
	if len(b.texts) != len(b.crumbs) {
		b.texts = make([]*canvas.Text, len(b.crumbs))
		b.seps = make([]*canvas.Image, len(b.crumbs))
		for i := range b.crumbs {
			t := canvas.NewText("", colText)
			t.TextSize = textSize
			b.texts[i] = t
			s := iconImage("go-next", colTextMuted, 10)
			b.seps[i] = s
		}
	}

	x := float32(22)
	for i, c := range b.crumbs {
		t := b.texts[i]
		t.Text = c.label
		style := fyne.TextStyle{Bold: i == len(b.crumbs)-1}
		t.TextStyle = style
		w := fyne.MeasureText(c.label, textSize, style).Width
		b.crumbs[i].x = x - 4
		b.crumbs[i].w = w + 8
		t.Move(fyne.NewPos(x, centerY(size.Height, measureHeight())-1))
		t.Resize(fyne.NewSize(w+4, 18))
		t.Color = colText
		t.Refresh()
		x += w + 8
		if i < len(b.crumbs)-1 {
			sep := b.seps[i]
			sep.Show()
			sep.Resize(fyne.NewSize(10, 10))
			sep.Move(fyne.NewPos(x, (size.Height-10)/2))
			x += 16
		} else {
			b.seps[i].Hide()
		}
	}
	if b.hover >= 0 && b.hover < len(b.crumbs) {
		c := b.crumbs[b.hover]
		b.hoverRect.Show()
		b.hoverRect.FillColor = colHover
		b.hoverRect.Resize(fyne.NewSize(c.w, size.Height-10))
		b.hoverRect.Move(fyne.NewPos(c.x, 5))
	} else {
		b.hoverRect.Hide()
	}

	r.objs = []fyne.CanvasObject{b.bg, b.arrow, b.hoverRect}
	for i := range b.texts {
		r.objs = append(r.objs, b.texts[i])
		if i < len(b.seps) {
			r.objs = append(r.objs, b.seps[i])
		}
	}
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
