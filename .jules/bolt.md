## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
## 2024-05-18 - [Optimizing Allocations and Data Races]
**Learning:** Using `unsafe.String` on a byte slice stored in a struct pooled by `sync.Pool` introduces a severe data race / data corruption bug because the string shares memory with the pooled byte slice, which can be modified by another goroutine after being returned.
**Action:** When using `unsafe.String`, ensure the backing byte array is exclusively owned by the string and won't be reused or modified.
