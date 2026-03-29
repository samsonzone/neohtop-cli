//go:build windows

package monitor

// collectSystemStats gathers CPU, memory, disk, and network stats on Windows.
// TODO: Implement using golang.org/x/sys/windows API calls.
func (m *Monitor) collectSystemStats(elapsed float64) SystemStats {
	return SystemStats{}
}
