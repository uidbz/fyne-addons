// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type bottomPanel struct {
	widget.BaseWidget

	name        *lineEdit
	filter      *comboBox
	open        *pushButton
	cancel      *pushButton
	nameLabel   *canvas.Text
	filterLabel *canvas.Text

	onFilter func(*fyne.PointEvent)

	bg      *canvas.Rectangle
	topLine *canvas.Rectangle
}

func newBottomPanel() *bottomPanel {
	b := &bottomPanel{}
	b.bg = canvas.NewRectangle(colWindow)
	b.topLine = canvas.NewRectangle(colBorder)
	b.nameLabel = canvas.NewText("Name:", colText)
	b.nameLabel.TextSize = textSize
	b.filterLabel = canvas.NewText("File type:", colText)
	b.filterLabel.TextSize = textSize
	b.name = newLineEdit()
	b.filter = newComboBox("All Files", func(e *fyne.PointEvent) {
		if b.onFilter != nil {
			b.onFilter(e)
		}
	})
	b.open = newPushButton("Open", "document-open", nil)
	b.open.SetEnabled(false)
	b.open.border = false
	b.cancel = newPushButton("Cancel", "dialog-cancel", nil)
	b.ExtendBaseWidget(b)
	return b
}

func (b *bottomPanel) CreateRenderer() fyne.WidgetRenderer {
	return &bottomPanelRenderer{b: b}
}

func (b *bottomPanel) MinSize() fyne.Size { return fyne.NewSize(600, bottomH) }

func (b *bottomPanel) SetModeOpenLabel(label string) { b.open.label = label }
func (b *bottomPanel) SetOpenEnabled(enabled bool)   { b.open.SetEnabled(enabled) }
func (b *bottomPanel) SetFilterLabel(label string)   { b.filter.SetValue(label) }

type bottomPanelRenderer struct {
	b    *bottomPanel
	objs []fyne.CanvasObject
}

func (r *bottomPanelRenderer) Layout(size fyne.Size) {
	b := r.b
	b.bg.Resize(size)
	b.bg.Move(fyne.NewPos(0, 0))
	b.topLine.Resize(fyne.NewSize(size.Width, 1))
	b.topLine.Move(fyne.NewPos(0, 0))

	fieldX := float32(101)
	fieldW := size.Width - fieldX - 9

	b.nameLabel.Move(fyne.NewPos(28, 8+centerY(formRowH, measureHeight())+1))
	b.nameLabel.Resize(fyne.NewSize(70, 18))
	b.name.Resize(fyne.NewSize(fieldW, formRowH))
	b.name.Move(fyne.NewPos(fieldX, 8))

	b.filterLabel.Move(fyne.NewPos(3, 51+centerY(formRowH, measureHeight())+1))
	b.filterLabel.Resize(fyne.NewSize(90, 18))
	b.filter.Resize(fyne.NewSize(fieldW, formRowH))
	b.filter.Move(fyne.NewPos(fieldX, 51))

	cancelW := float32(100)
	openW := float32(84)
	cancelX := size.Width - 10 - cancelW
	openX := cancelX - 10 - openW
	b.cancel.Resize(fyne.NewSize(cancelW, 38))
	b.cancel.Move(fyne.NewPos(cancelX, 95))
	b.open.Resize(fyne.NewSize(openW, 38))
	b.open.Move(fyne.NewPos(openX, 95))
}

func (r *bottomPanelRenderer) MinSize() fyne.Size { return fyne.NewSize(600, bottomH) }

func (r *bottomPanelRenderer) Refresh() {
	r.objs = []fyne.CanvasObject{
		r.b.bg, r.b.topLine,
		r.b.nameLabel, r.b.name,
		r.b.filterLabel, r.b.filter,
		r.b.open, r.b.cancel,
	}
	r.Layout(r.b.Size())
	canvas.Refresh(r.b)
}

func (r *bottomPanelRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *bottomPanelRenderer) Destroy()                     {}
