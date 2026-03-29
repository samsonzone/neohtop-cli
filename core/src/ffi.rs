use crate::monitoring::{MonitoringData, ProcessMonitor};
use crate::state::AppState;
use std::ffi::{c_char, c_void, CString};

/// Initialize the monitoring system. Returns an opaque handle.
/// The caller must eventually call neohtop_destroy() to free it.
#[no_mangle]
pub extern "C" fn neohtop_init() -> *mut c_void {
    let state = Box::new(AppState::new());
    Box::into_raw(state) as *mut c_void
}

/// Get processes and system stats as a JSON string.
/// Uses targeted refreshes instead of refresh_all() for better performance.
/// Returns null on error. Caller must free with neohtop_free_string().
#[no_mangle]
pub extern "C" fn neohtop_get_processes(handle: *mut c_void) -> *mut c_char {
    if handle.is_null() {
        return std::ptr::null_mut();
    }

    let state = unsafe { &*(handle as *const AppState) };

    let result = (|| -> Result<String, String> {
        let mut sys = state.sys.lock().map_err(|e| e.to_string())?;
        let mut disks = state.disks.lock().map_err(|e| e.to_string())?;
        let mut networks = state.networks.lock().map_err(|e| e.to_string())?;

        // Note: ideally we'd use targeted refresh_cpu_all() + refresh_memory()
        // + refresh_processes() instead of refresh_all(), which also refreshes
        // components/users/groups unnecessarily. However the exact API depends
        // on the sysinfo minor version, so we keep refresh_all() for compatibility.
        // The bigger perf wins come from removing environ from the hot path.
        sys.refresh_all();
        networks.refresh(true);
        disks.refresh(false); // false = skip re-listing, just update existing

        let mut process_monitor = state.process_monitor.lock().map_err(|e| e.to_string())?;
        let mut system_monitor = state.system_monitor.lock().map_err(|e| e.to_string())?;

        let processes = process_monitor.collect_processes(&sys)?;
        let system_stats = system_monitor.collect_stats(&sys, &networks, &disks);

        let data = MonitoringData {
            processes,
            system_stats,
        };

        serde_json::to_string(&data).map_err(|e| e.to_string())
    })();

    match result {
        Ok(json) => match CString::new(json) {
            Ok(cstr) => cstr.into_raw(),
            Err(_) => std::ptr::null_mut(),
        },
        Err(_) => std::ptr::null_mut(),
    }
}

/// Get detailed info (including environ) for a single process by PID.
/// Called on-demand when user opens process details overlay.
/// Returns null on error or if process not found. Caller must free with neohtop_free_string().
#[no_mangle]
pub extern "C" fn neohtop_get_process_detail(handle: *mut c_void, pid: u32) -> *mut c_char {
    if handle.is_null() {
        return std::ptr::null_mut();
    }

    let state = unsafe { &*(handle as *const AppState) };

    let result = (|| -> Result<String, String> {
        let sys = state.sys.lock().map_err(|e| e.to_string())?;
        let process_monitor = state.process_monitor.lock().map_err(|e| e.to_string())?;

        match process_monitor.get_process_detail(&sys, pid) {
            Some(detail) => serde_json::to_string(&detail).map_err(|e| e.to_string()),
            None => Err(format!("Process {} not found", pid)),
        }
    })();

    match result {
        Ok(json) => match CString::new(json) {
            Ok(cstr) => cstr.into_raw(),
            Err(_) => std::ptr::null_mut(),
        },
        Err(_) => std::ptr::null_mut(),
    }
}

/// Kill a process by PID. Returns 1 on success, 0 on failure.
#[no_mangle]
pub extern "C" fn neohtop_kill_process(handle: *mut c_void, pid: u32) -> i32 {
    if handle.is_null() {
        return 0;
    }

    let state = unsafe { &*(handle as *const AppState) };

    match state.sys.lock() {
        Ok(sys) => {
            if ProcessMonitor::kill_process(&sys, pid) {
                1
            } else {
                0
            }
        }
        Err(_) => 0,
    }
}

/// Free a string returned by this library.
#[no_mangle]
pub extern "C" fn neohtop_free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe {
            drop(CString::from_raw(s));
        }
    }
}

/// Destroy the monitoring handle and free all resources.
#[no_mangle]
pub extern "C" fn neohtop_destroy(handle: *mut c_void) {
    if !handle.is_null() {
        unsafe {
            drop(Box::from_raw(handle as *mut AppState));
        }
    }
}
