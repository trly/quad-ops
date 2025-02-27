-- First, create a temporary table to identify the records to keep
CREATE TEMPORARY TABLE units_to_keep AS
SELECT MAX(id) as id
FROM units
GROUP BY name, type;

-- Delete records that are not in the units_to_keep table
DELETE FROM units
WHERE id NOT IN (SELECT id FROM units_to_keep);

-- Now add the unique constraint (SQLite syntax)
CREATE UNIQUE INDEX unique_name_type ON units(name, type);
