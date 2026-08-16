## 2023-10-24 - PostgreSQL SQL Injection via fmt.Sprintf

**Vulnerability:** Constructing PostgreSQL table truncate queries using `fmt.Sprintf("TRUNCATE TABLE %s", fmt.Sprintf("%q", tableName))` creates an SQL Injection vulnerability because `%q` escapes characters differently than PostgreSQL identifier quoting rules (PostgreSQL requires doubling inner double-quotes `""`).
**Learning:** Never use `fmt` formatting to escape database identifiers.
**Prevention:** Use standard library parameterization where possible. When dealing with DDL (which cannot be parameterized), rely on database-specific libraries like `github.com/jackc/pgx/v5` and explicitly call `pgx.Identifier{name}.Sanitize()`.
