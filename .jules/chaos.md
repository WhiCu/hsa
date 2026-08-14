
## 2024-05-24 - [SecretManager Panic on Negative Token Length]
**Crash/Bug:** Calling `SecretManager.GenerateToken` with a negative length (`n < 0`) causes a panic (`runtime error: makeslice: len out of range`) due to `make([]byte, n)` in `generateRandomBase32`.
**Learning:** Functions accepting length parameters that directly feed into memory allocation functions like `make` must validate boundaries, especially against negative inputs. Without validation, this can lead to unhandled panics and potential DoS if the length is influenced by user input.
**Prevention:** Always bound-check length parameters (`n < 0` or against maximum allowed lengths) before using them in `make([]byte, n)`.

## 2024-08-14 - errkit.Append Mutates Underlying Error Slice
**Crash/Bug:** Calling `errkit.Append()` on a wrapped `errkit.Error` directly mutates the underlying shared slice of errors. This meant multiple concurrent accesses or successive distinct error wrappings based on the same parent error would erroneously share state and cause unpredictable mutations or race conditions (if concurrent). Also `errkit.Append` appending nil errors blindly caused unexpected issues.
**Learning:** Functions designed to aggregate or append to shared state exposed via error unwrapping tools like `errors.AsType` must deep-copy internal slices before modification to strictly preserve immutability.
**Prevention:** Always allocate a new slice when appending to a custom error type containing a slice of underlying errors instead of modifying the internal state directly.
