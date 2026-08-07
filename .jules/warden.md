## 2025-02-20 - [Modernization] Missing t.Parallel()

**Pattern/Issue:** Many pure, side-effect-free standard library tests (`pkg/errkit/format_test.go` and `internal/application/struct_strings_test.go`) were missing `t.Parallel()`, needlessly serializing their execution.
**Learning:** This repo is using Go 1.26.5 and relies heavily on Ginkgo for most of the test suite. The standard library tests are somewhat of a secondary test suite and were likely written sequentially without optimizing for concurrent execution.
**Prevention:** Since the standard library `testing` package is used here, ensure `t.Parallel()` is added to independent tests that do not share state, especially structural unit tests. Loop variables do not need explicit capture here because the repo requires Go 1.22+.
