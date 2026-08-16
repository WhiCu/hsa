
## 2024-05-24 - [SecretManager Panic on Negative Token Length]
**Crash/Bug:** Calling `SecretManager.GenerateToken` with a negative length (`n < 0`) causes a panic (`runtime error: makeslice: len out of range`) due to `make([]byte, n)` in `generateRandomBase32`.
**Learning:** Functions accepting length parameters that directly feed into memory allocation functions like `make` must validate boundaries, especially against negative inputs. Without validation, this can lead to unhandled panics and potential DoS if the length is influenced by user input.
**Prevention:** Always bound-check length parameters (`n < 0` or against maximum allowed lengths) before using them in `make([]byte, n)`.
## 2025-03-05 - errkit Mutability Bug
**Crash/Bug:** Calling `errkit.Append` on an existing `errkit.Error` directly modified the base error's underlying `Errors` slice. If the original base error was shared or appended to concurrently across different code paths, they would clobber or mistakenly share each other's appended errors, leading to incorrect error aggregations or potential panics/races during concurrent usage.
**Learning:** Returning a mutated version of a struct from an unwrapping/utility function like `errors.AsType` violates immutability expectations for error aggregators. When unwrapping composite errors, shared state slices must be deep-copied before appending new elements.
**Prevention:** Deep-copy internal slices before modifying them when returning new composite errors in helper utilities.
## 2025-08-16 - errkit.Group Unhandled Panics
**Crash/Bug:** Calling `errkit.Group.Go` or `errkit.Group.Finally` with a function that panics caused the panic to propagate unhandled, crashing the host process.
**Learning:** Utilities that spawn goroutines or execute callbacks on behalf of callers (like `errkit.Group`) must assume the callbacks might panic. If they do not recover, they fail to contain the failure to the calling context and cause application-wide crashes.
**Prevention:** Wrap callback executions in `defer func() { if r := recover(); r != nil { ... } }()` to explicitly convert runtime panics into collected errors (e.g., `errkit.PanicError`) so the application remains stable.
