## 2026-08-17 - Linter & Refactoring rules
**Finding:** Modifying structural code to expose internal fields (e.g. `Pool()`) for testing violates encapsulation. `ginkgo` requires correct formatting and assertions. Static analysis checks strictly enforce unused parameter naming (`_`).
**Learning:** Do not implement test-only getters that expose private APIs; use `// uncovered:` annotations correctly.
**Prevention/Action:** Added `// uncovered: defensive sync.Pool assertion, we only put valid *batchBuffers` instead of creating `.Pool()`. Renamed unused parameters to `_` and fixed struct suffix definitions for tests to satisfy `errname` and `revive` linters.
## 2023-10-27 - Testcontainers Fallback & webauthn Untestable Code
**Finding:** Running `go test ./... -tags=testcontainers` might fail in restricted CI/Docker environments (like Docker-in-Docker overlayfs errors). `github.com/go-webauthn/webauthn` has unreachable error branches internally related to crypto/rand or hardcoded JSON structs that cannot be meaningfully tested via its public API.
**Learning:** For testcontainers, always have a fallback. For webauthn internal failures, explicit `// uncovered: ...` documentation is the correct approach to prevent wasting time on unmockable deep dependencies.
**Prevention/Action:** Use `docker info` to guard testcontainers execution or fall back gracefully. Apply `// uncovered: internal crypto/rand failure is practically untestable without deep mocking of go-webauthn` for webauthn-specific gaps.
