//go:build darwin

package monitor

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <libproc.h>
#include <sys/sysctl.h>
#include <sys/proc_info.h>
#include <sys/resource.h>
#include <sys/time.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// Get process info by PID
int get_proc_info(pid_t pid, struct proc_taskinfo *info) {
	return proc_pidinfo(pid, PROC_PIDTASKINFO, 0, info, sizeof(struct proc_taskinfo));
}

// Get process name
int get_proc_name(pid_t pid, char *name, size_t namesize) {
	return proc_name(pid, name, (uint32_t)namesize);
}

// Get process path
int get_proc_path(pid_t pid, char *path, size_t pathsize) {
	return proc_pidpath(pid, path, (uint32_t)pathsize);
}

// Get rusage info for disk I/O
int get_proc_rusage(pid_t pid, struct rusage_info_v4 *info) {
	return proc_pid_rusage(pid, RUSAGE_INFO_V4, (rusage_info_t *)info);
}

// Helper to list all process IDs
int list_pids(pid_t *pidlist, int max) {
	return proc_listallpids(pidlist, max * sizeof(pid_t));
}

// p_starttime is a macro (#define p_starttime p_un.__p_starttime),
// so CGo cannot access it directly. Wrap it.
static struct timeval get_proc_starttime(struct kinfo_proc *kp) {
	return kp->kp_proc.p_starttime;
}
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os/user"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// userCache caches uid -> username mappings
var userCache = make(map[uint32]string)

// collectProcesses returns all running processes with stats
func (m *Monitor) collectProcesses(elapsed float64) []ProcessInfo {
	// Get all process IDs
	maxPids := 8192
	pidList := make([]C.pid_t, maxPids)

	// proc_listallpids returns the number of PIDs (not bytes)
	numPids := int(C.list_pids((*C.pid_t)(unsafe.Pointer(&pidList[0])), C.int(maxPids)))
	if numPids <= 0 {
		return []ProcessInfo{}
	}
	pidList = pidList[:numPids]

	var processes []ProcessInfo

	for _, pid := range pidList {
		pidInt := uint32(pid)

		// Skip kernel_task (pid 0)
		if pidInt == 0 {
			continue
		}

		proc := m.getProcessInfo(pidInt, elapsed)
		if proc != nil {
			processes = append(processes, *proc)
		}
	}

	return processes
}

// getProcessInfo retrieves detailed info for a single process.
// Uses kinfo_proc (sysctl) as the base — works for all visible processes.
// Layers on proc_taskinfo for CPU/memory/threads when permissions allow.
func (m *Monitor) getProcessInfo(pid uint32, elapsed float64) *ProcessInfo {
	cPid := C.pid_t(pid)

	// --- Base info from kinfo_proc (works for all processes) ---
	ppid, uid, status, startTime, commName := m.getProcStatus(pid)

	userName := m.getUserName(uid)

	// Get process name — try proc_name first, fall back to kinfo_proc p_comm
	nameBytes := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	nameLen := C.get_proc_name(cPid, (*C.char)(unsafe.Pointer(&nameBytes[0])), C.size_t(len(nameBytes)))
	var name string
	if nameLen > 0 {
		name = string(nameBytes[:nameLen])
	} else {
		name = commName // fallback: always available from kinfo_proc
	}

	// Get command/path — fall back to name if path unavailable
	pathBytes := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	pathLen := C.get_proc_path(cPid, (*C.char)(unsafe.Pointer(&pathBytes[0])), C.size_t(len(pathBytes)))
	var command string
	if pathLen > 0 {
		command = string(pathBytes[:pathLen])
	} else {
		command = name
	}

	// --- Optional: task info for CPU, memory, threads (needs permissions) ---
	var cpuUsage float32
	var memoryUsage, virtualMemory uint64
	var threadCount uint32
	var diskRead, diskWrite uint64

	var taskInfo C.struct_proc_taskinfo
	hasTaskInfo := C.get_proc_info(cPid, &taskInfo) > 0

	if hasTaskInfo {
		threadCount = uint32(taskInfo.pti_threadnum)
		virtualMemory = uint64(taskInfo.pti_virtual_size)
		memoryUsage = uint64(taskInfo.pti_resident_size)

		// Calculate CPU usage from deltas
		userTime := uint64(taskInfo.pti_total_user)
		systemTime := uint64(taskInfo.pti_total_system)

		prevTimes := m.prevProcTimes[pid]
		userDelta := userTime - prevTimes.User
		sysDelta := systemTime - prevTimes.System

		m.prevProcTimes[pid] = procTimes{User: userTime, System: systemTime}

		if elapsed > 0 {
			totalNsDelta := float64(userDelta + sysDelta)
			elapsedNs := elapsed * 1e9
			cpuUsage = float32((totalNsDelta / elapsedNs) * 100.0)
		}

		// Disk I/O rates
		diskRead, diskWrite = m.getProcessDiskIO(pid, elapsed)
	}

	// Calculate runtime
	var runTime uint64
	if startTime > 0 {
		nowSec := uint64(time.Now().Unix())
		if nowSec > startTime {
			runTime = nowSec - startTime
		}
	}

	proc := &ProcessInfo{
		PID:           pid,
		PPID:          ppid,
		Name:          name,
		CPUUsage:      cpuUsage,
		MemoryUsage:   memoryUsage,
		Status:        status,
		User:          userName,
		Command:       command,
		Threads:       &threadCount,
		Root:          "",
		VirtualMemory: virtualMemory,
		StartTime:     startTime,
		RunTime:       runTime,
		DiskRead:      diskRead,
		DiskWrite:     diskWrite,
		SessionID:     nil,
	}

	return proc
}

