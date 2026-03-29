use super::{ProcessData, ProcessDetail, ProcessInfo, ProcessStaticInfo};
use std::collections::HashMap;
use std::ffi::OsString;
use std::time::{SystemTime, UNIX_EPOCH};
use sysinfo::ProcessStatus;

fn os_string_vec_to_string_vec(v: &[OsString]) -> Vec<String> {
    v.iter()
        .map(|c| c.to_string_lossy().into_owned())
        .collect()
}

/// Monitors and manages system processes
#[derive(Debug)]
pub struct ProcessMonitor {
    process_cache: HashMap<u32, ProcessStaticInfo>,
    collect_count: u64,
}

impl ProcessMonitor {
    pub fn new() -> Self {
        Self {
            process_cache: HashMap::new(),
            collect_count: 0,
        }
    }

    /// Collects information about all running processes.
    /// Also prunes the cache of dead PIDs periodically.
    pub fn collect_processes(&mut self, sys: &sysinfo::System) -> Result<Vec<ProcessInfo>, String> {
        let current_time = Self::get_current_time()?;
        let processes_data = self.collect_process_data(sys, current_time);

        // Prune dead PIDs every 50 calls (~75s at default refresh rate)
        self.collect_count += 1;
        if self.collect_count % 50 == 0 {
            let live_pids: std::collections::HashSet<u32> =
                processes_data.iter().map(|p| p.pid).collect();
            self.process_cache.retain(|pid, _| live_pids.contains(pid));
        }

        Ok(self.build_process_info(processes_data))
    }

    /// Attempts to kill a process by PID
    pub fn kill_process(sys: &sysinfo::System, pid: u32) -> bool {
        sys.process(sysinfo::Pid::from(pid as usize))
            .map(|process| process.kill())
            .unwrap_or(false)
    }

    fn get_current_time() -> Result<u64, String> {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs())
            .map_err(|e| format!("Failed to get system time: {}", e))
    }

    fn collect_process_data(&self, sys: &sysinfo::System, current_time: u64) -> Vec<ProcessData> {
        sys.processes()
            .iter()
            .map(|(pid, process)| {
                let start_time = process.start_time();
                ProcessData {
                    pid: pid.as_u32(),
                    name: process.name().to_string_lossy().into_owned(),
                    cmd: os_string_vec_to_string_vec(&process.cmd()),
                    user_id: process.user_id().map(|uid| uid.to_string()),
                    cpu_usage: process.cpu_usage(),
                    memory: process.memory(),
                    status: process.status(),
                    ppid: process.parent().map(|p| p.as_u32()),
                    root: process
                        .root()
                        .map(|p| p.to_string_lossy().into_owned())
                        .unwrap_or_default(),
                    virtual_memory: process.virtual_memory(),
                    start_time,
                    run_time: if start_time > 0 {
                        current_time.saturating_sub(start_time)
                    } else {
                        0
                    },
                    disk_usage: process.disk_usage(),
                    session_id: process.session_id().map(|id| id.as_u32()),
                }
            })
            .collect()
    }

    fn build_process_info(&mut self, processes: Vec<ProcessData>) -> Vec<ProcessInfo> {
        processes
            .into_iter()
            .map(|data| {
                let cached_info =
                    self.process_cache
                        .entry(data.pid)
                        .or_insert_with(|| ProcessStaticInfo {
                            name: data.name.clone(),
                            command: data.cmd.join(" "),
                            user: data.user_id.unwrap_or_else(|| "-".to_string()),
                        });

                ProcessInfo {
                    pid: data.pid,
                    ppid: data.ppid.unwrap_or(0),
                    name: cached_info.name.clone(),
                    cpu_usage: data.cpu_usage,
                    memory_usage: data.memory,
                    status: Self::format_status(data.status),
                    user: cached_info.user.clone(),
                    command: cached_info.command.clone(),
                    threads: None,
                    root: data.root,
                    virtual_memory: data.virtual_memory,
                    start_time: data.start_time,
                    run_time: data.run_time,
                    disk_usage: (data.disk_usage.read_bytes, data.disk_usage.written_bytes),
                    session_id: data.session_id,
                }
            })
            .collect()
    }

    /// Get detailed info for a single process (includes environ).
    /// Called on-demand when user opens process details overlay.
    pub fn get_process_detail(&self, sys: &sysinfo::System, pid: u32) -> Option<ProcessDetail> {
        let process = sys.process(sysinfo::Pid::from(pid as usize))?;
        Some(ProcessDetail {
            pid,
            environ: os_string_vec_to_string_vec(&process.environ()),
            root: process
                .root()
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_default(),
            virtual_memory: process.virtual_memory(),
        })
    }

    pub fn format_status(status: ProcessStatus) -> String {
        match status {
            ProcessStatus::Run => "Running",
            ProcessStatus::Sleep => "Sleeping",
            ProcessStatus::Idle => "Idle",
            _ => "Unknown",
        }
        .to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use sysinfo::System;

    #[test]
    fn test_process_monitor_creation() {
        let monitor = ProcessMonitor::new();
        assert!(monitor.process_cache.is_empty());
    }

    #[test]
    fn test_process_collection() {
        let mut monitor = ProcessMonitor::new();
        let mut sys = System::new();
        sys.refresh_all();
        let result = monitor.collect_processes(&sys);
        assert!(result.is_ok());
    }
}
