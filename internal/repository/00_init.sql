-- Case-Insensitive Text extension
CREATE EXTENSION IF NOT EXISTS citext;

-- Automatically update "updated_at" to CURRENT_TIMESTAMP
CREATE
OR REPLACE FUNCTION update_modified_column() RETURNS TRIGGER AS $ $ BEGIN NEW.updated_at = CURRENT_TIMESTAMP;

RETURN NEW;

END;

-- Language
$ $ LANGUAGE plpgsql;