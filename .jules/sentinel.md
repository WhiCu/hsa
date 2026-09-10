## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.
## 2024-05-24 - [External ID leak in application layer DTO]
**Vulnerability:** The `RegistrationResult` DTO exposed the `ExternalID` (credential ID) in its `String()` method via `fmt.Sprintf("%v", r.ExternalID)`.
**Learning:** Application-layer DTOs that contain sensitive identifiers must manually redact them to prevent unintended leakage via generic formatters or logs.
**Prevention:** Explicitly redact sensitive credential identifiers (e.g. `***REDACTED***`) in `String()` methods for sensitive DTOs.
