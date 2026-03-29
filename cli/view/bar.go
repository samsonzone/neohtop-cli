package view

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// btop-style progress bar using braille dot-matrix characters.
//
// Braille characters (U+2800–U+28FF) render as a 2×4 dot grid per cell,
// giving the characteristic "pixelated / dotted" look of btop.
//
// Filled:  ⣿ (all 8 dots lit) in the fill color
// Half:    ⡇ (left column, 4 dots) for sub-character precision
// Empty:   ⠶ (middle 4 dots) in a dim color — shows the "track"
//
// Example:  ⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡇⠶⠶⠶⠶⠶⠶⠶⠶⠶⠶

const (
	brailleFull  = "⣿" // U+28FF — all 8 dots (filled)
	brailleHalf  = "⡇" // U+2847 — left column only (half-fill)
	brailleEmpty = "⠶" // U+2836 — middle dots (empty track)
)

// BarStyle controls how a btop bar is rendered.
type BarStyle struct {
	FillColor   color.Color // primary fill color
	AccentColor color.Color // lighter accent (nil = same as FillColor)
	EmptyColor  color.Color // dim color for empty track
	Width       int         // total character width of the bar
}

// RenderBar draws a btop-style horizontal bar at the given percentage (0.0–1.0).
// If AccentColor is set, the filled portion alternates between FillColor and
// AccentColor in bands, creating the btop gradient/striped effect.
func RenderBar(pct float64, style BarStyle) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	w := style.Width
	if w < 1 {
		w = 10
	}

	// Calculate filled characters (with half-char precision)
	filledExact := pct * float64(w)
	filledFull := int(filledExact)
	remainder := filledExact - float64(filledFull)

	fillStyle := lipgloss.NewStyle().Foreground(style.FillColor)
	accentStyle := fillStyle
	hasAccent := style.AccentColor != nil
	if hasAccent {
		accentStyle = lipgloss.NewStyle().Foreground(style.AccentColor)
	}
	emptyStyle := lipgloss.NewStyle().Foreground(style.EmptyColor)

	var sb strings.Builder

	// Filled portion — braille dots with optional accent banding
	if hasAccent && filledFull > 0 {
		// btop-style gradient: alternate bands of fill/accent every ~4 chars
		bandSize := 4
		for i := 0; i < filledFull; i++ {
			band := (i / bandSize) % 2
			if band == 0 {
				sb.WriteString(fillStyle.Render(brailleFull))
			} else {
				sb.WriteString(accentStyle.Render(brailleFull))
			}
		}
	} else if filledFull > 0 {
		sb.WriteString(fillStyle.Render(strings.Repeat(brailleFull, filledFull)))
	}

	// Half-fill character for sub-character precision
	emptyCount := w - filledFull
	if remainder >= 0.5 && emptyCount > 0 {
		sb.WriteString(fillStyle.Render(brailleHalf))
		emptyCount--
	}

	// Empty portion — dim braille dots showing the "track"
	if emptyCount > 0 {
		sb.WriteString(emptyStyle.Render(strings.Repeat(brailleEmpty, emptyCount)))
	}

	return sb.String()
}

// RenderBarSegmented draws a btop-style segmented bar with multiple colored sections.
// Each segment is a (fraction, color) pair. Fractions should sum to <= 1.0.
// Remaining space is filled with dim braille dots.
func RenderBarSegmented(segments []BarSegment, emptyColor color.Color, width int) string {
	if width < 1 {
		width = 10
	}

	emptyStyle := lipgloss.NewStyle().Foreground(emptyColor)
	var sb strings.Builder

	usedChars := 0
	for _, seg := range segments {
		chars := int(seg.Fraction * float64(width))
		if chars < 0 {
			chars = 0
		}
		if usedChars+chars > width {
			chars = width - usedChars
		}
		if chars > 0 {
			style := lipgloss.NewStyle().Foreground(seg.Color)
			sb.WriteString(style.Render(strings.Repeat(brailleFull, chars)))
			usedChars += chars
		}
	}

	// Fill remaining with dim braille track
	remaining := width - usedChars
	if remaining > 0 {
		sb.WriteString(emptyStyle.Render(strings.Repeat(brailleEmpty, remaining)))
	}

	return sb.String()
}

// BarSegment represents one colored section of a segmented bar.
type BarSegment struct {
	Fraction float64     // 0.0–1.0 portion of the bar
	Color    color.Color // fill color for this segment
}
