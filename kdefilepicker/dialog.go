// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Mode selects the kind of file dialog.
type Mode int

const (
	// ModeOpen opens an existing file.
	ModeOpen Mode = iota
	// ModeSave chooses a file name to save to.
	ModeSave
	// ModeDirectory selects a directory.
	ModeDirectory
)

// Filter is a file type filter displayed in the "File type" combo box.
// Patterns use shell glob syntax, for example []string{"*.png", "*.jpg"}.
type Filter struct {
	Name     string
	Patterns []string
}

// AllFiles is the standard catch-all filter.
var AllFiles = Filter{Name: "All Files", Patterns: []string{"*"}}

// Dialog is the KDE 6 styled file dialog widget. It is a normal
// fyne.CanvasObject and can therefore be embedded in custom window chrome.
type Dialog struct {
	widget.BaseWidget

	mode       Mode
	title      string
	startDir   string
	currentDir string
	history    []string
	histIdx    int

	places  *placesView
	toolbar *toolbar
	nav     *breadcrumb
	files   *fileView
	bottom  *bottomPanel

	pathEdit *lineEdit
	editing  bool

	quick        *lineEdit
	quickVisible bool
	quickLocked  bool

	preview        *previewPane
	previewVisible bool

	filters    []Filter
	filterIdx  int
	showHidden bool
	showPlaces bool

	onAccept func(string)
	onCancel func()

	window            fyne.Window
	pendingFocus      fyne.Focusable
	pendingFocusTries int
	noDelayedFocus    bool // set by tests to keep timers out of the test window

	bg       *canvas.Rectangle
	divider  *canvas.Rectangle
	msgBg    *canvas.Rectangle
	msg      *canvas.Text
	msgTimer *time.Timer
}

// NewOpenDialog creates a dialog for opening an existing file.
func NewOpenDialog(startDir string) *Dialog {
	return newDialog(ModeOpen, startDir, "")
}

// NewSaveDialog creates a dialog for choosing a file name to save.
func NewSaveDialog(startDir, fileName string) *Dialog {
	return newDialog(ModeSave, startDir, fileName)
}

// NewDirectoryDialog creates a dialog for selecting a directory.
func NewDirectoryDialog(startDir string) *Dialog {
	return newDialog(ModeDirectory, startDir, "")
}

func newDialog(mode Mode, startDir, fileName string) *Dialog {
	d := &Dialog{
		mode:       mode,
		startDir:   startDir,
		filters:    []Filter{AllFiles},
		showPlaces: true,
	}
	switch mode {
	case ModeSave:
		d.title = "Save File"
	case ModeDirectory:
		d.title = "Select Folder"
	default:
		d.title = "Open File"
	}
	if startDir == "" {
		startDir, _ = os.UserHomeDir()
	}
	if abs, err := filepath.Abs(startDir); err == nil {
		startDir = abs
	}
	if st, err := os.Stat(startDir); err == nil && !st.IsDir() {
		if fileName == "" {
			fileName = st.Name()
		}
		startDir = filepath.Dir(startDir)
	} else if err != nil {
		if home, herr := os.UserHomeDir(); herr == nil {
			startDir = home
		}
	}
	d.currentDir = startDir
	d.ExtendBaseWidget(d)

	d.places = newPlacesView(newPlacesModel())
	d.places.onOpen = func(path string) { d.openDir(path) }
	d.toolbar = newToolbar()
	d.nav = newBreadcrumb()
	d.nav.onNavigate = func(path string) { d.openDir(path) }
	d.nav.onDropdown = func() { d.showPathMenu() }
	d.files = newFileView(startDir)
	d.files.onOpenDir = func(path string) { d.openDir(path) }
	d.files.onSelect = func(e fileEntry) {
		d.bottom.name.SetText(e.name)
		d.bottom.name.cursor = len([]rune(e.name))
		d.updateOpenState()
		d.updatePreview()
	}
	d.files.onActivate = func(e fileEntry) { d.accept() }
	d.files.onEscape = func() { d.cancel() }
	d.preview = newPreviewPane()
	d.bottom = newBottomPanel()
	d.bottom.name.onChanged = func(string) { d.updateOpenState() }
	d.bottom.name.onSubmitted = func(string) { d.accept() }
	d.bottom.name.onEscape = func() { d.cancel() }
	d.bottom.cancel.tap = func() { d.cancel() }
	d.bottom.open.tap = func() { d.accept() }
	d.bottom.onFilter = func(e *fyne.PointEvent) { d.showFilterMenu() }

	// Path editor (Ctrl+L).
	d.pathEdit = newLineEdit()
	d.pathEdit.onSubmitted = func(text string) {
		d.endPathEdit()
		if text != "" {
			d.openDir(text)
		}
	}
	d.pathEdit.onEscape = func() { d.endPathEdit() }

	// Quick filter.
	d.quick = newLineEdit()
	d.quick.SetPlaceholder("Filter by name…")
	d.quick.onChanged = func(text string) { d.files.SetFilter(text) }
	d.quick.onEscape = func() { d.setQuickFilter(false) }

	d.bottom.SetModeOpenLabel(d.openLabel())
	d.wireToolbar()
	d.loadDir(startDir)
	d.history = []string{startDir}
	d.histIdx = 0
	d.updateNav()
	if fileName != "" {
		d.bottom.name.SetText(fileName)
		d.bottom.name.cursor = len([]rune(fileName))
	}
	d.updateOpenState()
	return d
}

