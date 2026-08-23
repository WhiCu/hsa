## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
## 2026-08-16 - [Avoiding premature optimization on cold paths]
**Learning:** Even obvious performance wins like strings.Builder vs fmt.Sprintf should be evaluated for their execution frequency. Optimizing code on cold paths (like database reset functions that only run in test setups) violates the strict rule against premature optimization without measurable impact, and creates unneeded noise. Error formatting logic in errkit, however, is a very hot path due to widespread error wrapping and logging usage.
**Action:** Always check the call sites and execution frequency of code being optimized before making changes. Avoid any optimizations on test setup/teardown functions or isolated one-off scripts.
## 2025-02-23 - [Optimizing fmt.Sprintf reflection on hot panic recovery paths]
**Learning:** `fmt.Sprintf` uses reflection to format arguments, which causes significant overhead and memory allocations. In hot code paths, such as generic panic recovery where values are predominantly of type `error` or `string`, using type assertions (`val.(error)` or `val.(string)`) combined with simple string concatenation is much faster.
**Action:** Replace `fmt.Sprintf` with type assertions and concatenation for common types before falling back to `fmt.Sprintf` when dealing with untyped (`any`) variables in hot paths to boost performance and reduce allocations.
## 2026-08-23 - [Optimizing IP Address Parsing Hot Path]
**Learning:** `strings.Split` allocates memory for every call, which creates garbage collection overhead when used in high-frequency hot paths like HTTP middleware (`firstUntrustedFromRight`).
**Action:** Replace `strings.Split` and backwards iteration with manual right-to-left string slicing using `strings.LastIndexByte` for zero-allocation parsing in performance-critical code.
