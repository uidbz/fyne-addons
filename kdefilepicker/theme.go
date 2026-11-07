// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette collects every colour the dialog paints with. The defaults are
// sampled from Breeze Dark, the default Plasma 6 colour scheme, at 100%
// scale; the file dialog is a pixel-faithful reproduction, not an adaptive
// Fyne widget. Hosts that want the dialog to match their own visual identity
// can replace the whole palette with SetPalette before creating a Dialog
// (existing dialogs pick the new colours up on their next refresh).
type Palette struct {
	Window         color.Color // dialog chrome background
	View           color.Color // file list background
	ViewAlternate  color.Color // alternating list stripe
	Header         color.Color // column header, menus
	Text           color.Color // primary text
	TextMuted      color.Color // secondary text, placeholders
	TextDisabled   color.Color // disabled text
	Border         color.Color // input and panel borders
	BorderLight    color.Color // lighter border variant
	Frame          color.Color // sidebar divider
	Accent         color.Color // focus rings, links, selection edges
	Hover          color.Color // row hover wash
	Pressed        color.Color // pressed row wash
	Selection      color.Color // selected file row
	SelectionEdge  color.Color // selected row border
	PlaceSelection color.Color // selected sidebar place row
	Button         color.Color // button background
	ButtonDisabled color.Color // disabled button background
	Combo          color.Color // text entry background
	ComboFocus     color.Color // text entry background while focused
	Scroll         color.Color // scroll bar thumb
	ScrollHover    color.Color // scroll bar thumb while hovered
	Track          color.Color // scroll bar track
	Ticks          color.Color // zoom slider ticks, capacity bars
	OverlayDim     color.Color // parent content behind an overlay dialog
}

// BreezeDarkPalette returns the default Breeze Dark colour set.
func BreezeDarkPalette() Palette {
	return Palette{
		Window:         color.NRGBA{R: 0x20, G: 0x23, B: 0x26, A: 0xff},
		View:           color.NRGBA{R: 0x14, G: 0x16, B: 0x18, A: 0xff},
		ViewAlternate:  color.NRGBA{R: 0x1d, G: 0x1f, B: 0x22, A: 0xff},
		Header:         color.NRGBA{R: 0x29, G: 0x2c, B: 0x30, A: 0xff},
		Text:           color.NRGBA{R: 0xfc, G: 0xfc, B: 0xfc, A: 0xff},
		TextMuted:      color.NRGBA{R: 0x9c, G: 0x9e, B: 0xa0, A: 0xff},
		TextDisabled:   color.NRGBA{R: 0x6c, G: 0x6f, B: 0x71, A: 0xff},
		Border:         color.NRGBA{R: 0x4c, G: 0x4e, B: 0x51, A: 0xff},
		BorderLight:    color.NRGBA{R: 0x53, G: 0x56, B: 0x59, A: 0xff},
		Frame:          color.NRGBA{R: 0x3f, G: 0x3f, B: 0x3f, A: 0xff},
		Accent:         color.NRGBA{R: 0x3d, G: 0xae, B: 0xe9, A: 0xff},
		Hover:          color.NRGBA{R: 0x2a, G: 0x2e, B: 0x32, A: 0xff},
		Pressed:        color.NRGBA{R: 0x3c, G: 0x3f, B: 0x41, A: 0xff},
		Selection:      color.NRGBA{R: 0x1c, G: 0x2a, B: 0x32, A: 0xff},
		SelectionEdge:  color.NRGBA{R: 0x1c, G: 0x3a, B: 0x4c, A: 0xff},
		PlaceSelection: color.NRGBA{R: 0x1b, G: 0x41, B: 0x55, A: 0xff},
		Button:         color.NRGBA{R: 0x29, G: 0x2c, B: 0x30, A: 0xff},
		ButtonDisabled: color.NRGBA{R: 0x25, G: 0x28, B: 0x2c, A: 0xff},
		Combo:          color.NRGBA{R: 0x14, G: 0x16, B: 0x18, A: 0xff},
		ComboFocus:     color.NRGBA{R: 0x1e, G: 0x23, B: 0x28, A: 0xff},
		Scroll:         color.NRGBA{R: 0x46, G: 0x49, B: 0x4b, A: 0xff},
		ScrollHover:    color.NRGBA{R: 0x5a, G: 0x5e, B: 0x62, A: 0xff},
		Track:          color.NRGBA{R: 0x4b, G: 0x4e, B: 0x51, A: 0xff},
		Ticks:          color.NRGBA{R: 0x5a, G: 0x5e, B: 0x62, A: 0xff},
		OverlayDim:     color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66},
	}
}

