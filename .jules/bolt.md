## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
## 2026-08-16 - [Avoiding premature optimization on cold paths]
**Learning:** Even obvious performance wins like strings.Builder vs fmt.Sprintf should be evaluated for their execution frequency. Optimizing code on cold paths (like database reset functions that only run in test setups) violates the strict rule against premature optimization without measurable impact, and creates unneeded noise. Error formatting logic in errkit, however, is a very hot path due to widespread error wrapping and logging usage.
**Action:** Always check the call sites and execution frequency of code being optimized before making changes. Avoid any optimizations on test setup/teardown functions or isolated one-off scripts.
## 2025-02-23 - [Optimizing fmt.Sprintf reflection on hot panic recovery paths]
**Learning:** `fmt.Sprintf` uses reflection to format arguments, which causes significant overhead and memory allocations. In hot code paths, such as generic panic recovery where values are predominantly of type `error` or `string`, using type assertions (`val.(error)` or `val.(string)`) combined with simple string concatenation is much faster.
**Action:** Replace `fmt.Sprintf` with type assertions and concatenation for common types before falling back to `fmt.Sprintf` when dealing with untyped (`any`) variables in hot paths to boost performance and reduce allocations.

## 2025-02-23 - [Optimize string splitting in loops]
**Learning:** `strings.Split` allocates a new slice containing all split segments. In performance-critical hot paths like HTTP middleware where headers are parsed (e.g., `X-Forwarded-For`), splitting strings and iterating with `slices.Backward` adds unnecessary allocation and GC pressure.
**Action:** Use manual right-to-left string parsing with `strings.LastIndexByte` instead of `strings.Split` when searching or iterating backwards over a comma-separated list to prevent temporary slice allocations.
## 2026-08-27 - Optimize domain entities String() formatters
**Learning:** Replaced `fmt.Sprintf` with direct string concatenation and explicit conversions (`time.Format`, `strconv`) in `.String()` methods of domain models. This avoids heavy reflection on hot paths and achieved up to 4x speedup (2431 ns/op -> 654 ns/op) during logging and debugging.
**Action:** Default to string concatenation or `strings.Builder` for critical or high-volume `.String()` implementations instead of `fmt.Sprintf`.
## 2026-08-28 - Optimize DTO struct String() formatters
**Learning:** Replaced `fmt.Sprintf` with string concatenation (`+`) and explicit `strconv` formatting in `.String()` methods of Application-layer DTOs (e.g., `LoginInput`, `RegistrationResult`). This avoids heavy runtime reflection in frequently invoked paths, such as logging middleware, significantly improving performance and reducing allocations. Slices of structs were formatted efficiently by pre-allocating or writing to a `strings.Builder`.
**Action:** Default to explicit string concatenation and `strconv`/`strings.Builder` for frequently invoked `.String()` formatters instead of relying on `fmt.Sprintf`.
