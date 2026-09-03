## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.
## 2026-09-03 - Bump golang.org/x/crypto to v0.56.0
**Vulnerability:** GO-2026-6355 / GO-2026-6354 found in golang.org/x/crypto/ssh where a deadlock could cause DoS.
**Learning:** External tooling and CI checks like `govulncheck` will flag standard sub-modules vulnerabilities that could halt deployments even in unused or test paths (e.g., testcontainers using ssh).
**Prevention:** Keep `golang.org/x/crypto` and other high-level `golang.org/x` dependencies up to date, especially those related to networking or cryptography.