// The package-level colour variables are what the renderers actually read.
// They start out as the Breeze Dark palette and are swapped wholesale by
// SetPalette.
var (
	colWindow         = toNRGBA(BreezeDarkPalette().Window)
	colView           = toNRGBA(BreezeDarkPalette().View)
	colViewAlternate  = toNRGBA(BreezeDarkPalette().ViewAlternate)
	colHeader         = toNRGBA(BreezeDarkPalette().Header)
	colText           = toNRGBA(BreezeDarkPalette().Text)
	colTextMuted      = toNRGBA(BreezeDarkPalette().TextMuted)
	colTextDisabled   = toNRGBA(BreezeDarkPalette().TextDisabled)
	colBorder         = toNRGBA(BreezeDarkPalette().Border)
	colBorderLight    = toNRGBA(BreezeDarkPalette().BorderLight)
	colFrame          = toNRGBA(BreezeDarkPalette().Frame)
	colAccent         = toNRGBA(BreezeDarkPalette().Accent)
	colHover          = toNRGBA(BreezeDarkPalette().Hover)
	colPressed        = toNRGBA(BreezeDarkPalette().Pressed)
	colSelection      = toNRGBA(BreezeDarkPalette().Selection)
	colSelectionEdge  = toNRGBA(BreezeDarkPalette().SelectionEdge)
	colPlaceSelection = toNRGBA(BreezeDarkPalette().PlaceSelection)
	colButton         = toNRGBA(BreezeDarkPalette().Button)
	colButtonDisabled = toNRGBA(BreezeDarkPalette().ButtonDisabled)
	colCombo          = toNRGBA(BreezeDarkPalette().Combo)
	colComboFocus     = toNRGBA(BreezeDarkPalette().ComboFocus)
	colScroll         = toNRGBA(BreezeDarkPalette().Scroll)
	colScrollHover    = toNRGBA(BreezeDarkPalette().ScrollHover)
	colTrack          = toNRGBA(BreezeDarkPalette().Track)
	colTicks          = toNRGBA(BreezeDarkPalette().Ticks)
	colTransparent    = color.NRGBA{}
)

func toNRGBA(c color.Color) color.NRGBA {
	if c == nil {
		return color.NRGBA{}
	}
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}

// SetPalette installs p as the dialog palette. Nil fields keep their current
// value. Call it before creating a Dialog; dialogs that are already built
// repaint with the new palette on their next refresh. It is not safe to
// change the palette concurrently with painting.
func SetPalette(p Palette) {
	set := func(dst *color.NRGBA, v color.Color) {
		if v != nil {
			*dst = toNRGBA(v)
		}
	}
	set(&colWindow, p.Window)
	set(&colView, p.View)
	set(&colViewAlternate, p.ViewAlternate)
	set(&colHeader, p.Header)
	set(&colText, p.Text)
	set(&colTextMuted, p.TextMuted)
	set(&colTextDisabled, p.TextDisabled)
	set(&colBorder, p.Border)
	set(&colBorderLight, p.BorderLight)
	set(&colFrame, p.Frame)
	set(&colAccent, p.Accent)
	set(&colHover, p.Hover)
	set(&colPressed, p.Pressed)
	set(&colSelection, p.Selection)
	set(&colSelectionEdge, p.SelectionEdge)
	set(&colPlaceSelection, p.PlaceSelection)
	set(&colButton, p.Button)
	set(&colButtonDisabled, p.ButtonDisabled)
	set(&colCombo, p.Combo)
	set(&colComboFocus, p.ComboFocus)
	set(&colScroll, p.Scroll)
	set(&colScrollHover, p.ScrollHover)
	set(&colTrack, p.Track)
	set(&colTicks, p.Ticks)
	set(&colOverlayDim, p.OverlayDim)
}

