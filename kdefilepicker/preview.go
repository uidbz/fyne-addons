// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type previewPane struct {
	widget.BaseWidget

	path  string
	entry fileEntry

	bg       *canvas.Rectangle
	headerBg *canvas.Rectangle
	line     *canvas.Rectangle
	header   *canvas.Text
	icon     *canvas.Image
	img      *canvas.Image
	name     *canvas.Text
	kind     *canvas.Text
	size     *canvas.Text
	date     *canvas.Text
}

func newPreviewPane() *previewPane {
	p := &previewPane{}
	p.bg = canvas.NewRectangle(colWindow)
	p.headerBg = canvas.NewRectangle(colHeader)
	p.line = canvas.NewRectangle(colBorder)
	p.header = canvas.NewText("Preview", colText)
	p.header.TextSize = textSize
	p.icon = iconImage("text-x-generic", colTextMuted, 64)
	p.img = iconImage("text-x-generic", colTextMuted, 64)
	p.img.FillMode = canvas.ImageFillContain
	p.img.Hide()
	p.name = canvas.NewText("No selection", colText)
	p.name.TextSize = textSize
	p.name.TextStyle = fyne.TextStyle{Bold: true}
	p.kind = canvas.NewText("", colTextMuted)
	p.kind.TextSize = textSize
	p.size = canvas.NewText("", colTextMuted)
	p.size.TextSize = textSize
	p.date = canvas.NewText("", colTextMuted)
	p.date.TextSize = textSize
	p.ExtendBaseWidget(p)
	return p
}

func (p *previewPane) CreateRenderer() fyne.WidgetRenderer {
	r := &previewRenderer{p: p}
	r.Refresh()
	return r
}

func (p *previewPane) MinSize() fyne.Size { return fyne.NewSize(280, 200) }

func (p *previewPane) Clear() {
	if p.img == nil || p.icon == nil || p.name == nil {
		p.path = ""
		p.entry = fileEntry{}
		return
	}
	p.path = ""
	p.entry = fileEntry{}
	p.img.Hide()
	p.icon.Resource = iconResource("text-x-generic", colTextMuted)
	p.icon.Show()
	p.name.Text = "No selection"
	p.kind.Text = ""
	p.size.Text = ""
	p.date.Text = ""
	canvas.Refresh(p)
}

func (p *previewPane) SetFile(path string, e fileEntry) {
	p.path = path
	p.entry = e
	p.name.Text = elide(e.name, 250, textSize)
	kind := "File"
	if e.isDir {
		kind = "Folder"
	}
	p.kind.Text = kind
	p.size.Text = formatSize(e.size, e.isDir, e.itemCount)
	p.date.Text = formatDate(e.modTime)

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		p.img.Hide()
		p.icon.Hide()
		img := canvas.NewImageFromFile(path)
		img.FillMode = canvas.ImageFillContain
		p.img = img
		p.img.Show()
	case ".svg":
		if data, err := os.ReadFile(path); err == nil {
			p.icon.Hide()
			p.img = canvas.NewImageFromResource(fyne.NewStaticResource(filepath.Base(path), data))
			p.img.FillMode = canvas.ImageFillContain
			p.img.Show()
		}
	default:
		p.img.Hide()
		p.icon.Resource = iconResource(fileIconName(e.name, e.isDir), colTextMuted)
		p.icon.Show()
	}
	canvas.Refresh(p)
}

type previewRenderer struct {
	p    *previewPane
	objs []fyne.CanvasObject
}

func (r *previewRenderer) Layout(size fyne.Size) {
	p := r.p
	p.bg.Resize(size)
	p.bg.Move(fyne.NewPos(0, 0))
	p.headerBg.Resize(fyne.NewSize(size.Width, 34))
	p.headerBg.Move(fyne.NewPos(0, 0))
	p.line.Resize(fyne.NewSize(1, size.Height))
	p.line.Move(fyne.NewPos(0, 0))
	p.header.Move(fyne.NewPos(10, centerY(34, measureHeight())))
	p.header.Resize(fyne.NewSize(size.Width-20, 18))

	previewH := size.Height - 150
	if previewH < 80 {
		previewH = 80
	}
	p.img.Resize(fyne.NewSize(size.Width-24, previewH))
	p.img.Move(fyne.NewPos(12, 46))
	p.icon.Resize(fyne.NewSize(64, 64))
	p.icon.Move(fyne.NewPos((size.Width-64)/2, 46+(previewH-64)/2))
	p.name.Move(fyne.NewPos(12, size.Height-94))
	p.name.Resize(fyne.NewSize(size.Width-24, 18))
	p.kind.Move(fyne.NewPos(12, size.Height-72))
	p.size.Move(fyne.NewPos(12, size.Height-52))
	p.date.Move(fyne.NewPos(12, size.Height-32))
}

func (r *previewRenderer) MinSize() fyne.Size { return fyne.NewSize(280, 200) }

func (r *previewRenderer) Refresh() {
	r.objs = []fyne.CanvasObject{
		r.p.bg, r.p.headerBg, r.p.line, r.p.header, r.p.img, r.p.icon,
		r.p.name, r.p.kind, r.p.size, r.p.date,
	}
	r.Layout(r.p.Size())
	canvas.Refresh(r.p)
}

func (r *previewRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *previewRenderer) Destroy()                     {}

const previewWidth float32 = 280

func (d *Dialog) setPreview(show bool) {
	d.previewVisible = show
	d.toolbar.preview.SetChecked(show)
	d.updatePreview()
	d.Refresh()
}

func (d *Dialog) updatePreview() {
	if !d.previewVisible {
		return
	}
	if e, ok := d.files.selectedEntry(); ok {
		d.preview.SetFile(e.path, e)
	} else {
		d.preview.Clear()
	}
}

// ShowPreview shows or hides the preview dock.
func (d *Dialog) ShowPreview(show bool) { d.setPreview(show) }
