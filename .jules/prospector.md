## 2026-08-17 - Uncovered comment rule compliance
**Finding:** Unreachable code should be explicitly labeled with an `// uncovered:` comment, rather than refactoring structural components just for the sake of triggering that specific line in a test.
**Learning:** Adding test-only API functions like `Pool()` specifically for tests breaks encapsulation and should be avoided in favor of documentation.
**Prevention/Action:** Reverted the `Pool()` exposure and added an explicit `// uncovered: defensive sync.Pool assertion, we only put valid *batchBuffers` comment instead.
