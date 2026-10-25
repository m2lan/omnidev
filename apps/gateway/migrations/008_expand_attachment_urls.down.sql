-- Revert storage_key and storage_url columns to original sizes
ALTER TABLE attachments ALTER COLUMN storage_key TYPE VARCHAR(500);
ALTER TABLE attachments ALTER COLUMN storage_url TYPE VARCHAR(500);
ALTER TABLE attachments ALTER COLUMN thumbnail_key TYPE VARCHAR(500);
