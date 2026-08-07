## 2024-05-18 - [Optimizing HMAC SHA256 in SecretManager]
**Learning:** HMAC operations in Go `hmac.New(sha256.New, key)` are surprisingly expensive due to allocations and setup time for every invocation. Reusing HMAC instances using `sync.Pool` significantly cuts down execution time and allocations.
**Action:** Use a `sync.Pool` with `hash.Hash` inside `SecretManager` when multiple HMACs are generated to reduce allocations and speed up hashing operations. Ensure `Reset()` is called on the retrieved hash object before use.
