## 2024-05-24 - [SQL Injection via Double Quote Escaping]
**Vulnerability:** Dynamic table truncation in `migrations.go` used `fmt.Sprintf("%q", tableName)` to quote identifiers.
**Learning:** Go's `%q` string format escapes double quotes using a backslash (`\"`), but PostgreSQL requires double quotes to be escaped with another double quote (`""`). This means `%q` is unsafe and incorrect for sanitizing SQL identifiers, allowing potential SQL injection if an identifier name is maliciously controlled.
**Prevention:** Always use a dedicated database driver utility, such as `pgx.Identifier{name}.Sanitize()`, to safely quote and escape SQL identifiers instead of standard string formatting.

## 2025-02-23 - [Exposed Sensitive Data in Logs]
**Vulnerability:** The `String()` method of `RegistrationResult` used `fmt.Sprintf("%v", r.ExternalID)` to format the `ExternalID`, unintentionally exposing sensitive credentials byte array in logs.
**Learning:** Returning sensitive data like credential identifiers in the `.String()` method causes it to be leaked globally through any standard logging or error wrapping frameworks that use `%v` or `%+v`.
**Prevention:** Always substitute sensitive fields with strings like `"***REDACTED***"` in manually implemented `.String()` methods of Application-layer DTOs to ensure they are not exposed in logs or debugging endpoints.