// getProcStatus retrieves PPID, UID, status, start time, and comm name for a process.
// The comm name (from kp_proc.p_comm) serves as a fallback when proc_name() fails.
func (m *Monitor) getProcStatus(pid uint32) (ppid uint32, uid uint32, status string, startTime uint64, comm string) {
	// Use sysctl to get process info
	mib := []C.int{C.CTL_KERN, C.KERN_PROC, C.KERN_PROC_PID, C.int(pid)}

	var kp C.struct_kinfo_proc
	size := C.size_t(unsafe.Sizeof(kp))

	ret := C.sysctl((*C.int)(unsafe.Pointer(&mib[0])), 4, unsafe.Pointer(&kp), &size, nil, 0)
	if ret != 0 {
		return 0, 0, "Unknown", 0, ""
	}

	ppid = uint32(kp.kp_eproc.e_ppid)
	uid = uint32(kp.kp_eproc.e_ucred.cr_uid)

	// Map process state to status string
	state := uint8(kp.kp_proc.p_stat)
	status = mapProcessState(state)

	// Start time — p_starttime is a macro, so use C helper
	startTime = uint64(C.get_proc_starttime(&kp).tv_sec)

	// Comm name — always available, up to 16 chars
	comm = C.GoString(&kp.kp_proc.p_comm[0])

	return ppid, uid, status, startTime, comm
}

// mapProcessState converts macOS process state to human-readable status
func mapProcessState(state uint8) string {
	// macOS process states from sys/proc.h
	switch state {
	case 1: // SIDL
		return "Idle"
	case 2: // SRUN
		return "Running"
	case 3: // SSLEEP
		return "Sleeping"
	case 4: // SSTOP
		return "Stopped"
	case 5: // SZOMB
		return "Zombie"
	case 6: // SWAIT
		return "Waiting"
	case 7: // SLOCK
		return "Locked"
	default:
		return "Unknown"
	}
}

// getUserName gets the username for a given UID
func (m *Monitor) getUserName(uid uint32) string {
	// Check cache first
	if name, ok := userCache[uid]; ok {
		return name
	}

	// Look up user
	uidStr := fmt.Sprintf("%d", uid)
	u, err := user.LookupId(uidStr)
	if err != nil {
		name := "#" + uidStr
		userCache[uid] = name
		return name
	}

	userCache[uid] = u.Username
	return u.Username
}

// getProcessDiskIO returns disk read/write rates in bytes/sec for a process
func (m *Monitor) getProcessDiskIO(pid uint32, elapsed float64) (readRate, writeRate uint64) {
	cPid := C.pid_t(pid)

	// Try to get rusage info (includes disk I/O counters)
	var rusageInfo C.struct_rusage_info_v4
	if C.get_proc_rusage(cPid, &rusageInfo) != 0 {
		return 0, 0
	}

	currentRead := uint64(rusageInfo.ri_diskio_bytesread)
	currentWrite := uint64(rusageInfo.ri_diskio_byteswritten)

	// Get previous values from cache
	prevDisk := m.prevProcDisk[pid]

	// Calculate deltas
	readDelta := currentRead - prevDisk.Read
	writeDelta := currentWrite - prevDisk.Write

	// Store current for next delta
	m.prevProcDisk[pid] = procDiskCounters{Read: currentRead, Write: currentWrite}

	// Calculate rates (bytes per second)
	if elapsed > 0 {
		readRate = uint64(float64(readDelta) / elapsed)
		writeRate = uint64(float64(writeDelta) / elapsed)
	}

	return readRate, writeRate
}

