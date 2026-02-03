# Go Back-end Rules

## Architecture
- **Standard Library**: Prefer Go standard library for HTTP server and routing where possible.
- **Dependency Management**: Use `viper` for config, `godotenv` for env vars, and `lib/pq` for Postgres.
- **Project Structure**:
    - `cmd/server/`: Main entry point.
    - `internal/config/`: Configuration logic.
    - `internal/database/`: Persistence layer.
    - `internal/handlers/`: Business logic and HTTP handlers.
    - `ui/`: Static assets and templates.

## Performance
- **Database Connection**: Use `sql.DB` connection pooling. 
- **Templating**: Use `html/template` with global functions for i18n lookup.
- **Concurrency**: Leverage Go routines for background tasks.

## Internationalization (i18n)
- **Format**: Translation files should be JSON-based in `resources/`.
- **Lookup**: Use the `i18n.go` service for safe key-based lookups with fallbacks.
- **Cookie-based**: Language preference should be stored in a `lang` cookie.

## Error Handling
- **Logging**: Use the standard `log` package or `zap` if high performance logging is needed.
- **Graceful Failures**: Return clean HTML fragments via HTMX for partial failures instead of crashing.
