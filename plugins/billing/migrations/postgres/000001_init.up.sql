-- ============================================================
-- BILLING & FINANCIAL MANAGEMENT SCHEMA
-- ============================================================

-- Enums
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'topup_status') THEN
        CREATE TYPE topup_status AS ENUM ('PENDING', 'SUCCESS', 'EXPIRED', 'FAILED', 'CANCELLED');
    END IF;
END$$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transaction_type') THEN
        CREATE TYPE transaction_type AS ENUM ('TOPUP', 'PURCHASE', 'REFUND', 'RENEWAL', 'ADMIN_ADJUSTMENT');
    END IF;
END$$;

-- ============================================================
-- ALTER customers table to add wallet_balance
-- ============================================================
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'wallet_balance'
    ) THEN
        ALTER TABLE customers 
        ADD COLUMN wallet_balance DECIMAL(15, 2) DEFAULT 0.00 NOT NULL;
    END IF;
END$$;

-- Add check constraint: wallet_balance tidak boleh negatif (bisa diubah jika allow negative)
ALTER TABLE customers 
DROP CONSTRAINT IF EXISTS check_wallet_balance_non_negative;

ALTER TABLE customers 
ADD CONSTRAINT check_wallet_balance_non_negative 
CHECK (wallet_balance >= 0);

-- ============================================================
-- TABLE: payment_gateways
-- Mendaftarkan payment providers (Midtrans, Xendit, Manual, dll)
-- ============================================================
CREATE TABLE IF NOT EXISTS payment_gateways (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,              -- 'midtrans', 'xendit', 'manual'
    gateway_type VARCHAR(50) NOT NULL,              -- 'AUTOMATIC', 'MANUAL'
    is_active BOOLEAN DEFAULT TRUE,
    config JSONB,                                   -- API keys, merchant IDs, callback URLs, etc
    fee_percentage DECIMAL(5, 2) DEFAULT 0.00,      -- Fee %
    fee_fixed DECIMAL(15, 2) DEFAULT 0.00,          -- Fee fixed amount
    min_amount DECIMAL(15, 2) DEFAULT 10000.00,     -- Minimum topup
    max_amount DECIMAL(15, 2) DEFAULT 10000000.00,  -- Maximum topup
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- TABLE: topup_requests
-- Tracking setiap request top-up dari customer
-- ============================================================
CREATE TABLE IF NOT EXISTS topup_requests (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    gateway_id UUID NOT NULL REFERENCES payment_gateways(id),
    
    amount DECIMAL(15, 2) NOT NULL,                 -- Jumlah yang diminta customer
    fee DECIMAL(15, 2) DEFAULT 0.00,                -- Fee yang dikenakan
    total_paid DECIMAL(15, 2) NOT NULL,             -- amount + fee
    
    external_id VARCHAR(255) UNIQUE,                -- Order ID dari payment gateway
    payment_url TEXT,                               -- Snap URL / Checkout URL
    payment_method VARCHAR(100),                    -- 'qris', 'bank_transfer', 'gopay', etc
    payment_channel VARCHAR(100),                   -- Additional detail
    
    status topup_status DEFAULT 'PENDING' NOT NULL,
    
    paid_at TIMESTAMPTZ,                            -- Waktu payment confirmed
    expired_at TIMESTAMPTZ,                         -- Expiry time from gateway
    
    webhook_data JSONB,                             -- Raw webhook payload for audit
    notes TEXT,                                     -- Admin notes (for manual confirmation)
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for topup_requests
CREATE INDEX IF NOT EXISTS idx_topup_requests_customer_id ON topup_requests(customer_id);
CREATE INDEX IF NOT EXISTS idx_topup_requests_gateway_id ON topup_requests(gateway_id);
CREATE INDEX IF NOT EXISTS idx_topup_requests_status ON topup_requests(status);
CREATE INDEX IF NOT EXISTS idx_topup_requests_external_id ON topup_requests(external_id);
CREATE INDEX IF NOT EXISTS idx_topup_requests_created_at ON topup_requests(created_at DESC);

-- ============================================================
-- TABLE: wallet_transactions
-- Audit trail untuk semua mutasi saldo (APPEND-ONLY)
-- ============================================================
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    
    amount DECIMAL(15, 2) NOT NULL,                 -- Positif (masuk), Negatif (keluar)
    balance_before DECIMAL(15, 2) NOT NULL,
    balance_after DECIMAL(15, 2) NOT NULL,
    
    type transaction_type NOT NULL,
    reference_id UUID,                              -- ID dari topup_requests, containers, dll
    reference_type VARCHAR(100),                    -- 'topup_request', 'container', 'manual'
    
    description TEXT,                               -- Human-readable description
    metadata JSONB,                                 -- Additional context
    
    created_by_admin_id UUID,                       -- NULL jika otomatis, filled jika admin adjustment
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for wallet_transactions
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_customer_id ON wallet_transactions(customer_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(type);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_reference_id ON wallet_transactions(reference_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_created_at ON wallet_transactions(created_at DESC);

-- ============================================================
-- SEED: Initial Payment Gateways
-- ============================================================
INSERT INTO payment_gateways (id, name, slug, gateway_type, is_active, config, fee_percentage, fee_fixed, min_amount, max_amount, display_order)
VALUES 
    ('018e0000-0000-7000-8000-000000000001'::UUID, 'Manual Transfer', 'manual', 'MANUAL', true, '{"bank_name": "BCA", "account_number": "1234567890", "account_name": "PT Contoh"}', 0.00, 0.00, 10000.00, 50000000.00, 1),
    ('018e0000-0000-7000-8000-000000000002'::UUID, 'Midtrans (QRIS/VA/E-wallet)', 'midtrans', 'AUTOMATIC', false, '{"server_key": "", "client_key": "", "is_production": false}', 2.50, 0.00, 10000.00, 10000000.00, 2),
    ('018e0000-0000-7000-8000-000000000003'::UUID, 'Xendit (VA/E-wallet)', 'xendit', 'AUTOMATIC', false, '{"api_key": "", "callback_token": "", "is_production": false}', 2.00, 0.00, 10000.00, 10000000.00, 3)
ON CONFLICT (slug) DO NOTHING;
