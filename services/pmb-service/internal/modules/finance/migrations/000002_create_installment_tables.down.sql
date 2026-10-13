-- Migration Down: Drop Installment Tables
ALTER TABLE payments DROP COLUMN IF EXISTS installment_id;
DROP TABLE IF EXISTS installments CASCADE;
DROP TABLE IF EXISTS installment_requests CASCADE;
