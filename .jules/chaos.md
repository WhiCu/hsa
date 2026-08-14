
## 2024-05-24 - [SecretManager Panic on Negative Token Length]
**Crash/Bug:** Calling `SecretManager.GenerateToken` with a negative length (`n < 0`) causes a panic (`runtime error: makeslice: len out of range`) due to `make([]byte, n)` in `generateRandomBase32`.
**Learning:** Functions accepting length parameters that directly feed into memory allocation functions like `make` must validate boundaries, especially against negative inputs. Without validation, this can lead to unhandled panics and potential DoS if the length is influenced by user input.
**Prevention:** Always bound-check length parameters (`n < 0` or against maximum allowed lengths) before using them in `make([]byte, n)`.
