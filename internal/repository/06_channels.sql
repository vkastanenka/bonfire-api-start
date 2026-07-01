CREATE TABLE channels(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    type SMALLINT NOT NULL,
    guild_id uuid REFERENCES guilds(id) ON DELETE CASCADE,
    name varchar(100),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Ensures valid integer type boundaries
    CONSTRAINT check_valid_channel_type CHECK (type IN (0, 1, 2, 3, 4)),
    -- Enforces that Guild channels (0, 2, 4) REQUIRE a guild_id,
    -- while Private channels (1=DM, 3=Group DM) strictly FORBID it.
    CONSTRAINT check_guild_id_placement CHECK ((type IN (0, 2, 4) AND guild_id IS NOT NULL) OR (type IN (1, 3) AND guild_id IS NULL))
);

-- Channel Members table to manage access to private DM and Group DM channels
CREATE TABLE channel_members(
    channel_id uuid REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (channel_id, user_id)
);

-- Indexes
CREATE INDEX idx_channels_guild_id ON channels(guild_id)
WHERE
    guild_id IS NOT NULL;

CREATE INDEX idx_channel_members_user ON channel_members(user_id);

-- Triggers
CREATE TRIGGER update_channels_modtime
    BEFORE UPDATE ON channels
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TRIGGER update_channel_members_modtime
    BEFORE UPDATE ON channel_members
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