func (d *Dialog) openLabel() string {
	if d.mode == ModeSave {
		return "Save"
	}
	return "Open"
}

// Title returns the window title appropriate for the dialog mode.
func (d *Dialog) Title() string { return d.title }

// SetTitle overrides the window title.
func (d *Dialog) SetTitle(title string) {
	d.title = title
	if d.window != nil {
		d.window.SetTitle(title)
	}
}

// SetFilters replaces the file type filters. The first filter is selected.
func (d *Dialog) SetFilters(filters []Filter) {
	if len(filters) == 0 {
		filters = []Filter{AllFiles}
	}
	d.filters = filters
	if d.filterIdx >= len(filters) {
		d.filterIdx = 0
	}
	d.bottom.SetFilterLabel(filters[d.filterIdx].Name)
	d.files.SetPatterns(filters[d.filterIdx].Patterns)
}

// SetSelectedFilter selects the filter at index.
func (d *Dialog) SetSelectedFilter(index int) {
	if index < 0 || index >= len(d.filters) {
		return
	}
	d.filterIdx = index
	d.bottom.SetFilterLabel(d.filters[index].Name)
	d.files.SetPatterns(d.filters[index].Patterns)
}

// SelectedFilter returns the index of the active filter.
func (d *Dialog) SelectedFilter() int { return d.filterIdx }

// SetShowHidden controls whether dot files are listed.
func (d *Dialog) SetShowHidden(show bool) {
	d.showHidden = show
	d.files.SetShowHidden(show)
}

// SetDirectory navigates to dir.
func (d *Dialog) SetDirectory(dir string) { d.openDir(dir) }

// SetFileName pre-fills the name field.
func (d *Dialog) SetFileName(name string) {
	d.bottom.name.SetText(name)
	d.bottom.name.cursor = len([]rune(name))
	d.updateOpenState()
}

// SetOnAccept registers the callback invoked with the chosen path.
func (d *Dialog) SetOnAccept(fn func(string)) { d.onAccept = fn }

// SetOnCancel registers the callback invoked when the dialog is cancelled.
func (d *Dialog) SetOnCancel(fn func()) { d.onCancel = fn }

// CurrentDir returns the directory currently shown.
func (d *Dialog) CurrentDir() string { return d.currentDir }

// ShowInWindow places the dialog in the supplied window, sizes it to the
// reference geometry (1400x869) and focuses the name entry.
func (d *Dialog) ShowInWindow(w fyne.Window) {
	d.window = w
	w.SetTitle(d.title)
	w.SetContent(d)
	d.InstallShortcuts(w)
	w.Resize(fyne.NewSize(1400, 869))
	w.CenterOnScreen()
	d.FocusName()
	// The first layout happens before the window's focus manager has
	// traversed the new content tree; retry once the window is on screen.
	time.AfterFunc(250*time.Millisecond, func() {
		fyne.Do(func() {
			if d.window == nil {
				return
			}
			d.window.Canvas().Focus(d.bottom.name)
			d.pendingFocus = nil
		})
	})
}

