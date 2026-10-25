-- Expand storage_key and storage_url columns to accommodate longer presigned URLs
ALTER TABLE attachments ALTER COLUMN storage_key TYPE VARCHAR(1024);
ALTER TABLE attachments ALTER COLUMN storage_url TYPE TEXT;
ALTER TABLE attachments ALTER COLUMN thumbnail_key TYPE VARCHAR(1024);
