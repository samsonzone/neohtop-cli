package monitor

// ProcessInfo represents a running process (lightweight — no environ in hot path)
type ProcessInfo struct {
	PID           uint32
	PPID          uint32
	Name          string
	CPUUsage      float32 // percentage, 0-100 per core
	MemoryUsage   uint64  // RSS in bytes
	Status        string  // "Running", "Sleeping", "Idle", "Stopped", "Zombie", "Unknown"
	User          string
	Command       string
	Threads       *uint32
	Root          string
	VirtualMemory uint64
	StartTime     uint64 // unix timestamp
	RunTime       uint64 // seconds
	DiskRead      uint64 // bytes/s since last refresh
	DiskWrite     uint64 // bytes/s since last refresh
	SessionID     *uint32
}

// ProcessDetail contains expensive fields fetched on-demand
type ProcessDetail struct {
	PID           uint32
	Environ       []string
	Root          string
	VirtualMemory uint64
}

// SystemStats holds system-wide monitoring data
type SystemStats struct {
	CPUBrand       string    // e.g. "Apple M1 Pro"
	CPUUsage       []float32 // per-core percentages
	MemoryTotal    uint64
	MemoryUsed     uint64
	MemoryFree     uint64
	MemoryCached   uint64
	Uptime         uint64
	LoadAvg        [3]float64
	NetworkRxBytes uint64 // bytes/s
	NetworkTxBytes uint64 // bytes/s
	DiskTotalBytes uint64
	DiskUsedBytes  uint64
	DiskFreeBytes  uint64
	Hostname       string
	OSVersion      string // e.g. "macOS 15.2" or "Linux 6.5"
	KernelVersion  string // e.g. "Darwin 24.2.0" or "6.5.0-generic"
	ProcessCount   int
}

// cpuTimes stores raw CPU tick counters for delta calculation
type cpuTimes struct {
	User   uint64
	System uint64
	Idle   uint64
	Nice   uint64
	IOWait uint64 // Linux only
}

// procTimes stores per-process CPU times for delta calculation
type procTimes struct {
	User   uint64 // user time in ticks/nanoseconds
	System uint64 // system time in ticks/nanoseconds
}

// netCounters stores raw network byte counters for rate calculation
type netCounters struct {
	RxBytes uint64
	TxBytes uint64
}

// procDiskCounters stores per-process cumulative disk I/O
type procDiskCounters struct {
	Read  uint64
	Write uint64
}