// FocusName focuses the name entry.
func (d *Dialog) FocusName() { d.requestFocus(d.bottom.name) }

func (d *Dialog) requestFocus(obj fyne.Focusable) {
	d.pendingFocus = obj
	d.pendingFocusTries = 0
	d.Refresh()
	d.focusDelayed(obj)
}

// focusDelayed retries focus after the window has painted the newly shown
// widget. Fyne only caches renderers during a paint, and the focus manager
// walks cached renderers.
func (d *Dialog) focusDelayed(obj fyne.Focusable) {
	if d.noDelayedFocus {
		return
	}
	time.AfterFunc(40*time.Millisecond, func() {
		fyne.Do(func() {
			if d.window == nil {
				return
			}
			d.window.Canvas().Focus(obj)
		})
	})
}

// Close closes the window hosting the dialog, if any.
func (d *Dialog) Close() {
	if d.window != nil {
		d.window.Close()
	}
}

// Accept accepts the current selection (same as pressing the Open button).
func (d *Dialog) Accept() { d.accept() }

// Cancel cancels the dialog.
func (d *Dialog) Cancel() { d.cancel() }

// RefreshDir reloads the current directory.
func (d *Dialog) RefreshDir() {
	d.loadDir(d.currentDir)
}

func (d *Dialog) createObjects() {
	d.bg = canvas.NewRectangle(colWindow)
	d.divider = canvas.NewRectangle(colFrame)
	d.msgBg = canvas.NewRectangle(colPlaceSelection)
	d.msg = canvas.NewText("", colText)
	d.msg.TextSize = textSize
	d.msgBg.Hide()
	d.msg.Hide()
}

// CreateRenderer implements fyne.Widget.
func (d *Dialog) CreateRenderer() fyne.WidgetRenderer {
	d.createObjects()
	r := &dialogRenderer{d: d}
	r.Refresh()
	return r
}

func (d *Dialog) wireToolbar() {
	t := d.toolbar
	t.SetAction("back", d.back)
	t.SetAction("forward", d.forward)
	t.SetAction("up", d.up)
	t.SetAction("reload", func() { d.loadDir(d.currentDir) })
	t.SetAction("icons", func() { d.setView(2) })
	t.SetAction("compact", func() { d.setView(1) })
	t.SetAction("details", func() { d.setView(0) })
	t.SetAction("preview", d.togglePreview)
	t.SetAction("sort", func() { d.showSortMenu() })
	t.SetAction("bookmarks", func() { d.showBookmarksMenu() })
	t.SetAction("zoomout", func() { d.changeZoom(-1) })
	t.SetAction("zoomin", func() { d.changeZoom(1) })
	t.SetAction("newfolder", d.newFolder)
	t.SetAction("options", func() { d.showOptionsMenu() })
	t.SetAction("search", func() { d.setQuickFilter(!d.quickVisible) })
	t.slider.changed = func() { d.applyZoom(t.slider.value) }
	t.SetView(0)
}

func (d *Dialog) setView(mode int) {
	d.files.viewMode = mode
	if mode == 2 {
		if d.files.iconSize < 48 {
			d.files.iconSize = 64
		}
		sizes := []float32{16, 24, 32, 48, 64, 96, 128, 256}
		for i, s := range sizes {
			if s == d.files.iconSize {
				d.toolbar.slider.value = float32(i) / float32(len(sizes)-1)
				d.toolbar.slider.Refresh()
				break
			}
		}
	}
	d.toolbar.SetView(mode)
	d.files.Refresh()
}

func (d *Dialog) changeZoom(delta int) {
	v := d.toolbar.slider.value + float32(delta)*0.14
	d.toolbar.slider.setValue(v)
}

func (d *Dialog) applyZoom(v float32) {
	sizes := []float32{16, 24, 32, 48, 64, 96, 128, 256}
	i := int(clamp(v, 0, 1) * float32(len(sizes)-1))
	d.files.iconSize = sizes[i]
	d.files.Refresh()
}

