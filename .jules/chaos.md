## 2025-02-12 - errkit Mutation Leak
**Crash/Bug:** errkit.Append mutates the underlying `*Error` target slice in-place, leading to data races and cross-contamination when the same base error is handled in multiple branches or concurrent goroutines.
**Learning:** `errors.AsType` yields a pointer to the original error object. Modifying properties on this pointer without a deep copy violates the immutability expectations of error handling.
**Prevention:** When extending error chains or composite errors, always create a deep copy of slices to ensure copy-on-write semantics. Avoid mutating state exposed by `errors.As`.
