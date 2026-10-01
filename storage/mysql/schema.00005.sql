-- order status reports and errors by an append-only id rather than
-- renumbering every row for the enrollment on each new status report.
-- existing rows are numbered in their (insertion) clustered index order.
ALTER TABLE status_reports
    ADD COLUMN id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY FIRST,
    DROP INDEX enrollment_id_2,
    DROP INDEX enrollment_id,
    DROP COLUMN row_count,
    ADD INDEX (enrollment_id, id);
ALTER TABLE status_errors
    ADD COLUMN id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY FIRST,
    DROP INDEX enrollment_id_2,
    DROP INDEX enrollment_id,
    DROP COLUMN row_count,
    ADD INDEX (enrollment_id, id);