// getProcessDetail fetches expensive per-process data on-demand
func (m *Monitor) getProcessDetail(pid uint32) *ProcessDetail {
	// Get virtual memory (already in task info)
	var taskInfo C.struct_proc_taskinfo
	cPid := C.pid_t(pid)
	if C.get_proc_info(cPid, &taskInfo) <= 0 {
		return nil
	}

	// Try to get root directory
	root := m.getProcessRoot(pid)

	// Get environ from process arguments
	environ := m.getProcessEnviron(pid)

	return &ProcessDetail{
		PID:           pid,
		Environ:       environ,
		Root:          root,
		VirtualMemory: uint64(taskInfo.pti_virtual_size),
	}
}

// getProcessRoot gets the working directory of a process
func (m *Monitor) getProcessRoot(pid uint32) string {
	// Use proc_pidinfo with PROC_PIDVNODEPATHINFO
	cPid := C.pid_t(pid)

	var vnodeInfo C.struct_proc_vnodepathinfo
	if C.proc_pidinfo(cPid, C.PROC_PIDVNODEPATHINFO, 0, unsafe.Pointer(&vnodeInfo), C.int(unsafe.Sizeof(vnodeInfo))) <= 0 {
		return ""
	}

	// Extract cwd from the structure
	cwd := C.GoString((*C.char)(unsafe.Pointer(&vnodeInfo.pvi_cdir.vip_path[0])))
	return cwd
}

// getProcessEnviron retrieves environment variables for a process
func (m *Monitor) getProcessEnviron(pid uint32) []string {
	// Use sysctl KERN_PROCARGS2 to get arguments and environment
	name := []C.int{C.CTL_KERN, C.KERN_PROCARGS2, C.int(pid)}

	// First call to get size
	size := C.size_t(0)
	ret := C.sysctl((*C.int)(unsafe.Pointer(&name[0])), 3, nil, &size, nil, 0)
	if ret != 0 || size == 0 {
		return []string{}
	}

	// Cap size to prevent excessive allocations
	if size > 1024*1024 { // 1MB max
		size = 1024 * 1024
	}

	args := make([]byte, size)
	ret = C.sysctl((*C.int)(unsafe.Pointer(&name[0])), 3, unsafe.Pointer(&args[0]), &size, nil, 0)
	if ret != 0 {
		return []string{}
	}

	return m.parseEnviron(args[:size])
}

// parseEnviron extracts environment variables from KERN_PROCARGS2 output
func (m *Monitor) parseEnviron(data []byte) []string {
	if len(data) < 4 {
		return []string{}
	}

	// First 4 bytes is argc
	argc := int32(binary.LittleEndian.Uint32(data[:4]))
	pos := 4

	// Skip executable path + argc arguments (null-terminated strings)
	// First string after argc is the executable path, then argc argv strings
	skipped := int32(0)
	for pos < len(data) && skipped <= argc {
		// Find next null
		nullIdx := bytes.IndexByte(data[pos:], 0)
		if nullIdx < 0 {
			return []string{}
		}
		pos += nullIdx + 1
		skipped++
	}

	// Skip trailing nulls between argv and environ
	for pos < len(data) && data[pos] == 0 {
		pos++
	}

	// Parse environment variables (null-terminated strings)
	var environ []string
	remaining := data[pos:]

	for len(remaining) > 0 {
		nullIdx := bytes.IndexByte(remaining, 0)
		if nullIdx < 0 {
			break
		}
		if nullIdx > 0 {
			envStr := string(remaining[:nullIdx])
			if strings.Contains(envStr, "=") {
				environ = append(environ, envStr)
			}
		}
		remaining = remaining[nullIdx+1:]
	}

	return environ
}

// killProcess sends SIGTERM to a process
func (m *Monitor) killProcess(pid uint32) bool {
	err := syscall.Kill(int(pid), syscall.SIGTERM)
	return err == nil
}
