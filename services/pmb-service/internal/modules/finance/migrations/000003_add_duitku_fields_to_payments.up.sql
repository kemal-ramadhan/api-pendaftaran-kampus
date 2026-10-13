-- 000003_add_duitku_fields_to_payments.up.sql
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS payment_url VARCHAR(500),
    ADD COLUMN IF NOT EXISTS payment_code VARCHAR(50),
    ADD COLUMN IF NOT EXISTS publisher_order_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS settlement_date TIMESTAMP WITH TIME ZONE;
