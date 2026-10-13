-- Migration Down: Rollback CR-006 Data
DROP TRIGGER IF EXISTS trg_prevent_application_changes_delete ON application_changes;
DROP TRIGGER IF EXISTS trg_prevent_application_changes_update ON application_changes;
DROP FUNCTION IF EXISTS prevent_application_changes_modification();
DROP TABLE IF EXISTS path_change_requests;
DROP TABLE IF EXISTS application_changes;
ALTER TABLE applications DROP COLUMN IF EXISTS previous_application_id;
ALTER TABLE applications DROP COLUMN IF EXISTS path_change_count_after_payment;
