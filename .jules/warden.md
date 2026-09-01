## 2025-02-20 - [Modernization] Missing t.Parallel()

**Pattern/Issue:** Many pure, side-effect-free standard library tests (`pkg/errkit/format_test.go` and `internal/application/struct_strings_test.go`) were missing `t.Parallel()`, needlessly serializing their execution.
**Learning:** This repo is using Go 1.26.5 and relies heavily on Ginkgo for most of the test suite. The standard library tests are somewhat of a secondary test suite and were likely written sequentially without optimizing for concurrent execution.
**Prevention:** Since the standard library `testing` package is used here, ensure `t.Parallel()` is added to independent tests that do not share state, especially structural unit tests. Loop variables do not need explicit capture here because the repo requires Go 1.22+.
## 2025-02-18 - Improve test coverage in domain/session
**Pattern/Issue:** Incomplete test coverage (50%) for the `session` domain entity, missing testing for simple getters, token rotation (`Rotate`), struct reconstruction (`Reconstruct`), and invalid IP validation edge case.
**Learning:** Even trivial logic like getters and manual structure reconstruction functions require testing to meet Codecov requirements (80% diff hit threshold). Tests should also fully exhaust validation pathways.
**Prevention:** Always write comprehensive Ginkgo specs to test newly added methods and structs, no matter how simple they might seem, especially simple domain entity operations. Include boundary condition testing like invalid initializations (e.g., zero `netip.Addr`).
## 2025-02-21 - [Modernization] Outdated context.Background() usage

**Pattern/Issue:** Many Ginkgo specs in the `storage` and `logger` packages were manually overriding DI containers or injecting `context.Background()` instead of utilizing the test-bound contexts.
**Learning:** `t.Context()` (Go 1.24+) or Ginkgo's native test context (`SpecContext`) are strictly better as they natively cancel when the test finishes, reducing the chance of goroutine leaks.
**Prevention:** In Ginkgo specs, use `ctx SpecContext` and pass it directly to `do.OverrideValue[context.Context](injector, ctx)`. For standard `testing.T` tests, use `t.Context()`.
## 2025-02-21 - [Flaky Test] Non-deterministic time.Sleep synchronization
**Pattern/Issue:** In `internal/application/create_invite_test.go`, tests verifying concurrency control used `time.Sleep(5 * time.Millisecond)` to "give goroutines time to prepare" before starting the benchmark.
**Learning:** Wall-clock synchronization is inherently flaky in CI. The correct idiomatic approach is to use a `readyWg sync.WaitGroup` that blocks the main test thread until all worker goroutines have called `readyWg.Done()` right before blocking on the `startGate` channel.
**Prevention:** Never use `time.Sleep` to coordinate goroutine startup in tests. Always use a dedicated wait group or channel signaling pattern to guarantee deterministic synchronization.

## 2025-02-21 - [Modernization] Context swallowed by testify mock Returns
**Pattern/Issue:** In `internal/presentation/http/handler_test.go`, the `mockAuthUnauthorized` function was mocking `HandleBearerAuth` by returning `context.Background()` directly using `.Return(context.Background(), err)`.
**Learning:** Returning a bare background context drops any cancellation signals or values that should logically pass through from the request context.
**Prevention:** When mocking middleware or security handlers that modify and return a context, use `.RunAndReturn(func(ctx ...) { return ctx, err })` to pass the contextual state correctly, even on failure.
## 2025-02-21 - Ginkgo TB Helpers Location Constraints
**Pattern/Issue:** Using `GinkgoT().Setenv()` or `GinkgoT().TempDir()` directly inside container nodes (e.g., `Describe`, `Context`) or at the top level of the test file causes a runtime panic.
**Learning:** Ginkgo's `GinkgoT()` helper wraps standard library `testing.TB` functionalities. Methods that register cleanups (`Setenv`, `TempDir`, `Cleanup`) rely on Ginkgo's internal `DeferCleanup` mechanism, which is strictly scoped. Calling these outside of setup nodes (`BeforeEach`, `BeforeAll`) or subject nodes (`It`) violates Ginkgo's execution model because container nodes execute during the initial tree-building phase, not during the actual test execution.
**Prevention:** Always ensure calls to `GinkgoT().Setenv()`, `GinkgoT().TempDir()`, and `GinkgoT().Cleanup()` are placed inside `BeforeEach` (or `It`) blocks, rather than directly in `Describe` or `Context` closures.

## 2024-05-24 - Ginkgo DeferCleanup and os.TempDir
**Pattern/Issue:** Found `os.TempDir()` usage in `logger_test.go` dropping files into the shared `/tmp` dir and relying on a manual `closer.Close()` assertion at the end of the test block.
**Learning:** This caused test pollution when run repeatedly because files weren't explicitly cleaned up, and moving to `GinkgoT().TempDir()` revealed a flaw: the temp dir cleanup happens *before* the garbage collector closes the file, leading to "directory not empty" failures.
**Prevention:** Always pair `GinkgoT().TempDir()` with `DeferCleanup(closer.Close)` when writing to files inside it, to ensure file descriptors are released *before* Ginkgo attempts to tear down the temp directory.

## 2025-02-21 - [Modernization] Proper usage of Ginkgo DeferCleanup
**Pattern/Issue:** Using `DeferCleanup` at the very end of an `It` block in Ginkgo tests to clean up resources (like file descriptors from `GinkgoT().TempDir()`).
**Learning:** If an assertion (like `Expect(err).NotTo(HaveOccurred())`) fails earlier in the test, test execution halts, and the `DeferCleanup` registered at the end of the block is never reached. This defeats its purpose of guaranteeing teardown on failure.
**Prevention:** Register `DeferCleanup(func)` immediately after the object is created and nil-checked (e.g., right after `Expect(closer).NotTo(BeNil())`), so it acts like a standard `defer` that executes even on failure.

