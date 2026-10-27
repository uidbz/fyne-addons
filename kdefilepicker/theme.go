// SPDX-License-Identifier: MIT

package kdefilepicker

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The colours below are sampled from Breeze Dark, the default Plasma 6 colour
// scheme, at 100% scale. They are deliberately constants instead of theme
// lookups: the file dialog is a pixel-faithful reproduction, not an adaptive
// Fyne widget.
var (
	colWindow         = color.NRGBA{R: 0x20, G: 0x23, B: 0x26, A: 0xff}
	colView           = color.NRGBA{R: 0x14, G: 0x16, B: 0x18, A: 0xff}
	colViewAlternate  = color.NRGBA{R: 0x1d, G: 0x1f, B: 0x22, A: 0xff}
	colHeader         = color.NRGBA{R: 0x29, G: 0x2c, B: 0x30, A: 0xff}
	colText           = color.NRGBA{R: 0xfc, G: 0xfc, B: 0xfc, A: 0xff}
	colTextMuted      = color.NRGBA{R: 0x9c, G: 0x9e, B: 0xa0, A: 0xff}
	colTextDisabled   = color.NRGBA{R: 0x6c, G: 0x6f, B: 0x71, A: 0xff}
	colBorder         = color.NRGBA{R: 0x4c, G: 0x4e, B: 0x51, A: 0xff}
	colBorderLight    = color.NRGBA{R: 0x53, G: 0x56, B: 0x59, A: 0xff}
	colFrame          = color.NRGBA{R: 0x3f, G: 0x3f, B: 0x3f, A: 0xff}
	colAccent         = color.NRGBA{R: 0x3d, G: 0xae, B: 0xe9, A: 0xff}
	colHover          = color.NRGBA{R: 0x2a, G: 0x2e, B: 0x32, A: 0xff}
	colPressed        = color.NRGBA{R: 0x3c, G: 0x3f, B: 0x41, A: 0xff}
	colSelection      = color.NRGBA{R: 0x1c, G: 0x2a, B: 0x32, A: 0xff}
	colSelectionEdge  = color.NRGBA{R: 0x1c, G: 0x3a, B: 0x4c, A: 0xff}
	colPlaceSelection = color.NRGBA{R: 0x1b, G: 0x41, B: 0x55, A: 0xff}
	colButton         = color.NRGBA{R: 0x29, G: 0x2c, B: 0x30, A: 0xff}
	colButtonDisabled = color.NRGBA{R: 0x25, G: 0x28, B: 0x2c, A: 0xff}
	colCombo          = color.NRGBA{R: 0x14, G: 0x16, B: 0x18, A: 0xff}
	colComboFocus     = color.NRGBA{R: 0x1e, G: 0x23, B: 0x28, A: 0xff}
	colScroll         = color.NRGBA{R: 0x46, G: 0x49, B: 0x4b, A: 0xff}
	colScrollHover    = color.NRGBA{R: 0x5a, G: 0x5e, B: 0x62, A: 0xff}
	colTrack          = color.NRGBA{R: 0x4b, G: 0x4e, B: 0x51, A: 0xff}
	colTicks          = color.NRGBA{R: 0x5a, G: 0x5e, B: 0x62, A: 0xff}
	colTransparent    = color.NRGBA{}
)

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
