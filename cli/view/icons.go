package view

// Unicode icons that render reliably in virtually all modern terminals and fonts.
// These use standard Unicode blocks (Miscellaneous Symbols, Arrows, Box Drawing,
// Geometric Shapes, Dingbats) — no Nerd Fonts required.

// ── Navigation & Actions ─────────────────────────────────────────────
const (
	IconSearch   = "/"   // search slash
	IconFilter   = "≡"   // filter (triple bar)
	IconColumns  = "☰"   // hamburger menu / columns
	IconPause    = "‖"   // pause (double bar)
	IconPlay     = "▶"   // play / live
	IconFrozen   = "‖"   // frozen state
	IconTheme    = "◑"   // half circle / theme toggle
	IconHelp     = "?"   // help
	IconRefresh  = "⟳"   // refresh cycle
	IconChevronL = "‹"   // left chevron
	IconChevronR = "›"   // right chevron
	IconClose    = "✕"   // close / cancel
	IconCheck    = "✓"   // checkmark
	IconSortAsc  = "▲"   // sort ascending
	IconSortDesc = "▼"   // sort descending
	IconSortNone = "⇅"   // unsorted
)

// ── Process Actions ──────────────────────────────────────────────────
const (
	IconPin     = "\U000F0403" // 󰐃 nf-md-pin (filled)
	IconPinOff  = "\U000F0404" // 󰐄 nf-md-pin_off
	IconInfo    = "ℹ"   // info
	IconKill    = "✕"   // kill / close
	IconWarning = "⚠"   // warning
	IconProcess = "⚙"   // process / cog
	IconTerminal = "❯"  // terminal prompt
)

// ── Stats Bar Panels ─────────────────────────────────────────────────
const (
	IconCPU      = "⣿"  // CPU (braille full block — looks like a chip)
	IconMemory   = "▦"   // memory (grid)
	IconDisk     = "◉"   // disk
	IconNetwork  = "⇄"   // network (bidirectional arrows)
	IconSystem   = "◆"   // system
	IconUptime   = "◔"   // timer / uptime (quarter circle)
	IconDownload = "↓"   // download
	IconUpload   = "↑"   // upload
)

// ── Misc UI ──────────────────────────────────────────────────────────
const (
	IconKeyboard = "⌨"   // keyboard
	IconLock     = "⊘"   // locked / required
	IconEye      = "◉"   // visible
	IconEyeOff   = "○"   // hidden
	IconArrowR   = "▸"   // right arrow (selection marker)
	IconDot      = "·"   // middle dot
	IconCursor   = "█"   // block cursor for search input
	IconBullet   = "●"   // bullet for active items
	IconSep      = "│"   // vertical separator
)
