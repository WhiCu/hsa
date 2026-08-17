## 2026-08-17 - Linter & Refactoring rules
**Finding:** Modifying structural code to expose internal fields (e.g. `Pool()`) for testing violates encapsulation. `ginkgo` requires correct formatting and assertions. Static analysis checks strictly enforce unused parameter naming (`_`).
**Learning:** Do not implement test-only getters that expose private APIs; use `// uncovered:` annotations correctly.
**Prevention/Action:** Added `// uncovered: defensive sync.Pool assertion, we only put valid *batchBuffers` instead of creating `.Pool()`. Renamed unused parameters to `_` and fixed struct suffix definitions for tests to satisfy `errname` and `revive` linters.
