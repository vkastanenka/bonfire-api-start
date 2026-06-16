-- Extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- Functions
CREATE
OR REPLACE FUNCTION update_modified_column() RETURNS TRIGGER AS $ $ BEGIN NEW.updated_at = CURRENT_TIMESTAMP;

RETURN NEW;

END;

-- Language
$ $ LANGUAGE plpgsql;