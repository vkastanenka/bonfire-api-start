#!/bin/bash
# List of files in the correct order
FILES=(
  "internal/repository/tables.sql"
)

echo "Applying migrations to database..."

for f in "${FILES[@]}"; do
  echo "Executing $f"
  cat "$f" | docker exec -i bonfire_postgres psql -U postgres -d bonfire_db
done

echo "Done!"