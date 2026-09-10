## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.

## 2024-06-25 - [Sensitive Data Exposure in RegistrationResult]
**Vulnerability:** The `CredentialID` (which represents the `ExternalID`) in `RegistrationResult` was being formatted using `fmt.Sprintf("%v", r.ExternalID)` and included in the `String()` method output, exposing sensitive credential data in logs.
**Learning:** Application-layer DTOs that contain sensitive data (like credential identifiers or external IDs) must manually redact those fields in their `.String()` methods to prevent leakage into standard logs or generic error formatters.
**Prevention:** Always use `***REDACTED***` for sensitive fields like `CredentialID`, `ExternalID`, or `PublicKey` in the custom `String()` methods of domain and application DTOs.
