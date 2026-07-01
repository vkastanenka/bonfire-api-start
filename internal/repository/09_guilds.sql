-- ==========================================
-- GUILDS
-- ==========================================
CREATE TABLE guilds(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    owner_id uuid NOT NULL REFERENCES users(id),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TRIGGER update_guilds_modtime
    BEFORE UPDATE ON guilds
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- GUILDS PROFILES
-- ==========================================
CREATE TABLE guild_profiles(
    guild_id uuid PRIMARY KEY REFERENCES guilds(id) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    icon_url text,
    banner_hex varchar(7) DEFAULT '#99aab5',
    description text,
    visibility smallint NOT NULL DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT check_valid_visibility CHECK (visibility IN (0, 1)),
    CONSTRAINT check_hex_color_format CHECK (banner_hex ~* '^#[A-Fa-f0-9]{6}$')
);

CREATE TRIGGER update_guild_profiles_modtime
    BEFORE UPDATE ON guild_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- GUILD MEMBERS
-- ==========================================
CREATE TABLE guild_members(
    guild_id uuid NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (guild_id, user_id)
);

CREATE INDEX idx_guild_members_user ON guild_members(user_id);

CREATE TRIGGER update_guild_members_modtime
    BEFORE UPDATE ON guild_members
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- ROLES (The Permission Container)
-- ==========================================
CREATE TABLE guild_roles(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    guild_id uuid NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    name varchar(255) NOT NULL,
    color_hex varchar(7) DEFAULT '#99aab5',
    permissions bigint NOT NULL DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT check_hex_color_format CHECK (banner_color ~* '^#[A-Fa-f0-9]{6}$')
);

CREATE INDEX idx_guild_roles_guild ON guild_roles(guild_id);

CREATE TRIGGER update_guild_roles_modtime
    BEFORE UPDATE ON guild_roles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- MEMBER ROLES (Junction Table)
-- ==========================================
CREATE TABLE guild_member_roles(
    guild_id uuid NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES guild_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (guild_id, user_id, role_id),
    FOREIGN KEY (guild_id, user_id) REFERENCES guild_members(guild_id, user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_guild_member_roles_modtime
    BEFORE UPDATE ON guild_member_roles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE INDEX idx_guild_member_roles_user ON guild_member_roles(user_id, guild_id);

