---
trigger: always_on
---

# PostgreSQL Technical Standards

*This file supplements the policies in `antigravity.md`.*

## 1. Oracle to PostgreSQL Type Mapping
- **Identity**: Convert Oracle Sequences to `GENERATED ALWAYS AS IDENTITY`.
- **Numeric**: Convert `NUMBER(p,0)` to `INTEGER` or `BIGINT` depending on precision.
- **Floating**: Do not use physical constraints on scalar types; use `NUMERIC` instead of `NUMBER(p,s)`.
- **Strings**: Always map `VARCHAR2` to `TEXT` (as per Master Rule).

## 2. Function Implementation Details
- **Procedural Logic**: `LANGUAGE plpgsql` is permitted ONLY for complex logic (loops/temporary variables) that cannot be expressed in standard SQL.
- **Immutability**: Mark getters and simple calculation functions as `IMMUTABLE` for performance.
- **Example Pattern**:
   ```postgresql
   CREATE OR REPLACE FUNCTION get_char(p_date DATE, p_fmt TEXT) 
   RETURNS TEXT LANGUAGE SQL IMMUTABLE 
   RETURN to_char(p_date, p_fmt);
   ```
- **CRUD**: 
-- Create CRUD functions for base tables input / output JSON except if only first parameter
-- Create SQL Query (pipline) functions instead of view, input JSON, output JSON Array   

## 3. Deployment Context
- **Target Version**: PostgreSQL 18.0.
- **Verification**: Always verify row counts and data integrity after DML execution using audit views.

## 4. CRUD and Data Access
- **JSON Integration**: Prefer database functions that accept and return `JSONB` for CRUD operations. This simplifies Go-to-DB mapping and enables consistent data structures across layers.
- **Example CRUD**: Use `jsonb_agg` for lists and `to_jsonb` for single records.

## 5. SQL Formatting & Schema
- **Column Definitions**: Keep each column definition in exactly one line. For generated columns, use explicit spacing around all parentheses and operators (e.g., `( ( EXTRACT ( YEAR FROM col ) * 100 ) )`).
- **Search Path**: Never use `search_path` settings in any code. Set the search path only at connection time.

## 6. DDL
- **Comments**: prefer the line comment on short length columns) and object comments too