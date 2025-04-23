-- This down migration removes the repository_id column and repositories table

-- 1. Create a temporary table without the repository_id column
CREATE TABLE units_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR,
    type VARCHAR,
    cleanup_policy VARCHAR,
    sha1_hash BLOB,
    user_mode BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, type)
);

-- 2. Copy data from the current table to the temporary table
INSERT INTO units_old (id, name, type, cleanup_policy, sha1_hash, user_mode, created_at)
SELECT id, name, type, cleanup_policy, sha1_hash, user_mode, created_at FROM units;

-- 3. Drop the current table
DROP TABLE units;

-- 4. Rename the temporary table to the original name
ALTER TABLE units_old RENAME TO units;

-- 5. Drop the repositories table
DROP TABLE repositories;