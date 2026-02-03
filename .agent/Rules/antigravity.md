# Antigravity Rules for SlideForge

## 1. Core Behavior
- **Proactive Execution**: Do not ask for permission to run tests, build scripts, or SQL. 
- **Integrity**: Never modify "copied" files (check source/origin before editing). 
- **Environment Awareness**: Always read `.env` from the project root using the `config` package.
- **Documentation**: Keep `README.md` and requirements updated.

## 2. PostgreSQL Master Policies
- **Idempotency**: All DDL MUST be idempotent (`CREATE OR REPLACE`, `IF NOT EXISTS`, `DROP ... CASCADE`).
- **No Schema Hardcoding**: NEVER use schema names (e.g., `dwh.table`) in DDL scripts. 
- **No search_path in Scripts**: NEVER use `SET search_path` inside SQL files. 
- **External Configuration**: search_path must be set via connection string (e.g. Go config) or `PGOPTIONS`.
- **Naming**: Use `snake_case` for all database objects. Use English for labels/comments.

## 3. Implementation Standards
- **Function Style**: Prefer `LANGUAGE sql` over `plpgsql`. Use the `$fn_name$` delimiter format.
- **SQL Functions**: Use the `return expression;` syntax for simple SQL function implementations.
- **Data Types**: Prefer `TEXT` over `VARCHAR`/`VARCHAR2`. Use `TIMESTAMP` for Oracle `DATE`.
- **Security**: Never commit `.env` or sensitive credentials to the repository.

## 4. Migration Patterns
- **Asynchronous Flow**: Separate `orcl` export from `pg` ingestion.
- **Staging Tier**: Always use a staging layer before transforming data into `dwh`.
