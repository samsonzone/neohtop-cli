//go:build darwin

package monitor

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <mach/mach.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <sys/sysctl.h>
#include <sys/time.h>
#include <ifaddrs.h>
#include <net/if.h>
#include <net/if_dl.h>
#include <net/route.h>
#include <stdlib.h>
#include <string.h>

// CGo cannot access C preprocessor macros directly, so we wrap them.

static mach_port_t get_host_self(void) {
	return mach_host_self();
}

static mach_port_t get_task_self(void) {
	return mach_task_self();
}

static mach_msg_type_number_t vm_stats64_count(void) {
	return HOST_VM_INFO64_COUNT;
}

// Wrapper to get loadavg without depending on unavailable symbols
int get_loadavg(double *avg) {
	return getloadavg(avg, 3);
}

// Get network interface stats via getifaddrs + if_data
// Returns 0 on success, -1 on failure
typedef struct {
	uint64_t rx_bytes;
	uint64_t tx_bytes;
} net_if_stats_t;

// Collect total RX/TX across all non-loopback interfaces
static int collect_net_stats(uint64_t *rx_total, uint64_t *tx_total) {
	struct ifaddrs *ifaddrs, *ifa;
	*rx_total = 0;
	*tx_total = 0;

	if (getifaddrs(&ifaddrs) != 0) return -1;

	for (ifa = ifaddrs; ifa != NULL; ifa = ifa->ifa_next) {
		if (ifa->ifa_addr == NULL) continue;
		if (ifa->ifa_addr->sa_family != AF_LINK) continue;
		if (strcmp(ifa->ifa_name, "lo0") == 0) continue;

		struct if_data *data = (struct if_data *)ifa->ifa_data;
		if (data != NULL) {
			*rx_total += data->ifi_ibytes;
			*tx_total += data->ifi_obytes;
		}
	}

	freeifaddrs(ifaddrs);
	return 0;
}
*/
import "C"

