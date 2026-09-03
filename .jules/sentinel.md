## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.
## 2025-02-23 - [Application DTO Leakage via Format Strings]
**Vulnerability:** The `RegistrationResult` DTO implemented a custom `String()` method but used `fmt.Sprintf("%v", r.ExternalID)` to append the WebAuthn Credential ID (a sensitive `[]byte` identifier) instead of redacting it.
**Learning:** Even when custom `String()` methods are implemented to redact sensitive data, developers may mistakenly use `fmt.Sprintf` for fields they do not immediately recognize as sensitive (like external identifiers), negating the protection.
**Prevention:** Strictly enforce literal `***REDACTED***` strings for all cryptographic and credential identifiers (like `ExternalID` and `PublicKey`) in application-layer DTO stringers.
