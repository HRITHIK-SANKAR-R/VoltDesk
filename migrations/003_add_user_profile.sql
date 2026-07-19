-- Add name and avatar_url for user profile
ALTER TABLE users ADD COLUMN name TEXT;
ALTER TABLE users ADD COLUMN avatar_url TEXT;
