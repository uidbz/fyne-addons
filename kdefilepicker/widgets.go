// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

func newRect(fill, stroke color.Color, strokeWidth, radius float32) *canvas.Rectangle {
	r := canvas.NewRectangle(fill)
	r.StrokeColor = stroke
	r.StrokeWidth = strokeWidth
	r.CornerRadius = radius
	return r
}

func newLine(clr color.Color) *canvas.Rectangle {
	return canvas.NewRectangle(clr)
}

// ---------------------------------------------------------------------------
// Icon button used by the toolbar.

type iconButton struct {
	widget.BaseWidget

	icon    string
	tooltip string
	color   color.Color
	enabled bool
	checked bool
	hovered bool
	pressed bool
	tap     func()

	bg  *canvas.Rectangle
	img *canvas.Image
}

func newIconButton(icon, tooltip string, tap func()) *iconButton {
	b := &iconButton{icon: icon, tooltip: tooltip, color: colText, enabled: true, tap: tap}
	b.ExtendBaseWidget(b)
	return b
}

func (b *iconButton) CreateRenderer() fyne.WidgetRenderer {
	b.bg = newRect(colTransparent, colTransparent, 0, 4)
	b.img = iconImage(b.icon, b.color, 22)
	return &iconButtonRenderer{b: b, objs: []fyne.CanvasObject{b.bg, b.img}}
}

func (b *iconButton) MinSize() fyne.Size { return fyne.NewSize(32, 30) }

func (b *iconButton) SetEnabled(enabled bool) {
	b.enabled = enabled
	b.Refresh()
}

func (b *iconButton) SetChecked(checked bool) {
	b.checked = checked
	b.Refresh()
}

func (b *iconButton) Tapped(*fyne.PointEvent) {
	if !b.enabled || b.tap == nil {
		return
	}
	b.tap()
}

func (b *iconButton) MouseIn(*desktop.MouseEvent) { b.hovered = true; b.Refresh() }
func (b *iconButton) MouseMoved(*desktop.MouseEvent) {
}
func (b *iconButton) MouseOut() { b.hovered = false; b.pressed = false; b.Refresh() }

func (b *iconButton) MouseDown(*desktop.MouseEvent) {
	if !b.enabled {
		return
	}
	b.pressed = true
	b.Refresh()
}

func (b *iconButton) MouseUp(*desktop.MouseEvent) {
	b.pressed = false
	b.Refresh()
}

