use crate::monitoring::{ProcessMonitor, SystemMonitor};
use std::sync::Mutex;
use sysinfo::{Disks, Networks, System};

/// Application state for the monitoring core
pub struct AppState {
    pub sys: Mutex<System>,
    pub networks: Mutex<Networks>,
    pub disks: Mutex<Disks>,
    pub process_monitor: Mutex<ProcessMonitor>,
    pub system_monitor: Mutex<SystemMonitor>,
}

impl AppState {
    pub fn new() -> Self {
        let sys = System::new_all();
        let disks = Disks::new_with_refreshed_list();
        let networks = Networks::new_with_refreshed_list();

        Self {
            process_monitor: Mutex::new(ProcessMonitor::new()),
            system_monitor: Mutex::new(SystemMonitor::new(&networks)),
            sys: Mutex::new(sys),
            disks: Mutex::new(disks),
            networks: Mutex::new(networks),
        }
    }
}
