package monitor

import (
	"time"
)

// Monitor collects system and process data directly from OS interfaces.
// No FFI, no JSON serialization — just direct syscalls like btop.
type Monitor struct {
	// Delta state for CPU percentage calculation
	prevCPUTimes  []cpuTimes
	prevProcTimes map[uint32]procTimes
	prevProcDisk  map[uint32]procDiskCounters

	// Delta state for network rate calculation
	prevNet         netCounters
	prevRefreshTime time.Time

	// Results from last Refresh()
	processes   []ProcessInfo
	systemStats SystemStats
}

// New creates a new Monitor with initial state.
func New() *Monitor {
	m := &Monitor{
		prevProcTimes: make(map[uint32]procTimes),
		prevProcDisk:  make(map[uint32]procDiskCounters),
	}
	// Do an initial refresh to seed delta state (CPU values won't be accurate yet)
	m.Refresh()
	return m
}

// Refresh collects all system and process data from OS interfaces.
// Must be called periodically (e.g. every 1.5s).
// After calling, use Processes() and Stats() to read results.
func (m *Monitor) Refresh() {
	now := time.Now()
	elapsed := now.Sub(m.prevRefreshTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1.0
	}

	m.systemStats = m.collectSystemStats(elapsed)
	m.processes = m.collectProcesses(elapsed)
	m.prevRefreshTime = now

	// Prune dead PIDs from delta maps every ~50 refreshes
	if len(m.prevProcTimes) > len(m.processes)*2 {
		m.pruneDeadPIDs()
	}
}

// Processes returns the process list from the last Refresh().
func (m *Monitor) Processes() []ProcessInfo {
	return m.processes
}

// Stats returns system stats from the last Refresh().
func (m *Monitor) Stats() SystemStats {
	return m.systemStats
}

// GetProcessDetail fetches expensive per-process data on-demand.
// Called when user opens the process details overlay.
func (m *Monitor) GetProcessDetail(pid uint32) *ProcessDetail {
	return m.getProcessDetail(pid)
}

// KillProcess sends SIGTERM to a process.
func (m *Monitor) KillProcess(pid uint32) bool {
	return m.killProcess(pid)
}

// Destroy cleans up any resources.
func (m *Monitor) Destroy() {
	// No-op for now — pure Go, no external resources to free
}

// pruneDeadPIDs removes entries for processes that no longer exist
func (m *Monitor) pruneDeadPIDs() {
	live := make(map[uint32]bool, len(m.processes))
	for _, p := range m.processes {
		live[p.PID] = true
	}
	for pid := range m.prevProcTimes {
		if !live[pid] {
			delete(m.prevProcTimes, pid)
		}
	}
	for pid := range m.prevProcDisk {
		if !live[pid] {
			delete(m.prevProcDisk, pid)
		}
	}
}
