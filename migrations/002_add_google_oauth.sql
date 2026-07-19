-- Add google_id column for OAuth
ALTER TABLE users ADD COLUMN google_id TEXT UNIQUE;
