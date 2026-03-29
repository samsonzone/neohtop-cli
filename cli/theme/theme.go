package theme

import "image/color"

// Theme defines the color palette for the TUI
type Theme struct {
	Name     string
	Label    string
	Base     color.Color
	Mantle   color.Color
	Crust    color.Color
	Text     color.Color
	Subtext0 color.Color
	Subtext1 color.Color
	Surface0 color.Color
	Surface1 color.Color
	Surface2 color.Color
	Overlay0 color.Color
	Overlay1 color.Color
	Blue     color.Color
	Lavender color.Color
	Sapphire color.Color
	Sky      color.Color
	Red      color.Color
	Maroon   color.Color
	Peach    color.Color
	Yellow   color.Color
	Green    color.Color
	Teal     color.Color

	// Charm-specific accent colors (purple/pink gradient palette)
	Purple  color.Color
	Pink    color.Color
	Indigo  color.Color
	Fuchsia color.Color
}
