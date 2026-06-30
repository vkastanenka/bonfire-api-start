CREATE TABLE delete_requests(
    -- Primary key
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Audit metadata
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    scheduled_at timestamp with time zone NOT NULL,
    -- Constraints
    CONSTRAINT valid_schedule_date CHECK (scheduled_at > created_at)
);

-- Indexes
CREATE INDEX idx_delete_requests_scheduled_at ON delete_requests(scheduled_at);

