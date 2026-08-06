#!/bin/bash
set -e

MIGRATIONS_DIR="db/migrations"
CONTAINER_NAME="bonfire_postgres"
DB_USER="postgres"
DB_NAME="bonfire_db"

echo "Applying migrations to database..."

# Execute all *.up.sql files in alphabetical/numerical order
for f in $(ls "$MIGRATIONS_DIR"/*.up.sql | sort -V); do
  echo "Executing $f..."
  docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" < "$f"
done

echo "Migrations complete!"