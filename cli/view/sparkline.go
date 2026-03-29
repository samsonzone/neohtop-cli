package view

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Sparkline renders a horizontal sparkline graph using braille dot-matrix characters.
// Each character position represents one data point; the braille dot pattern encodes
// the value (0–100 mapped to 5 vertical levels using the 2×4 braille grid).
//
// This gives the characteristic btop "dotted" look for CPU core and network graphs.

// Braille vertical levels (both columns, filling from bottom up):
//
//	Level 0: ⠀ (U+2800) — empty
//	Level 1: ⣀ (U+28C0) — bottom row (dots 7,8)
//	Level 2: ⣤ (U+28E4) — bottom 2 rows (dots 3,6,7,8)
//	Level 3: ⣶ (U+28F6) — bottom 3 rows (dots 2,3,5,6,7,8)
//	Level 4: ⣿ (U+28FF) — all 4 rows (all dots)
var brailleLevels = [5]string{"⠀", "⣀", "⣤", "⣶", "⣿"}

// brailleTrack is the dim background character for empty sparkline positions
const brailleTrack = "⠤" // U+2824 — subtle bottom dots as track marker

// SparklineHistory is a fixed-size ring buffer that stores float values
// and renders them as a horizontal sparkline.
type SparklineHistory struct {
	data []float64
	size int
	pos  int
	full bool
}

// NewSparklineHistory creates a history buffer of the given width (number of columns).
func NewSparklineHistory(width int) *SparklineHistory {
	if width < 1 {
		width = 1
	}
	return &SparklineHistory{
		data: make([]float64, width),
		size: width,
	}
}

// Push adds a new value (0.0–100.0) to the history.
func (s *SparklineHistory) Push(value float64) {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	s.data[s.pos] = value
	s.pos = (s.pos + 1) % s.size
	if !s.full && s.pos == 0 {
		s.full = true
	}
}

// Len returns how many values have been pushed so far (up to size).
func (s *SparklineHistory) Len() int {
	if s.full {
		return s.size
	}
	return s.pos
}

// Values returns the historical data in chronological order (oldest first).
func (s *SparklineHistory) Values() []float64 {
	n := s.Len()
	result := make([]float64, n)
	if s.full {
		copy(result, s.data[s.pos:])
		copy(result[s.size-s.pos:], s.data[:s.pos])
	} else {
		copy(result, s.data[:s.pos])
	}
	return result
}

// Resize changes the width of the sparkline, preserving as much history as possible.
func (s *SparklineHistory) Resize(newWidth int) {
	if newWidth < 1 {
		newWidth = 1
	}
	if newWidth == s.size {
		return
	}

	old := s.Values()
	s.data = make([]float64, newWidth)
	s.size = newWidth
	s.pos = 0
	s.full = false

	// Copy the most recent values that fit
	start := 0
	if len(old) > newWidth {
		start = len(old) - newWidth
	}
	for _, v := range old[start:] {
		s.Push(v)
	}
}

// valueToBraille maps a 0–100 value to a braille level (0–4).
func valueToBraille(v float64) string {
	level := int(v / 100.0 * 4.0)
	if level < 0 {
		level = 0
	}
	if level > 4 {
		level = 4
	}
	if v > 0 && level == 0 {
		level = 1 // show at least ⣀ for non-zero values
	}
	return brailleLevels[level]
}

// RenderBlock renders the sparkline as braille dot characters.
// Width is the number of characters to output. If history is shorter
// than width, the left side shows a dim track.
func (s *SparklineHistory) RenderBlock(width int, fg color.Color) string {
	vals := s.Values()

	var sb strings.Builder
	style := lipgloss.NewStyle().Foreground(fg)

	// Pad left with dim track if we have fewer values than width
	pad := width - len(vals)
	if pad < 0 {
		vals = vals[len(vals)-width:]
		pad = 0
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	for i := 0; i < pad; i++ {
		sb.WriteString(dimStyle.Render(brailleTrack))
	}

	for _, v := range vals {
		sb.WriteString(style.Render(valueToBraille(v)))
	}

	return sb.String()
}

// RenderBlockGradient renders the sparkline with color that changes based on value.
// Low values use lowColor, mid values use midColor, high values use highColor.
func (s *SparklineHistory) RenderBlockGradient(width int, lowColor, midColor, highColor, critColor color.Color) string {
	vals := s.Values()

	var sb strings.Builder

	pad := width - len(vals)
	if pad < 0 {
		vals = vals[len(vals)-width:]
		pad = 0
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	for i := 0; i < pad; i++ {
		sb.WriteString(dimStyle.Render(brailleTrack))
	}

	for _, v := range vals {
		var fg color.Color
		switch {
		case v > 90:
			fg = critColor
		case v > 75:
			fg = highColor
		case v > 50:
			fg = midColor
		default:
			fg = lowColor
		}

		style := lipgloss.NewStyle().Foreground(fg)
		sb.WriteString(style.Render(valueToBraille(v)))
	}

	return sb.String()
}

// RenderMini renders a compact sparkline (just braille dots, no label) at the given width.
func RenderMini(values []float64, width int, fg color.Color) string {
	var sb strings.Builder
	style := lipgloss.NewStyle().Foreground(fg)

	start := 0
	if len(values) > width {
		start = len(values) - width
	}

	pad := width - (len(values) - start)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	for i := 0; i < pad; i++ {
		sb.WriteString(dimStyle.Render(brailleTrack))
	}

	for _, v := range values[start:] {
		sb.WriteString(style.Render(valueToBraille(v)))
	}

	return sb.String()
}
