//go:build windows

package monitor

// collectProcesses enumerates running processes on Windows.
// TODO: Implement using CreateToolhelp32Snapshot / Process32First / Process32Next.
func (m *Monitor) collectProcesses(elapsed float64) []ProcessInfo {
	return nil
}

// getProcessDetail returns detailed info for a single process on Windows.
// TODO: Implement using NtQueryInformationProcess.
func (m *Monitor) getProcessDetail(pid uint32) *ProcessDetail {
	return nil
}

// killProcess terminates a process on Windows.
// TODO: Implement using OpenProcess + TerminateProcess.
func (m *Monitor) killProcess(pid uint32) bool {
	return false
}
