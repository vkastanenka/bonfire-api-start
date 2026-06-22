CREATE TABLE outbox_events(
    -- Primary key
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Audit metadata
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Event payload
    event_type varchar(100) NOT NULL,
    payload jsonb NOT NULL,
    -- Queue state
    processed_at timestamptz,
    attempts int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    next_attempt_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error text,
    -- Constraints
    CONSTRAINT check_attempts_ceiling CHECK (attempts <= max_attempts)
);

-- Indexes
CREATE INDEX idx_outbox_events_unprocessed ON outbox_events(next_attempt_at, id ASC)
WHERE
    processed_at IS NULL;

-- Trigger
CREATE TRIGGER update_outbox_events_modtime
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

