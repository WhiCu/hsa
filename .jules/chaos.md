## 2025-08-05 - Nil callbacks panic on execution
**Crash/Bug:** errkit.Group panics asynchronously or during Wait() if a nil function is provided to Go() or Finally().
**Learning:** Callbacks passed to wrapper structs without validation can cause unrecoverable segment faults when the internal library executes the nil pointer.
**Prevention:** Always add a nil check when receiving callback functions in library utilities, before executing them or queueing them for execution.
