#ifndef NEOHTOP_CORE_H
#define NEOHTOP_CORE_H

#include <stdint.h>

// Initialize the monitoring system. Returns an opaque handle.
void* neohtop_init(void);

// Get processes and system stats as a JSON string.
// Caller must free the returned string with neohtop_free_string().
char* neohtop_get_processes(void* handle);

// Get detailed info (incl. environ) for a single process by PID.
// Returns null if process not found. Caller must free with neohtop_free_string().
char* neohtop_get_process_detail(void* handle, uint32_t pid);

// Kill a process by PID. Returns 1 on success, 0 on failure.
int neohtop_kill_process(void* handle, uint32_t pid);

// Free a string returned by the library.
void neohtop_free_string(char* s);

// Destroy the monitoring handle and free all resources.
void neohtop_destroy(void* handle);

#endif
