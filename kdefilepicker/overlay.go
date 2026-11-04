// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// overlayMargin is the gap kept between the dialog and the window edges
// when the dialog is shown as an overlay.
const overlayMargin = float32(24)

// colOverlayDim dims the parent content behind the overlay dialog, in the
// spirit of KWin's modal dialog dimming.
var colOverlayDim = color.NRGBA{R: 0, G: 0, B: 0, A: 0x66}

// ShowOverlay presents the dialog as an overlay on the canvas of w: the
// parent content is dimmed and the dialog is centered on top, with the
// lifecycle of a regular Fyne dialog. Accepting or cancelling dismisses the
// overlay automatically, and Close dismisses it when shown this way. The
// dialog scales with the window up to the reference size (1400x869) and
// never shrinks below its minimum chrome size.
func (d *Dialog) ShowOverlay(w fyne.Window) {
	if w == nil || w.Canvas() == nil {
		return
	}
	d.window = w
	if d.overlay == nil {
		d.overlay = newDialogOverlay(d)
	}
	if d.overlayShown {
		d.Dismiss()
	}
	d.overlayShown = true
	w.Canvas().Overlays().Add(d.overlay)
	d.overlay.Resize(w.Canvas().Size())
	d.InstallShortcuts(w)
	d.FocusName()
	d.focusNameAfterPaint()
}

// ShowingOverlay reports whether the dialog is currently presented as an
// overlay.
func (d *Dialog) ShowingOverlay() bool { return d.overlayShown }

// Dismiss removes an overlay presentation and uninstalls the keyboard
// shortcuts. It is a no-op when the dialog is not shown as an overlay.
// Accept and Cancel call this automatically in overlay mode.
func (d *Dialog) Dismiss() {
	if !d.overlayShown || d.overlay == nil {
		return
	}
	d.overlayShown = false
	if d.window != nil {
		if c := d.window.Canvas(); c != nil {
			c.Overlays().Remove(d.overlay)
		}
	}
	d.RemoveShortcuts()
	if d.msgTimer != nil {
		d.msgTimer.Stop()
	}
}

// dialogOverlay is the full-canvas layer used by ShowOverlay: it dims the
// parent window and centers the dialog. Fyne resizes overlays to the full
// canvas size, so Layout keeps the dialog centered when the window resizes.
type dialogOverlay struct {
	widget.BaseWidget

	d   *Dialog
	dim *canvas.Rectangle
}

func newDialogOverlay(d *Dialog) *dialogOverlay {
	o := &dialogOverlay{d: d}
	o.dim = canvas.NewRectangle(colOverlayDim)
	o.ExtendBaseWidget(o)
	return o
}

// CreateRenderer implements fyne.Widget.
func (o *dialogOverlay) CreateRenderer() fyne.WidgetRenderer {
	return &dialogOverlayRenderer{o: o}
}

type dialogOverlayRenderer struct {
	o    *dialogOverlay
	objs []fyne.CanvasObject
}

func (r *dialogOverlayRenderer) Layout(size fyne.Size) {
	o := r.o
	o.dim.Resize(size)
	o.dim.Move(fyne.NewPos(0, 0))

	w := min(size.Width-2*overlayMargin, refWidth)
	h := min(size.Height-2*overlayMargin, refHeight)
	if w < minDialogWidth {
		w = min(size.Width, minDialogWidth)
	}
	if h < minDialogHeight {
		h = min(size.Height, minDialogHeight)
	}
	o.d.Resize(fyne.NewSize(w, h))
	o.d.Move(fyne.NewPos((size.Width-w)/2, (size.Height-h)/2))
	r.objs = []fyne.CanvasObject{o.dim, o.d}
}

func (r *dialogOverlayRenderer) MinSize() fyne.Size { return fyne.NewSize(1, 1) }

func (r *dialogOverlayRenderer) Refresh() {
	r.Layout(r.o.Size())
	canvas.Refresh(r.o)
}

func (r *dialogOverlayRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *dialogOverlayRenderer) Destroy()                     {}