// GetPalette returns the palette the dialog currently paints with.
func GetPalette() Palette {
	return Palette{
		Window:         colWindow,
		View:           colView,
		ViewAlternate:  colViewAlternate,
		Header:         colHeader,
		Text:           colText,
		TextMuted:      colTextMuted,
		TextDisabled:   colTextDisabled,
		Border:         colBorder,
		BorderLight:    colBorderLight,
		Frame:          colFrame,
		Accent:         colAccent,
		Hover:          colHover,
		Pressed:        colPressed,
		Selection:      colSelection,
		SelectionEdge:  colSelectionEdge,
		PlaceSelection: colPlaceSelection,
		Button:         colButton,
		ButtonDisabled: colButtonDisabled,
		Combo:          colCombo,
		ComboFocus:     colComboFocus,
		Scroll:         colScroll,
		ScrollHover:    colScrollHover,
		Track:          colTrack,
		Ticks:          colTicks,
		OverlayDim:     colOverlayDim,
	}
}

const (
	// Sizes taken from the default Breeze style at 100% scale.
	textSize     float32 = 20
	sidebarWidth float32 = 199
	toolbarH     float32 = 46
	breadcrumbH  float32 = 44
	headerH      float32 = 37
	rowH         float32 = 31
	bottomH      float32 = 142
	formRowH     float32 = 37
)

// Theme is the Breeze Dark theme used by the replica. Install it on the
// application before creating a Dialog so that pop-up menus, scroll bars and
// other transient Fyne widgets use the same palette and fonts.
type Theme struct{}

// NewTheme returns a Breeze Dark theme matching the KDE 6 file dialog.
func NewTheme() fyne.Theme { return &Theme{} }

var _ fyne.Theme = (*Theme)(nil)

// Color implements fyne.Theme.
func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colWindow
	case theme.ColorNameForeground:
		return colText
	case theme.ColorNameButton:
		return colButton
	case theme.ColorNameDisabledButton:
		return colButtonDisabled
	case theme.ColorNameDisabled:
		return colTextDisabled
	case theme.ColorNamePlaceHolder:
		return colTextMuted
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return colAccent
	case theme.ColorNameHover:
		return colHover
	case theme.ColorNamePressed:
		return colPressed
	case theme.ColorNameSelection:
		return colAccent
	case theme.ColorNameInputBackground:
		return colCombo
	case theme.ColorNameInputBorder:
		return colBorder
	case theme.ColorNameScrollBar:
		return colScroll
	case theme.ColorNameScrollBarBackground:
		return colTransparent
	case theme.ColorNameSeparator:
		return colBorder
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x99}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{A: 0x88}
	case theme.ColorNameMenuBackground:
		return colHeader
	case theme.ColorNameForegroundOnPrimary:
		return colText
	case theme.ColorNameError:
		return color.NRGBA{R: 0xda, G: 0x44, B: 0x53, A: 0xff}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x27, G: 0xae, B: 0x60, A: 0xff}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xfd, G: 0xbc, B: 0x4b, A: 0xff}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

// Font implements fyne.Theme.
func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	loadFonts()
	switch {
	case style.Bold && style.Italic:
		return fontBoldItalic
	case style.Bold:
		return fontBold
	case style.Italic:
		return fontItalic
	}
	return fontRegular
}

// Icon implements fyne.Theme.
func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size implements fyne.Theme.
func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return textSize
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 7
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameInputRadius:
		return 4
	case theme.SizeNameSelectionRadius:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}
