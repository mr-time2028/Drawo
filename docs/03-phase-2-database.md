# Phase 2 — Database Schema & Migrations

**Status:** Completed.

## What was delivered

1.  **PostgreSQL Schema Design**
    *   Designed a complete relational schema for all 11 persistent entities.
    *   Enforced 1:1, 1:N, and N:M relationships with proper foreign keys.
    *   Implemented UUIDs as primary keys for all tables.
    *   Applied `ON DELETE CASCADE` for user-dependent data and `ON DELETE SET NULL` for historical records to maintain data integrity.

2.  **SQL Migrations**
    *   `app/migrations/000001_init_schema.up.sql`: Full schema creation script with indexes and constraints.
    *   `app/migrations/000001_init_schema.down.sql`: Rollback script.

3.  **Documentation**
    *   `docs/database-schema.md`: Comprehensive documentation of table roles, columns, and integrity rules.

4.  **Performance Optimization**
    *   Added indexes for high-traffic query paths: `username` lookups, `email` lookups, foreign key traversal, and historical log lookups.

## Verification

The migrations follow standard `golang-migrate` naming conventions and can be applied using the project's CLI or migration tools once the database infrastructure is wired up.

## Next step

**Phase 3 — Domain Entities & Repository Interfaces.**
We will now refine the pure domain structs and repository interfaces to match this schema perfectly, ensuring the "Ports" layer is ready for implementation.