func (d *Dialog) togglePreview() {
	// Preview is intentionally lightweight: the toolbar toggle state is kept
	// in sync by the icon button itself. The panel is implemented in preview.go.
	d.setPreview(!d.previewVisible)
}

func (d *Dialog) loadDir(dir string) {
	entries, err := readEntries(dir, true)
	if err != nil {
		d.showError(err.Error())
		return
	}
	d.currentDir = dir
	d.files.SetDir(dir, entries)
	d.files.SetShowHidden(d.showHidden)
	d.files.SetPatterns(d.filters[d.filterIdx].Patterns)
	d.nav.SetPath(dir)
	d.places.SetCurrent(dir)
	d.bottom.name.SetText("")
	d.updateNav()
	d.updateOpenState()
	d.preview.Clear()
}

func (d *Dialog) openDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(d.currentDir, dir)
	}
	dir = filepath.Clean(dir)
	if dir == d.currentDir {
		return
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		d.showError("The folder " + dir + " does not exist.")
		return
	}
	entries, err := readEntries(dir, true)
	if err != nil {
		d.showError(err.Error())
		return
	}
	d.currentDir = dir
	d.history = append(d.history[:d.histIdx+1], dir)
	d.histIdx = len(d.history) - 1
	d.files.SetDir(dir, entries)
	d.files.SetShowHidden(d.showHidden)
	d.files.SetPatterns(d.filters[d.filterIdx].Patterns)
	d.nav.SetPath(dir)
	d.places.SetCurrent(dir)
	d.bottom.name.SetText("")
	d.preview.Clear()
	d.updateNav()
	d.updateOpenState()
}

func (d *Dialog) back() {
	if d.histIdx <= 0 {
		return
	}
	d.histIdx--
	d.loadDir(d.history[d.histIdx])
}

func (d *Dialog) forward() {
	if d.histIdx >= len(d.history)-1 {
		return
	}
	d.histIdx++
	d.loadDir(d.history[d.histIdx])
}

func (d *Dialog) up() {
	parent := filepath.Dir(d.currentDir)
	if parent == d.currentDir {
		return
	}
	d.openDir(parent)
}

func (d *Dialog) updateNav() {
	canUp := filepath.Dir(d.currentDir) != d.currentDir
	d.toolbar.SetNav(d.histIdx > 0, d.histIdx < len(d.history)-1, canUp)
}

func (d *Dialog) updateOpenState() {
	enabled := strings.TrimSpace(d.bottom.name.Text()) != ""
	if d.mode == ModeDirectory {
		enabled = true
	}
	d.bottom.SetOpenEnabled(enabled)
}

func (d *Dialog) accept() {
	name := strings.TrimSpace(d.bottom.name.Text())
	var target string
	if sel, ok := d.files.selectedEntry(); ok && (name == "" || name == sel.name) {
		target = sel.path
	} else if name != "" {
		if filepath.IsAbs(name) {
			target = filepath.Clean(name)
		} else {
			target = filepath.Join(d.currentDir, name)
		}
	}
	if target == "" {
		if d.mode == ModeDirectory {
			d.deliver(d.currentDir)
		}
		return
	}
	st, err := os.Stat(target)
	if err == nil && st.IsDir() {
		if d.mode == ModeDirectory {
			d.deliver(target)
			return
		}
		d.openDir(target)
		return
	}
	if err != nil && d.mode != ModeSave {
		d.showError("The file " + target + " does not exist.")
		return
	}
	if d.mode == ModeDirectory {
		d.showError("Please select a folder.")
		return
	}
	d.deliver(target)
}

func (d *Dialog) deliver(path string) {
	if d.onAccept != nil {
		d.onAccept(path)
	}
}

func (d *Dialog) cancel() {
	if d.onCancel != nil {
		d.onCancel()
	}
}

func (d *Dialog) showError(text string) {
	d.msg.Text = text
	d.msgBg.Show()
	d.msg.Show()
	d.msg.Refresh()
	d.msgBg.Refresh()
	if d.msgTimer != nil {
		d.msgTimer.Stop()
	}
	d.msgTimer = time.AfterFunc(6*time.Second, func() {
		fyne.Do(func() {
			d.msgBg.Hide()
			d.msg.Hide()
			canvas.Refresh(d)
		})
	})
	canvas.Refresh(d)
}

