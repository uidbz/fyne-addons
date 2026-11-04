// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestOverlayShowAndAutoDismiss(t *testing.T) {
	test.NewApp()
	dir := makeTestDir(t)
	host := test.NewWindow(widget.NewLabel("host window"))
	defer host.Close()
	host.Resize(fyne.NewSize(1200, 800))

	d := NewOpenDialog(dir)
	d.noDelayedFocus = true

	d.ShowOverlay(host)
	if !d.ShowingOverlay() {
		t.Fatal("dialog not reported as shown")
	}
	if len(host.Canvas().Overlays().List()) != 1 {
		t.Fatalf("overlay stack = %d objects, want 1", len(host.Canvas().Overlays().List()))
	}
	// The dialog is centered and inset by overlayMargin within the bounds.
	if s := d.Size(); s.Width != 1200-2*overlayMargin || s.Height != 800-2*overlayMargin {
		t.Errorf("overlay dialog size = %v", s)
	}
	if len(d.installed) == 0 {
		t.Fatal("shortcuts not installed on ShowOverlay")
	}

	// Accepting delivers the path and dismisses the overlay automatically.
	var got string
	d.SetOnAccept(func(p string) { got = p })
	d.SetFileName("alpha.txt")
	d.Accept()
	if want := filepath.Join(dir, "alpha.txt"); got != want {
		t.Fatalf("accept delivered %q, want %q", got, want)
	}
	if d.ShowingOverlay() || len(host.Canvas().Overlays().List()) != 0 {
		t.Fatal("overlay not dismissed after accept")
	}
	if len(d.installed) != 0 {
		t.Fatal("shortcuts not removed after auto-dismiss")
	}

	// Cancelling delivers the cancel callback and dismisses.
	var cancelled bool
	d.SetOnCancel(func() { cancelled = true })
	d.ShowOverlay(host)
	d.Cancel()
	if !cancelled {
		t.Fatal("cancel callback not fired")
	}
	if len(host.Canvas().Overlays().List()) != 0 {
		t.Fatal("overlay not dismissed after cancel")
	}

	// Re-showing works and does not stack overlays or shortcut handlers.
	d.ShowOverlay(host)
	d.ShowOverlay(host)
	if len(host.Canvas().Overlays().List()) != 1 {
		t.Fatalf("overlay stack = %d objects, want 1", len(host.Canvas().Overlays().List()))
	}
	d.Dismiss()
	if len(host.Canvas().Overlays().List()) != 0 {
		t.Fatal("Dismiss did not remove the overlay")
	}

	// Close() in overlay mode dismisses instead of closing the host window.
	d.ShowOverlay(host)
	d.Close()
	if v, ok := host.(interface{ Visible() bool }); ok && !v.Visible() {
		t.Fatal("Close closed the host window in overlay mode")
	}
	if len(host.Canvas().Overlays().List()) != 0 {
		t.Fatal("Close did not dismiss the overlay")
	}
}

func TestOverlayScalesWithHost(t *testing.T) {
	test.NewApp()
	d := NewOpenDialog("")
	d.noDelayedFocus = true
	host := test.NewWindow(widget.NewLabel("host window"))
	defer host.Close()
	host.Resize(fyne.NewSize(2000, 1400))
	d.ShowOverlay(host)
	if s := d.Size(); s.Width != refWidth || s.Height != refHeight {
		t.Errorf("dialog size on large host = %v, want reference %vx%v", s, refWidth, refHeight)
	}
	host.Resize(fyne.NewSize(700, 500))
	if s := d.Size(); s.Width != 700 || s.Height != 500 {
		t.Errorf("dialog size on small host = %v, want clamped to host", s)
	}
}

func TestButtonLabels(t *testing.T) {
	test.NewApp()
	d := NewSaveDialog(makeTestDir(t), "")
	w := test.NewWindow(d)
	defer w.Close()
	if d.bottom.open.label != "Save" {
		t.Fatalf("default save label = %q", d.bottom.open.label)
	}
	d.SetOpenLabel("Save File")
	d.SetCancelLabel("Discard")
	if d.bottom.open.label != "Save File" || d.bottom.open.txt.Text != "Save File" {
		t.Errorf("open label not applied: %q / %q", d.bottom.open.label, d.bottom.open.txt.Text)
	}
	if d.bottom.cancel.label != "Discard" || d.bottom.cancel.txt.Text != "Discard" {
		t.Errorf("cancel label not applied: %q / %q", d.bottom.cancel.label, d.bottom.cancel.txt.Text)
	}
	d.SetOpenLabel("  ")
	if d.bottom.open.label != "Save File" {
		t.Error("blank label should be ignored")
	}
}
