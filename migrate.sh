#!/bin/bash
# List of files in the correct order
FILES=(
  "internal/repository/00_init.sql"
  "internal/repository/01_outbox_events.sql"
  "internal/repository/02_users.sql"
  "internal/repository/03_delete_requests.sql"
  "internal/repository/04_profiles.sql"
  "internal/repository/05_sessions.sql"
  "internal/repository/06_channels.sql"
  "internal/repository/07_relationships.sql"
)

echo "Applying migrations to database..."

for f in "${FILES[@]}"; do
  echo "Executing $f"
  cat "$f" | docker exec -i bonfire_postgres psql -U postgres -d bonfire_db
done

echo "Done!"