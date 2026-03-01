-- ============================================================
-- ROLLBACK BILLING & FINANCIAL MANAGEMENT SCHEMA
-- ============================================================

-- Drop tables (cascade will handle foreign key dependencies)
DROP TABLE IF EXISTS wallet_transactions CASCADE;
DROP TABLE IF EXISTS topup_requests CASCADE;
DROP TABLE IF EXISTS payment_gateways CASCADE;

-- Remove wallet_balance column from customers
ALTER TABLE customers 
DROP CONSTRAINT IF EXISTS check_wallet_balance_non_negative;

ALTER TABLE customers 
DROP COLUMN IF EXISTS wallet_balance;

-- Drop enums
DROP TYPE IF EXISTS transaction_type CASCADE;
DROP TYPE IF EXISTS topup_status CASCADE;
