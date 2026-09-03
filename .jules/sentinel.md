## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.
## 2025-02-23 - [Application DTO Leakage via Format Strings]
**Vulnerability:** The `RegistrationResult` DTO implemented a custom `String()` method but used `fmt.Sprintf("%v", r.ExternalID)` to append the WebAuthn Credential ID (a sensitive `[]byte` identifier) instead of redacting it.
**Learning:** Even when custom `String()` methods are implemented to redact sensitive data, developers may mistakenly use `fmt.Sprintf` for fields they do not immediately recognize as sensitive (like external identifiers), negating the protection.
**Prevention:** Strictly enforce literal `***REDACTED***` strings for all cryptographic and credential identifiers (like `ExternalID` and `PublicKey`) in application-layer DTO stringers.
## 2025-02-23 - [Vulnerable Dependency in x/crypto/ssh]
**Vulnerability:** A vulnerability check using `govulncheck` identified two DoS risks (GO-2026-6355, GO-2026-6354) involving deadlocked channels in `golang.org/x/crypto/ssh` v0.55.0, used indirectly by test container helpers via `testutil.StartPostgresContainer`.
**Learning:** Even indirect transitive dependencies in test utilities can flag critical or high-severity vulnerabilities during CI vulnerability scans, preventing pipeline completion.
**Prevention:** Regularly audit and bump transitive dependencies to their latest secure versions (e.g. bumping `golang.org/x/crypto` from `v0.55.0` to `v0.56.0` via `go get` and `go mod tidy`) to proactively clear CI security vulnerability checks.
