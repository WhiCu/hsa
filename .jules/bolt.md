## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
## 2024-05-18 - [Optimizing Append Overhead]
**Learning:** Using `append` with a pre-allocated capacity `make([]T, 0, len)` involves slight overhead from bounds and capacity checking inside the loop. Allocating with full length `make([]T, len)` and assigning by index `slice[i] = v` avoids these checks.
**Action:** When mapping elements 1-to-1 from one slice to another, prefer pre-allocating the target slice with length instead of capacity and assigning by index to maximize loop performance.
