# Design notes: replicating KFileWidget

These notes document how the Fyne widget maps to the KDE implementation and
which measurements were used. The reference stack was:

* Plasma 6.7.5
* KIO 6.31.0 sources (`src/filewidgets/`)
* Breeze Dark colour scheme
* Noto Sans at the desktop font size
* 100% display scaling
* Reference content size: **1400 × 869 px**

The live KDE dialog was captured at 1400 × 869 and the replica was compared
against it at the same size, with identical test data.

## Source map

| KDE / KIO component | Fyne implementation |
| --- | --- |
| `KFileWidget` / `KFileWidgetPrivate::initGUI` | `Dialog`, `dialogRenderer.Layout` |
| `KFileWidgetPrivate::initToolbar` | `toolbar`, `iconButton`, `zoomSlider` |
| `KUrlNavigator`, `KUrlNavigatorButton` | `breadcrumb`, `breadcrumbRenderer` |
| `KUrlNavigatorDropDownButton` | leading chevron + `showPathMenu` |
| `KFilePlacesView`, `KFilePlacesViewDelegate` | `placesView`, `placesRenderer` |
| `KFilePlacesModel`, `user-places.xbel` | `placesModel`, `loadBookmarks` |
| `KDirOperator`, `KDirOperatorDetailView` | `fileView` (details mode) |
| icon/compact views | `fileView` modes 2 and 1 |
| `KUrlComboBox` / location edit | `lineEdit`, `bottomPanel.name` |
| `KFileFilterCombo` | `comboBox`, `entryMatches` |
| `KFileWidgetPrivate::initLocationWidget` | `bottomPanel` layout |
| `KStandardGuiItem::ok/cancel` | `pushButton` |
| Breeze style, KColorScheme, KIconLoader | `theme.go`, sampled palette, embedded Breeze icons |
| `KFileBookmarkHandler` | bookmarks menu in `menus.go` |
| KMessageWidget (errors) | `Dialog.showError` overlay |

## Layout

The top-level splitter is not a Fyne `HSplit`; its dimensions are fixed to the
KDE reference and drawn by the dialog renderer.

```text
┌───────────────────────┬────────────────────────────────────────────────────┐
│ places panel          │ toolbar                              46 px         │
│ 199 px                ├────────────────────────────────────────────────────┤
│                       │ breadcrumb (6 px inset, 37 px high)              │
│                       ├────────────────────────────────────────────────────┤
│                       │ column header                        37 px         │
│                       ├────────────────────────────────────────────────────┤
│                       │ details view (flexible), scroll bar 24 px wide    │
│                       ├────────────────────────────────────────────────────┤
│                       │ name row                             37 px         │
│                       │ file type row                        37 px         │
│                       │ action buttons                       38 px         │
│                       │ bottom panel total                  142 px         │
└───────────────────────┴────────────────────────────────────────────────────┘
```

Specific values:

| Item | Value |
| --- | --- |
| window background | `#202326` |
| list background | `#141618` |
| alternating row | `#1d1f22` |
| column header | `#292c30` |
| border / frame | `#4c4e51`, `#3f3f3f` |
| text | `#fcfcfc` |
| muted text / section headers | `#9c9ea0` |
| accent / active highlight | `#3daee9` |
| inactive list selection | `#1c2a32` |
| inactive places selection | `#1b4155` |
| toolbar button pitch | 34 px |
| toolbar icon | 22 px |
| details row height | 31 px |
| sidebar place row | 24 px |
| row icon | 16 px |
| page scroll thumb | 7 px |
| details Name column | remaining width |
| details Size column | 109 px |
| details Date column | 215 px |
| action button heights | 38 px |

## Typography

The dialog embeds Noto Sans Regular/Bold/Italic/Bold Italic. Body text is
rendered at 20 px in Fyne's 1.0 scale to match the reference desktop's large
Noto Sans rendering. `canvas.Text` is used everywhere instead of Fyne labels so
baselines and bearings can be aligned to the measured reference.

## Icons

Icons come from Breeze Dark (`breeze-icons` 6.31.0), embedded as SVG and
recoloured in memory by replacing `currentColor`. The Root place uses the
Breeze red folder icon; file rows use a curated extension → MIME icon map.
Toolbar icons are 22 px; row/place/device icons are 16 px.

## Behaviour notes

* Directory listing is folders-first, case-insensitively sorted by name by
  default, matching KDirOperator's default.
* Directory item counts are computed eagerly, like KIO's details view.
* A double-click on a directory navigates into it. A double-click on a file
  accepts it. The Open button on a selected directory navigates, matching
  KFileWidget.
* The details-view chevron expands directories into a recursive tree. With
  `SetSingleClickDirectories(true)` a click on the folder row navigates
  immediately while the chevron still expands.
* The scroll bar supports clicking the track, dragging the thumb (including a
  hover-move fallback), and wheel scrolling. The zoom slider supports clicking
  and dragging anywhere along its groove, and scales the icon grid as well as
  the row icon/height in details and compact views. None of the custom widgets
  install a special mouse cursor.
* Selecting a row fills the name field and updates the preview dock.
* History is a simple index into a path list; back/forward start disabled.
* The places model preserves XBEL order, skips hidden entries and unsupported
  URL schemes, and highlights the deepest matching place.
* Devices are read from `lsblk`; mounted filesystems get capacity bars from
  `statfs`. Non-Linux builds omit this group.
* The path dropdown, filter, sort, options and bookmark menus use the custom
  `breezeMenu` renderer rather than Fyne's stock pop-up menu.
* `Ctrl+L`, `Ctrl+I`, `Ctrl+H`, `Ctrl+Shift+N`, `F5`, `F9`, `Alt+Arrow`,
  `Enter` and `Escape` implement the corresponding KDE actions.

## Screenshot verification

The comparison procedure was:

1. Build a clean demo home and a matching `user-places.xbel`.
2. Run the real Qt/KDE file dialog with that data and capture its content at
   1400 × 869.
3. Run the Fyne demo with the same data and window content size.
4. Compare toolbar icon centres, column positions, row baselines, selection
   colours, field positions and footer controls.

The published screenshots use the clean demo data. The toolbar centres, column
edges, header baselines, breadcrumb baseline, footer labels and buttons match
the reference within 0–2 px; small differences remain in text width because
Qt uses subpixel antialiasing while Fyne renders grayscale antialiasing.

## Known differences

* Window decorations are provided by the desktop environment, as with a native
  dialog; no title bar is drawn inside the content.
* KIO remote URLs, MTP, search timelines, drag-and-drop and inline rename are
  not implemented.
* Device labels/order come from `lsblk` and can differ from Solid's ordering.
* Date rendering uses Go's local timezone by default (the reference session
  displayed UTC because of its KDE configuration).
