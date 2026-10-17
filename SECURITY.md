# Security Policy

## Supported versions

Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| 0.5.x   | ✅        |
| < 0.5   | ❌        |

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Report vulnerabilities privately via [GitHub Security Advisories](https://github.com/ruko1202/swaggerui/security/advisories/new). Include:

- affected version (module version and `current_version.txt`)
- steps to reproduce or a proof of concept
- impact as you understand it

You should get a response within 7 days. Once the issue is confirmed, a fix will be released and the advisory published.

## Scope

This module embeds the [Swagger UI](https://github.com/swagger-api/swagger-ui) dist assets as-is.

- Issues in the Go handlers (`Handler`, `HandlerWithSpecs`) or the initializer template — report here.
- Issues in Swagger UI itself — report upstream to [swagger-api/swagger-ui](https://github.com/swagger-api/swagger-ui/security). Feel free to also open an issue here asking to bump the embedded assets once a fixed upstream release is out.

Note that the handlers serve your specs without authentication. Protecting the Swagger UI route (e.g. not exposing it in production or putting it behind auth middleware) is the responsibility of the application that mounts it.
