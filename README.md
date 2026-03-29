# NeoHtop CLI

A terminal-based system monitor with full feature parity with [NeoHtop](https://github.com/abdenasser/NeoHtop). Built with a **Rust** monitoring backend and a **Go** TUI frontend using the [Charm](https://charm.sh) ecosystem (Bubble Tea + Lip Gloss).

## Features

- Real-time process monitoring with per-core CPU bars, memory, disk, and network stats
- Sortable, searchable process table with regex support
- Process details overlay with child processes and environment variables
- Kill processes with confirmation
- Pin important processes to the top
- Catppuccin Mocha (dark) and Latte (light) themes
- Configurable refresh rate, columns, and pagination
- Single binary — Rust core compiled as a static library linked into the Go binary

## Architecture

```
┌──────────────────────────────────────┐
│          Go TUI (Bubble Tea)         │
│  Stats Bar │ Process Table │ Overlays│
├──────────────────────────────────────┤
│           CGo / FFI Bridge           │
├──────────────────────────────────────┤
│     Rust Core (libneohtop_core.a)    │
│   Process Monitor │ System Monitor   │
└──────────────────────────────────────┘
```

## Prerequisites

- **Rust** toolchain — [rustup.rs](https://rustup.rs)
- **Go** 1.21+ — [go.dev](https://go.dev/dl)
- **C compiler** (gcc or clang) — required by CGo
- **Make**

## Build

```bash
# Build everything (Rust core + Go CLI)
make build

# The binary is at ./neohtop-cli
./neohtop-cli
```

### Development

```bash
# Faster iteration with debug Rust build
make dev

# Run tests
make test
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `?` | Help overlay |
| `/` | Search mode |
| `f` | Freeze/unfreeze updates |
| `↑/↓` / `j/k` | Navigate processes |
| `PgUp/PgDn` | Page up/down |
| `←/→` / `h/l` | Previous/next page |
| `Enter` | Process details |
| `x` / `Delete` | Kill process |
| `p` | Pin/unpin process |
| `1-9` | Sort by column |
| `s` | Cycle sort field |
| `r` | Reverse sort |
| `t` | Cycle theme |
| `+/-` | Adjust refresh rate |

## Search Syntax

- Plain text: filters by process name, command, or PID
- Comma-separated: `chrome, firefox` matches either
- Regex: `^sys.*` matches names starting with "sys"
- PID: `1234` matches the exact PID

## Configuration

Settings are saved to `~/.config/neohtop-cli/config.json`:

```json
{
  "columns": ["pid", "name", "cpu", "memory", "status", "user", "command"],
  "items_per_page": 50,
  "refresh_rate_ms": 1500,
  "theme": "mocha"
}
```

## Project Structure

```
NeoHtopCLI/
├── core/                  # Rust static library
│   ├── src/
│   │   ├── lib.rs         # Library root
│   │   ├── ffi.rs         # C-compatible FFI exports
│   │   ├── state.rs       # AppState
│   │   └── monitoring/    # Process & system monitors
│   └── Cargo.toml
├── cli/                   # Go TUI application
│   ├── main.go            # Entry point
│   ├── bridge/            # CGo FFI bindings
│   ├── model/             # Bubble Tea model & types
│   ├── view/              # UI components
│   ├── filter/            # Filter & sort logic
│   ├── theme/             # Catppuccin themes
│   └── config/            # User settings persistence
├── Makefile
└── README.md
```

## License

MIT