func (d *Dialog) setQuickFilter(show bool) {
	d.quickVisible = show
	if show {
		d.quick.SetText("")
		d.quick.SetPlaceholder("Filter by name…")
		d.requestFocus(d.quick)
	} else {
		d.files.SetFilter("")
		d.requestFocus(d.bottom.name)
	}
	d.files.SetFilter("")
	d.Refresh()
}

func (d *Dialog) beginPathEdit() {
	d.editing = true
	d.pathEdit.SetText(d.currentDir)
	d.pathEdit.cursor = len([]rune(d.currentDir))
	d.requestFocus(d.pathEdit)
}

func (d *Dialog) endPathEdit() {
	d.editing = false
	d.requestFocus(d.bottom.name)
}

// ---------------------------------------------------------------------------

type dialogRenderer struct {
	d    *Dialog
	objs []fyne.CanvasObject
}

func (r *dialogRenderer) Layout(size fyne.Size) {
	d := r.d
	d.bg.Resize(size)
	d.bg.Move(fyne.NewPos(0, 0))

	sidebar := sidebarWidth
	if !d.showPlaces {
		sidebar = 0
	}
	d.places.Resize(fyne.NewSize(sidebarWidth, size.Height))
	d.places.Move(fyne.NewPos(0, 0))
	if sidebar > 0 {
		d.places.Show()
		d.divider.Show()
		d.divider.Resize(fyne.NewSize(1, size.Height))
		d.divider.Move(fyne.NewPos(sidebarWidth-1, 0))
	} else {
		d.places.Hide()
		d.divider.Hide()
	}

	rightX := sidebar
	rightW := size.Width - rightX
	d.toolbar.Resize(fyne.NewSize(rightW, toolbarH))
	d.toolbar.Move(fyne.NewPos(rightX, 0))

	d.nav.Resize(fyne.NewSize(rightW-12, 37))
	d.nav.Move(fyne.NewPos(rightX+6, 46))
	if d.editing {
		d.nav.Hide()
		d.pathEdit.Show()
		d.pathEdit.Resize(fyne.NewSize(rightW-12, 37))
		d.pathEdit.Move(fyne.NewPos(rightX+6, 46))
	} else {
		d.pathEdit.Hide()
		d.nav.Show()
	}

	top := float32(90)
	if d.quickVisible {
		d.quick.Show()
		d.quick.SetPlaceholder("Filter by name…")
		d.quick.Resize(fyne.NewSize(rightW-12, 31))
		d.quick.Move(fyne.NewPos(rightX+6, 89))
		top += 40
	} else {
		d.quick.Hide()
	}

	fileH := size.Height - top - bottomH
	if fileH < 50 {
		fileH = 50
	}
	fileW := rightW
	if d.previewVisible {
		fileW = rightW - previewWidth - 1
		d.preview.Show()
		d.preview.Resize(fyne.NewSize(previewWidth, fileH))
		d.preview.Move(fyne.NewPos(rightX+fileW+1, top))
	} else {
		d.preview.Hide()
	}
	d.files.Resize(fyne.NewSize(fileW, fileH))
	d.files.Move(fyne.NewPos(rightX, top))

	d.bottom.Resize(fyne.NewSize(rightW, bottomH))
	d.bottom.Move(fyne.NewPos(rightX, size.Height-bottomH))

	d.msgBg.Resize(fyne.NewSize(rightW, 34))
	d.msgBg.Move(fyne.NewPos(rightX, top))
	d.msg.Move(fyne.NewPos(rightX+12, top+8))
	d.msg.Resize(fyne.NewSize(rightW-24, 20))

	r.objs = []fyne.CanvasObject{
		d.bg, d.places, d.divider, d.toolbar, d.nav, d.pathEdit,
		d.quick, d.files, d.preview, d.bottom, d.msgBg, d.msg,
	}
}

func (r *dialogRenderer) MinSize() fyne.Size { return fyne.NewSize(900, 600) }

func (r *dialogRenderer) Refresh() {
	r.Layout(r.d.Size())
	canvas.Refresh(r.d)
}

func (r *dialogRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *dialogRenderer) Destroy()                     {}
