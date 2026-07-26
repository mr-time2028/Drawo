# Database Migrations Guide

Drawo uses a built-in migration system powered by `golang-migrate`. This guide explains how to set up the database, run existing migrations, and create new ones using the CLI.

---

## 1. Initial Setup

Before running migrations, ensure your environment is configured.

### 1.1 Start the Database
The easiest way to start the required PostgreSQL instance is via Docker:

```bash
# From the project root (Drawo/)
make dev-up
```

### 1.2 Configuration
Ensure your `.env` file in `Drawo/app/` has the correct database credentials. The migration tool reads these automatically:

```env
DRAWO_DATABASE_HOST=localhost
DRAWO_DATABASE_PORT=5432
DRAWO_DATABASE_NAME=drawo
DRAWO_DATABASE_USER=postgres
DRAWO_DATABASE_PASSWORD=123456
DRAWO_DATABASE_SSLMODE=disable
DRAWO_APP_MIGRATIONSPATH=migrations
```

---

## 2. Running Migrations

All migration commands are subcommands of `migrate`.

### 2.1 Apply All Pending Migrations
Run this at the start of the project or after pulling new code to bring your schema up to date.

```bash
cd app/
go run main.go migrate
```

### 2.2 Incremental Updates
To apply only the next single pending migration:

```bash
go run main.go migrate up
```

### 2.3 Rollbacks
To undo the very last migration:

```bash
go run main.go migrate down
```

To rollback **everything** and start from a blank database:

```bash
go run main.go migrate down --all
```

---

## 3. Monitoring & Maintenance

### 3.1 Check Status
To see which version the database is currently on and whether it is in a "dirty" state:

```bash
go run main.go migrate status
```

### 3.2 Fixing "Dirty" States
If a migration fails midway (e.g., due to a syntax error in SQL), PostgreSQL might mark the migration as "dirty," preventing further attempts. To force the version back to a clean state:

```bash
# Force the version to 1 (assuming version 1 is clean)
go run main.go migrate force 1
```

### 3.3 Target a Specific Version
To migrate the database specifically to version `X`:

```bash
go run main.go migrate go_to 5
```

---

## 4. Development Workflow

### 4.1 Creating a New Migration
When you need to change the database schema, use the `generate_migration` command. 

**Note:** Unlike the standard `migrate` tool, this is natively implemented in the Drawo binary. You do **not** need to install any external tools.

```bash
# Format: go run main.go generate_migration <module_name> <description>
go run main.go generate_migration user add_email_to_profiles
```

The command automatically prepends the next version number and the module name to the filename. In the example above, it would generate files named like `000013_user_add_email_to_profiles.up.sql`.

This will create two files in `app/migrations/`:
1. `[Version]_[Module]_[Description].up.sql`: Add your `ALTER TABLE` or `CREATE TABLE` logic here.
2. `[Version]_[Module]_[Description].down.sql`: Add the logic to undo the change (e.g., `DROP TABLE`).

### 4.2 Rules for Migrations
1. **Never leave a file empty.** The validator will throw an error. Add a comment like `-- Up migration` if it must be empty.
2. **Every `.up.sql` must have a matching `.down.sql`.**
3. **Use snake_case** for filenames.
4. **Test your rollbacks.** Always run `migrate down` and `migrate up` locally to ensure your migration is reversible.
