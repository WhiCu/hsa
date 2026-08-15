## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
## 2024-05-24 - Preallocate errkit slice capacity
**Learning:** In highly trafficked error composition code paths, repeated `.Errors` slice allocations due to un-capped appends can add significant allocation overhead. When composing errors with variadic inputs, pre-allocating the underlying slice capacity with `make([]error, len(base), len(base)+len(errs))` avoids O(log N) slice capacity re-allocations in Go when appending multiple errors.
**Action:** When implementing composable collections or error trees like `errkit.Append`, always calculate and pre-allocate capacity bounds ahead of variadic iteration to minimize allocations and latency.
