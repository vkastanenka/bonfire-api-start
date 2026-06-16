CREATE TABLE user_delete_requests (
    -- Primary key
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Audit metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    -- Constraints
    CONSTRAINT valid_schedule_date CHECK (scheduled_at > created_at)
);

-- Indexes
CREATE INDEX idx_delete_requests_scheduled_at ON user_delete_requests(scheduled_at);