import (
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// collectSystemStats gathers system-wide stats on macOS
func (m *Monitor) collectSystemStats(elapsed float64) SystemStats {
	stats := SystemStats{}

	// CPU brand name (e.g. "Apple M1 Pro")
	if brand, err := unix.Sysctl("machdep.cpu.brand_string"); err == nil {
		stats.CPUBrand = brand
	}

	// Collect CPU usage
	stats.CPUUsage = m.collectCPUUsage(elapsed)

	// Collect memory stats
	stats.MemoryTotal, stats.MemoryUsed, stats.MemoryFree, stats.MemoryCached = m.collectMemoryStats()

	// Collect disk stats
	stats.DiskTotalBytes, stats.DiskUsedBytes, stats.DiskFreeBytes = m.collectDiskStats()

	// Collect network stats
	stats.NetworkRxBytes, stats.NetworkTxBytes = m.collectNetworkStats(elapsed)

	// Collect uptime
	stats.Uptime = m.collectUptime()

	// Collect load average
	stats.LoadAvg = m.collectLoadAvg()

	// Collect system info (hostname, OS, kernel)
	if hostname, err := unix.Sysctl("kern.hostname"); err == nil {
		stats.Hostname = hostname
	}
	if osVersion, err := unix.Sysctl("kern.osproductversion"); err == nil {
		stats.OSVersion = "macOS " + osVersion
	}
	if kernVersion, err := unix.Sysctl("kern.osrelease"); err == nil {
		stats.KernelVersion = "Darwin " + kernVersion
	}

	return stats
}

// collectCPUUsage returns per-core CPU usage percentages
func (m *Monitor) collectCPUUsage(elapsed float64) []float32 {
	var cpuCount C.natural_t
	var cpuInfo C.processor_info_array_t
	var infoCount C.mach_msg_type_number_t

	// Get number of CPUs
	ret := C.host_processor_info(C.get_host_self(), C.PROCESSOR_CPU_LOAD_INFO, &cpuCount, &cpuInfo, &infoCount)
	if ret != C.KERN_SUCCESS {
		return []float32{}
	}
	defer C.vm_deallocate(C.get_task_self(), C.vm_address_t(uintptr(unsafe.Pointer(cpuInfo))), C.vm_size_t(infoCount*4))

	cpuCountInt := int(cpuCount)

	// Parse CPU info into our slice
	var currentTimes []cpuTimes
	for i := 0; i < cpuCountInt; i++ {
		// processor_cpu_load_info_t layout: user, system, idle, nice (each 4 bytes)
		offset := i * 4
		base := uintptr(unsafe.Pointer(cpuInfo))
		userPtr := (*C.uint32_t)(unsafe.Pointer(base + uintptr(offset*4)))
		systemPtr := (*C.uint32_t)(unsafe.Pointer(base + uintptr((offset+1)*4)))
		idlePtr := (*C.uint32_t)(unsafe.Pointer(base + uintptr((offset+2)*4)))
		nicePtr := (*C.uint32_t)(unsafe.Pointer(base + uintptr((offset+3)*4)))

		currentTimes = append(currentTimes, cpuTimes{
			User:   uint64(*userPtr),
			System: uint64(*systemPtr),
			Idle:   uint64(*idlePtr),
			Nice:   uint64(*nicePtr),
		})
	}

	// Calculate deltas from previous state
	usage := make([]float32, cpuCountInt)
	for i := 0; i < cpuCountInt; i++ {
		if i < len(m.prevCPUTimes) {
			prev := m.prevCPUTimes[i]
			curr := currentTimes[i]

			userDelta := curr.User - prev.User
			sysDelta := curr.System - prev.System
			idleDelta := curr.Idle - prev.Idle
			niceDelta := curr.Nice - prev.Nice

			totalDelta := userDelta + sysDelta + idleDelta + niceDelta
			if totalDelta > 0 {
				usedDelta := userDelta + sysDelta
				usage[i] = float32(usedDelta) / float32(totalDelta) * 100.0
			}
		}
	}

	// Store current for next delta
	m.prevCPUTimes = currentTimes

	return usage
}

// collectMemoryStats returns memory totals in bytes
func (m *Monitor) collectMemoryStats() (total, used, free, cached uint64) {
	// Get total memory from sysctl
	total = m.sysctlUint64("hw.memsize")
	if total == 0 {
		return 0, 0, 0, 0
	}

	// Get VM info
	var vmStats C.vm_statistics64_data_t
	var count C.mach_msg_type_number_t = C.vm_stats64_count()

	ret := C.host_statistics64(C.get_host_self(), C.HOST_VM_INFO64, (*C.int32_t)(unsafe.Pointer(&vmStats)), &count)
	if ret != C.KERN_SUCCESS {
		return total, 0, 0, 0
	}

	pageSize := uint64(m.sysctlInt("hw.pagesize"))
	if pageSize == 0 {
		pageSize = 4096
	}

	// Extract memory values
	freePages := uint64(vmStats.free_count)
	inactivePages := uint64(vmStats.inactive_count)
	cachedPages := uint64(vmStats.purgeable_count)

	free = freePages * pageSize
	cached = cachedPages * pageSize
	used = total - free - (inactivePages * pageSize) - cached

	return total, used, free, cached
}

// collectDiskStats returns disk space info for root mount
func (m *Monitor) collectDiskStats() (total, used, free uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		return 0, 0, 0
	}

	blockSize := uint64(stat.Bsize)
	totalBlocks := uint64(stat.Blocks)
	freeBlocks := uint64(stat.Bfree)
	usedBlocks := totalBlocks - freeBlocks

	total = totalBlocks * blockSize
	free = freeBlocks * blockSize
	used = usedBlocks * blockSize

	return total, used, free
}

