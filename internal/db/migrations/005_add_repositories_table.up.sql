-- Create repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR NOT NULL UNIQUE,
    url VARCHAR NOT NULL,
    reference VARCHAR,
    compose_dir VARCHAR,
    cleanup_policy VARCHAR DEFAULT 'keep',
    use_podman_default_names BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add repository_id column to units table
ALTER TABLE units ADD COLUMN repository_id INTEGER;

-- Create index on repository_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_units_repository_id ON units(repository_id);

-- Create foreign key constraint
-- SQLite doesn't support adding foreign keys with ALTER TABLE, so we need to do this in phases:

-- 1. First migration - just add the column (done above)

-- 2. Create a temporary table with the new schema including foreign key
CREATE TABLE units_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR,
    type VARCHAR,
    cleanup_policy VARCHAR,
    sha1_hash BLOB,  -- From migration 002
    user_mode BOOLEAN DEFAULT 0, -- From migration 004
    repository_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, type), -- From migration 003
    FOREIGN KEY(repository_id) REFERENCES repositories(id) ON DELETE SET NULL
);

-- 3. Detect existing repositories from config and populate them
INSERT INTO repositories (name, url, reference, compose_dir, cleanup_policy, use_podman_default_names)
VALUES
    -- MIGRATION_PLACEHOLDER: This will be replaced during the migration
    -- The Go code will read the config file and insert appropriate VALUES here
    ('migration_placeholder', 'placeholder', NULL, NULL, 'keep', 0);

-- 4. Update units' repository_id based on name prefixes, relying on repository names from the config
UPDATE units SET repository_id = (
    SELECT r.id FROM repositories r 
    WHERE units.name LIKE (r.name || '-%')
    ORDER BY LENGTH(r.name) DESC -- Match the longest repository name prefix first
    LIMIT 1
);

-- 5. Copy all data from the old table to the new table
INSERT INTO units_new (id, name, type, cleanup_policy, sha1_hash, user_mode, repository_id, created_at)
SELECT id, name, type, cleanup_policy, sha1_hash, user_mode, repository_id, created_at FROM units;

-- 6. Drop the old table
DROP TABLE units;

-- 7. Rename the new table to the original name
ALTER TABLE units_new RENAME TO units;