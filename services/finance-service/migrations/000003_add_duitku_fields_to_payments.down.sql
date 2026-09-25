-- 000003_add_duitku_fields_to_payments.down.sql
ALTER TABLE payments
    DROP COLUMN IF EXISTS payment_url,
    DROP COLUMN IF EXISTS payment_code,
    DROP COLUMN IF EXISTS publisher_order_id,
    DROP COLUMN IF EXISTS settlement_date;