func (b *iconButton) Cursor() desktop.Cursor {
	if !b.enabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

type iconButtonRenderer struct {
	b    *iconButton
	objs []fyne.CanvasObject
}

func (r *iconButtonRenderer) Layout(size fyne.Size) {
	r.b.bg.Resize(size)
	iconSize := float32(22)
	r.b.img.Resize(fyne.NewSize(iconSize, iconSize))
	r.b.img.Move(fyne.NewPos((size.Width-iconSize)/2, (size.Height-iconSize)/2))
}

func (r *iconButtonRenderer) MinSize() fyne.Size { return fyne.NewSize(32, 30) }

func (r *iconButtonRenderer) Refresh() {
	clr := r.b.color
	if !r.b.enabled {
		clr = colTextDisabled
	}
	r.b.img.Resource = iconResource(r.b.icon, clr)
	r.b.img.Refresh()

	switch {
	case r.b.pressed:
		r.b.bg.FillColor = colPressed
	case r.b.checked:
		r.b.bg.FillColor = colPressed
	case r.b.hovered:
		r.b.bg.FillColor = colHover
	default:
		r.b.bg.FillColor = colTransparent
	}
	r.b.bg.Refresh()
	canvas.Refresh(r.b)
}

func (r *iconButtonRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *iconButtonRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// Push button used for Open / Cancel.

type pushButton struct {
	widget.BaseWidget

	label   string
	icon    string
	enabled bool
	border  bool
	hovered bool
	pressed bool
	tap     func()

	bg  *canvas.Rectangle
	img *canvas.Image
	txt *canvas.Text
}

func newPushButton(label, icon string, tap func()) *pushButton {
	b := &pushButton{label: label, icon: icon, enabled: true, border: true, tap: tap}
	b.ExtendBaseWidget(b)
	return b
}

func (b *pushButton) CreateRenderer() fyne.WidgetRenderer {
	b.bg = newRect(colButton, colBorder, 1, 4)
	b.img = iconImage(b.icon, colText, 16)
	b.txt = canvas.NewText(b.label, colText)
	b.txt.TextSize = textSize
	return &pushButtonRenderer{b: b, objs: []fyne.CanvasObject{b.bg, b.img, b.txt}}
}

func (b *pushButton) SetEnabled(enabled bool) {
	b.enabled = enabled
	b.Refresh()
}

func (b *pushButton) Tapped(*fyne.PointEvent) {
	if b.enabled && b.tap != nil {
		b.tap()
	}
}

func (b *pushButton) MouseIn(*desktop.MouseEvent) { b.hovered = true; b.Refresh() }
func (b *pushButton) MouseMoved(*desktop.MouseEvent) {
}
func (b *pushButton) MouseOut() { b.hovered = false; b.pressed = false; b.Refresh() }
func (b *pushButton) MouseDown(*desktop.MouseEvent) {
	if b.enabled {
		b.pressed = true
		b.Refresh()
	}
}
func (b *pushButton) MouseUp(*desktop.MouseEvent) { b.pressed = false; b.Refresh() }
func (b *pushButton) Cursor() desktop.Cursor {
	if !b.enabled {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

type pushButtonRenderer struct {
	b    *pushButton
	objs []fyne.CanvasObject
}

func (r *pushButtonRenderer) Layout(size fyne.Size) {
	r.b.bg.Resize(size)
	txtSize := fyne.MeasureText(r.b.label, textSize, fyne.TextStyle{})
	iconSize := float32(16)
	const gap = 7
	total := iconSize + gap + txtSize.Width
	x := (size.Width - total) / 2
	r.b.img.Resize(fyne.NewSize(iconSize, iconSize))
	r.b.img.Move(fyne.NewPos(x, (size.Height-iconSize)/2+2))
	r.b.txt.Resize(fyne.NewSize(txtSize.Width+2, txtSize.Height))
	r.b.txt.Move(fyne.NewPos(x+iconSize+gap, (size.Height-txtSize.Height)/2+2))
}

func (r *pushButtonRenderer) MinSize() fyne.Size {
	txtSize := fyne.MeasureText(r.b.label, textSize, fyne.TextStyle{})
	return fyne.NewSize(txtSize.Width+16+7+24, 38)
}

func (r *pushButtonRenderer) Refresh() {
	if r.b.enabled {
		r.b.txt.Color = colText
		r.b.img.Resource = iconResource(r.b.icon, colText)
		if r.b.border {
			r.b.bg.StrokeColor = colBorder
			r.b.bg.StrokeWidth = 1
		} else {
			r.b.bg.StrokeColor = colTransparent
			r.b.bg.StrokeWidth = 0
		}
		switch {
		case r.b.pressed:
			r.b.bg.FillColor = colPressed
		case r.b.hovered:
			r.b.bg.FillColor = colHover
		default:
			r.b.bg.FillColor = colButton
		}
	} else {
		r.b.txt.Color = colTextDisabled
		r.b.img.Resource = iconResource(r.b.icon, colTextDisabled)
		r.b.bg.FillColor = colButtonDisabled
		r.b.bg.StrokeColor = colTransparent
		r.b.bg.StrokeWidth = 0
	}
	r.b.img.Refresh()
	r.b.bg.Refresh()
	r.b.txt.Refresh()
	canvas.Refresh(r.b)
}

func (r *pushButtonRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *pushButtonRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// Zoom slider.

type zoomSlider struct {
	widget.BaseWidget

	value   float32
	hovered bool
	tap     func(float32)
	changed func()

	track  *canvas.Rectangle
	ticks  []*canvas.Rectangle
	marker *canvas.Rectangle
	outer  *canvas.Circle
	inner  *canvas.Circle
}

func newZoomSlider(tap func(float32)) *zoomSlider {
	s := &zoomSlider{value: 0, tap: tap}
	s.ExtendBaseWidget(s)
	return s
}

func (s *zoomSlider) CreateRenderer() fyne.WidgetRenderer {
	s.track = newRect(colTrack, colTransparent, 0, 2)
	s.outer = canvas.NewCircle(colBorder)
	s.inner = canvas.NewCircle(colWindow)
	s.marker = newRect(colAccent, colTransparent, 0, 0)
	s.ticks = s.ticks[:0]
	for i := 0; i < 8; i++ {
		s.ticks = append(s.ticks, newRect(colTicks, colTransparent, 0, 0))
	}
	objs := []fyne.CanvasObject{s.track, s.marker, s.outer, s.inner}
	for _, t := range s.ticks {
		objs = append(objs, t)
	}
	return &zoomSliderRenderer{s: s, objs: objs}
}

func (s *zoomSlider) MinSize() fyne.Size { return fyne.NewSize(90, 30) }

func (s *zoomSlider) setValue(v float32) {
	v = clamp(v, 0, 1)
	if v == s.value {
		return
	}
	s.value = v
	if s.changed != nil {
		s.changed()
	}
	if s.tap != nil {
		s.tap(v)
	}
	s.Refresh()
}

func (s *zoomSlider) Tapped(e *fyne.PointEvent) {
	s.setValue(e.Position.X / s.Size().Width)
}

func (s *zoomSlider) Dragged(e *fyne.DragEvent) {
	s.setValue((e.Position.X + e.Dragged.DX) / s.Size().Width)
}

func (s *zoomSlider) DragEnd() {}

func (s *zoomSlider) MouseIn(*desktop.MouseEvent) {
	s.hovered = true
	s.Refresh()
}

func (s *zoomSlider) MouseMoved(*desktop.MouseEvent) {}
func (s *zoomSlider) MouseOut()                      { s.hovered = false; s.Refresh() }
func (s *zoomSlider) Cursor() desktop.Cursor         { return desktop.PointerCursor }

type zoomSliderRenderer struct {
	s    *zoomSlider
	objs []fyne.CanvasObject
}

func (r *zoomSliderRenderer) Layout(size fyne.Size) {
	const thumb = 18
	const margin = thumb / 2
	trackW := size.Width - thumb
	if trackW < 1 {
		trackW = 1
	}
	r.s.track.Resize(fyne.NewSize(trackW, 4))
	r.s.track.Move(fyne.NewPos(margin, 13))
	center := margin + r.s.value*trackW
	r.s.outer.Resize(fyne.NewSize(thumb, thumb))
	r.s.outer.Move(fyne.NewPos(center-thumb/2, 6))
	r.s.inner.Resize(fyne.NewSize(thumb-4, thumb-4))
	r.s.inner.Move(fyne.NewPos(center-thumb/2+2, 8))
	r.s.marker.Resize(fyne.NewSize(2, 6))
	r.s.marker.Move(fyne.NewPos(center-1, 24))
	for i, t := range r.s.ticks {
		x := margin + float32(i)*(trackW/7)
		t.Resize(fyne.NewSize(1, 6))
		t.Move(fyne.NewPos(x, 24))
	}
}

func (r *zoomSliderRenderer) MinSize() fyne.Size { return fyne.NewSize(90, 30) }

func (r *zoomSliderRenderer) Refresh() {
	if r.s.hovered {
		r.s.outer.FillColor = colScrollHover
	} else {
		r.s.outer.FillColor = colBorder
	}
	r.s.outer.Refresh()
	r.s.inner.FillColor = colWindow
	r.s.inner.Refresh()
	canvas.Refresh(r.s)
}

func (r *zoomSliderRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *zoomSliderRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// Text entry. Implemented from scratch so that the Breeze border, padding and
// metrics match the reference exactly (Fyne's Entry has different chrome).

type lineEdit struct {
	widget.BaseWidget

	text        []rune
	cursor      int
	placeholder string
	focused     bool
	hovered     bool
	clearable   bool

	onChanged   func(string)
	onSubmitted func(string)
	onTapped    func()
	onEscape    func()

	bg    *canvas.Rectangle
	txt   *canvas.Text
	caret *canvas.Rectangle
	ph    *canvas.Text
}

func newLineEdit() *lineEdit {
	e := &lineEdit{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *lineEdit) CreateRenderer() fyne.WidgetRenderer {
	e.bg = newRect(colCombo, colBorder, 1, 4)
	e.txt = canvas.NewText("", colText)
	e.txt.TextSize = textSize
	e.ph = canvas.NewText("", colTextMuted)
	e.ph.TextSize = textSize
	e.ph.Hide()
	e.caret = newRect(colText, colTransparent, 0, 0)
	return &lineEditRenderer{e: e, objs: []fyne.CanvasObject{e.bg, e.ph, e.txt, e.caret}}
}

func (e *lineEdit) MinSize() fyne.Size { return fyne.NewSize(120, 37) }

func (e *lineEdit) Text() string { return string(e.text) }

func (e *lineEdit) SetText(text string) {
	e.text = []rune(text)
	e.cursor = len(e.text)
	if e.onChanged != nil {
		e.onChanged(text)
	}
	e.Refresh()
}

func (e *lineEdit) SetPlaceholder(text string) {
	e.placeholder = text
	if e.ph != nil {
		e.ph.Text = text
	}
	e.Refresh()
}

func (e *lineEdit) FocusGained() {
	e.focused = true
	e.Refresh()
}

func (e *lineEdit) FocusLost() {
	e.focused = false
	e.Refresh()
}

func (e *lineEdit) Tapped(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		c.Focus(e)
	}
	e.setCursorFromX(pe.Position.X)
	if e.onTapped != nil {
		e.onTapped()
	}
}

func (e *lineEdit) DoubleTapped(pe *fyne.PointEvent) { e.Tapped(pe) }

func (e *lineEdit) MouseIn(*desktop.MouseEvent) {
	e.hovered = true
	e.Refresh()
}

func (e *lineEdit) MouseMoved(*desktop.MouseEvent) {}
func (e *lineEdit) MouseOut()                      { e.hovered = false; e.Refresh() }
func (e *lineEdit) Cursor() desktop.Cursor         { return desktop.TextCursor }

func (e *lineEdit) setCursorFromX(x float32) {
	const pad = 7.0
	text := string(e.text)
	best := 0
	for i := 0; i <= len(e.text); i++ {
		w := fyne.MeasureText(string([]rune(text)[:i]), textSize, fyne.TextStyle{}).Width
		if pad+w <= x {
			best = i
		} else {
			break
		}
	}
	e.cursor = best
	e.Refresh()
}

func (e *lineEdit) TypedRune(r rune) {
	e.text = append(e.text[:e.cursor], append([]rune{r}, e.text[e.cursor:]...)...)
	e.cursor++
	e.notifyChanged()
}

func (e *lineEdit) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyBackspace:
		if e.cursor > 0 {
			e.text = append(e.text[:e.cursor-1], e.text[e.cursor:]...)
			e.cursor--
			e.notifyChanged()
		}
	case fyne.KeyDelete:
		if e.cursor < len(e.text) {
			e.text = append(e.text[:e.cursor], e.text[e.cursor+1:]...)
			e.notifyChanged()
		}
	case fyne.KeyLeft:
		if e.cursor > 0 {
			e.cursor--
			e.Refresh()
		}
	case fyne.KeyRight:
		if e.cursor < len(e.text) {
			e.cursor++
			e.Refresh()
		}
	case fyne.KeyHome:
		e.cursor = 0
		e.Refresh()
	case fyne.KeyEnd:
		e.cursor = len(e.text)
		e.Refresh()
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.onSubmitted != nil {
			e.onSubmitted(string(e.text))
		}
	case fyne.KeyEscape:
		if e.onEscape != nil {
			e.onEscape()
		}
	}
}

func (e *lineEdit) notifyChanged() {
	if e.onChanged != nil {
		e.onChanged(string(e.text))
	}
	e.Refresh()
}

func (e *lineEdit) TypedShortcut(s fyne.Shortcut) {
	clip := fyne.CurrentApp().Clipboard()
	switch s.(type) {
	case *fyne.ShortcutPaste:
		text := strings.ReplaceAll(clip.Content(), "\n", "")
		e.text = append(e.text[:e.cursor], append([]rune(text), e.text[e.cursor:]...)...)
		e.cursor += len([]rune(text))
		e.notifyChanged()
	case *fyne.ShortcutCut:
		clip.SetContent(string(e.text))
		e.text = nil
		e.cursor = 0
		e.notifyChanged()
	case *fyne.ShortcutCopy:
		clip.SetContent(string(e.text))
	}
}

type lineEditRenderer struct {
	e    *lineEdit
	objs []fyne.CanvasObject
}

const editPadX = 7.0

func (r *lineEditRenderer) Layout(size fyne.Size) {
	r.e.bg.Resize(size)
	r.e.bg.Move(fyne.NewPos(0, 0))

	textH := fyne.MeasureText("Ag", textSize, fyne.TextStyle{}).Height
	textY := (size.Height-textH)/2 + 1

	full := string(r.e.text)
	avail := size.Width - 2*editPadX
	display, offset := visibleText(full, r.e.cursor, avail)
	r.e.txt.Text = display
	r.e.txt.Resize(fyne.NewSize(avail, textH))
	r.e.txt.Move(fyne.NewPos(editPadX, textY))

	r.e.ph.Text = r.e.placeholder
	r.e.ph.Resize(fyne.NewSize(avail, textH))
	r.e.ph.Move(fyne.NewPos(editPadX, textY))

	cx := float32(editPadX)
	if offset <= r.e.cursor && r.e.cursor <= offset+len([]rune(display)) {
		prefix := string([]rune(full)[offset:r.e.cursor])
		cx += fyne.MeasureText(prefix, textSize, fyne.TextStyle{}).Width
	}
	r.e.caret.Resize(fyne.NewSize(1, textH-2))
	r.e.caret.Move(fyne.NewPos(cx, textY+1))
}

func visibleText(full string, cursor int, avail float32) (string, int) {
	runes := []rune(full)
	if fyne.MeasureText(full, textSize, fyne.TextStyle{}).Width <= avail {
		return full, 0
	}
	start := 0
	for start < cursor {
		if fyne.MeasureText(string(runes[start:cursor]), textSize, fyne.TextStyle{}).Width <= avail-8 {
			break
		}
		start++
	}
	end := len(runes)
	for end > start && fyne.MeasureText(string(runes[start:end]), textSize, fyne.TextStyle{}).Width > avail {
		end--
	}
	return string(runes[start:end]), start
}

func (r *lineEditRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 37) }

func (r *lineEditRenderer) Refresh() {
	if r.e.focused {
		r.e.bg.FillColor = colComboFocus
		r.e.bg.StrokeColor = colAccent
		r.e.bg.StrokeWidth = 2
	} else if r.e.hovered {
		r.e.bg.FillColor = colCombo
		r.e.bg.StrokeColor = colBorderLight
		r.e.bg.StrokeWidth = 1
	} else {
		r.e.bg.FillColor = colCombo
		r.e.bg.StrokeColor = colBorder
		r.e.bg.StrokeWidth = 1
	}
	r.e.bg.Refresh()

	r.e.txt.Color = colText
	r.e.txt.Refresh()
	r.e.ph.Text = r.e.placeholder
	if len(r.e.text) == 0 && r.e.placeholder != "" {
		r.e.ph.Show()
	} else {
		r.e.ph.Hide()
	}
	r.e.ph.Refresh()

	if r.e.focused {
		r.e.caret.Show()
	} else {
		r.e.caret.Hide()
	}
	r.e.caret.Refresh()
	r.Layout(r.e.Size())
	canvas.Refresh(r.e)
}

func (r *lineEditRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *lineEditRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// Combo box (file type). Only the visual part is custom; the choices are
// presented in a regular Fyne popup menu.

type comboBox struct {
	widget.BaseWidget

	value   string
	hovered bool
	tap     func(*fyne.PointEvent)

	bg    *canvas.Rectangle
	txt   *canvas.Text
	clear *canvas.Image
	arrow *canvas.Image
}

func newComboBox(value string, tap func(*fyne.PointEvent)) *comboBox {
	c := &comboBox{value: value, tap: tap}
	c.ExtendBaseWidget(c)
	return c
}

func (c *comboBox) CreateRenderer() fyne.WidgetRenderer {
	c.bg = newRect(colCombo, colBorder, 1, 4)
	c.txt = canvas.NewText(c.value, colText)
	c.txt.TextSize = textSize
	c.clear = iconImage("edit-clear", colTextMuted, 14)
	c.arrow = iconImage("arrow-down", colText, 12)
	return &comboBoxRenderer{c: c, objs: []fyne.CanvasObject{c.bg, c.txt, c.clear, c.arrow}}
}

func (c *comboBox) MinSize() fyne.Size { return fyne.NewSize(200, 37) }

func (c *comboBox) SetValue(v string) { c.value = v; c.Refresh() }

func (c *comboBox) Tapped(e *fyne.PointEvent) {
	if c.tap != nil {
		c.tap(e)
	}
}

func (c *comboBox) MouseIn(*desktop.MouseEvent) { c.hovered = true; c.Refresh() }
func (c *comboBox) MouseMoved(*desktop.MouseEvent) {
}
func (c *comboBox) MouseOut()              { c.hovered = false; c.Refresh() }
func (c *comboBox) Cursor() desktop.Cursor { return desktop.PointerCursor }

type comboBoxRenderer struct {
	c    *comboBox
	objs []fyne.CanvasObject
}

func (r *comboBoxRenderer) Layout(size fyne.Size) {
	r.c.bg.Resize(size)
	textH := fyne.MeasureText("Ag", textSize, fyne.TextStyle{}).Height
	y := (size.Height - textH) / 2
	r.c.txt.Resize(fyne.NewSize(size.Width-70, textH))
	r.c.txt.Move(fyne.NewPos(editPadX, y+1))
	r.c.clear.Resize(fyne.NewSize(14, 14))
	r.c.clear.Move(fyne.NewPos(size.Width-45, (size.Height-14)/2))
	r.c.arrow.Resize(fyne.NewSize(12, 12))
	r.c.arrow.Move(fyne.NewPos(size.Width-25, (size.Height-12)/2))
}

func (r *comboBoxRenderer) MinSize() fyne.Size { return fyne.NewSize(200, 37) }

func (r *comboBoxRenderer) Refresh() {
	r.c.txt.Text = r.c.value
	r.c.txt.Color = colText
	r.c.txt.Refresh()
	if r.c.hovered {
		r.c.bg.StrokeColor = colBorderLight
	} else {
		r.c.bg.StrokeColor = colBorder
	}
	r.c.bg.Refresh()
	if len(r.c.value) == 0 {
		r.c.clear.Hide()
	} else {
		r.c.clear.Show()
	}
	r.c.clear.Refresh()
	r.Layout(r.c.Size())
	canvas.Refresh(r.c)
}

func (r *comboBoxRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *comboBoxRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// Helpers.

func measureHeight() float32 {
	return fyne.MeasureText("Ag", textSize, fyne.TextStyle{}).Height
}

func centerY(containerH, contentH float32) float32 {
	return float32(math.Round(float64((containerH - contentH) / 2)))
}