// collectNetworkStats returns network RX/TX rates (bytes per second)
func (m *Monitor) collectNetworkStats(elapsed float64) (rxBytes, txBytes uint64) {
	var currentRx, currentTx C.uint64_t
	if C.collect_net_stats(&currentRx, &currentTx) != 0 {
		return 0, 0
	}

	rx := uint64(currentRx)
	tx := uint64(currentTx)

	// Calculate rates from deltas
	var rxDelta, txDelta uint64
	if m.prevNet.RxBytes > 0 && rx >= m.prevNet.RxBytes {
		rxDelta = rx - m.prevNet.RxBytes
	}
	if m.prevNet.TxBytes > 0 && tx >= m.prevNet.TxBytes {
		txDelta = tx - m.prevNet.TxBytes
	}

	// Store for next delta
	m.prevNet.RxBytes = rx
	m.prevNet.TxBytes = tx

	// Return rates (bytes per second)
	if elapsed > 0 {
		rxBytes = uint64(float64(rxDelta) / elapsed)
		txBytes = uint64(float64(txDelta) / elapsed)
	}

	return rxBytes, txBytes
}

// collectUptime returns system uptime in seconds
func (m *Monitor) collectUptime() uint64 {
	// Get boot time from kern.boottime
	bootTime := m.getBootTime()
	if bootTime == 0 {
		return 0
	}

	return uint64(time.Now().Unix()) - bootTime
}

// getBootTime returns the Unix timestamp when the system booted
func (m *Monitor) getBootTime() uint64 {
	// Use sysctl kern.boottime to get boot time as timeval struct
	name := []C.int{C.CTL_KERN, C.KERN_BOOTTIME}
	var tv C.struct_timeval
	size := C.size_t(unsafe.Sizeof(tv))

	ret := C.sysctl((*C.int)(unsafe.Pointer(&name[0])), 2, unsafe.Pointer(&tv), &size, nil, 0)
	if ret != 0 {
		return 0
	}

	return uint64(tv.tv_sec)
}

// collectLoadAvg returns 1min, 5min, 15min load averages
func (m *Monitor) collectLoadAvg() [3]float64 {
	var avg [3]C.double
	if C.get_loadavg((*C.double)(unsafe.Pointer(&avg[0]))) == -1 {
		return [3]float64{0, 0, 0}
	}

	return [3]float64{
		float64(avg[0]),
		float64(avg[1]),
		float64(avg[2]),
	}
}

// sysctlUint64 reads a uint64 sysctl value
func (m *Monitor) sysctlUint64(name string) uint64 {
	val, err := unix.SysctlUint64(name)
	if err != nil {
		return 0
	}
	return val
}

// sysctl reads an integer sysctl value
func (m *Monitor) sysctlInt(name string) int {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return 0
	}

	// For common keys, use direct parsing
	mib, err := parseSysctlMIB(name)
	if err != nil || len(mib) == 0 {
		return 0
	}

	var val C.int
	size := C.size_t(unsafe.Sizeof(val))
	ret := C.sysctl((*C.int)(unsafe.Pointer(&mib[0])), C.uint(len(mib)), unsafe.Pointer(&val), &size, nil, 0)
	if ret != 0 {
		return 0
	}

	return int(val)
}

// parseSysctlMIB converts a sysctl name string to MIB array
func parseSysctlMIB(name string) ([]C.int, error) {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil, nil
	}

	// Common macOS sysctl conversions
	switch parts[0] {
	case "hw":
		switch parts[1] {
		case "memsize":
			return []C.int{C.CTL_HW, C.HW_MEMSIZE}, nil
		case "pagesize":
			return []C.int{C.CTL_HW, C.HW_PAGESIZE}, nil
		}
	case "kern":
		switch parts[1] {
		case "boottime":
			return []C.int{C.CTL_KERN, C.KERN_BOOTTIME}, nil
		}
	case "net":
		// Handle net.if.* interfaces
		if len(parts) >= 4 && parts[1] == "if" {
			// net.if.en0.ibytes -> CTL_NET, NET_RT_IFLIST2, ...
			// This is complex; use alternative approach
			return nil, nil
		}
	}

	return nil, nil
}
