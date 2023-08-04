ALTER TABLE status_errors ADD COLUMN row_count INT DEFAULT 0 NOT NULL;
ALTER TABLE status_errors ADD INDEX (enrollment_id, row_count);
