-- =============================================================================
-- Migration: Create Billing tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS billing_accounts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID,
    org_id           UUID,
    plan             VARCHAR(20) NOT NULL DEFAULT 'free',
    balance          DECIMAL(12,4) NOT NULL DEFAULT 0,
    credit           DECIMAL(12,4) NOT NULL DEFAULT 0,
    monthly_budget   DECIMAL(12,4),
    alert_threshold  DECIMAL(5,2) NOT NULL DEFAULT 80.00,
    stripe_customer_id VARCHAR(255),
    payment_method   JSONB,
    status           VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_user ON billing_accounts(user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_org ON billing_accounts(org_id) WHERE org_id IS NOT NULL;

CREATE TRIGGER update_billing_accounts_updated_at
    BEFORE UPDATE ON billing_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS usage_records (
    id                 UUID NOT NULL DEFAULT gen_random_uuid(),
    billing_account_id UUID NOT NULL,
    user_id            UUID NOT NULL,
    service            VARCHAR(50) NOT NULL,
    model_id           UUID,
    input_tokens       BIGINT NOT NULL DEFAULT 0,
    output_tokens      BIGINT NOT NULL DEFAULT 0,
    cost               DECIMAL(10,6) NOT NULL DEFAULT 0,
    metadata           JSONB NOT NULL DEFAULT '{}',
    recorded_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id, recorded_at)
) PARTITION BY RANGE (recorded_at);

-- Create partitions
DO $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        start_date := DATE_TRUNC('month', CURRENT_DATE) + (i || ' months')::INTERVAL;
        end_date := start_date + '1 month'::INTERVAL;
        partition_name := 'usage_records_' || TO_CHAR(start_date, 'YYYY_MM');
        EXECUTE FORMAT('CREATE TABLE IF NOT EXISTS %I PARTITION OF usage_records FOR VALUES FROM (%L) TO (%L)', partition_name, start_date, end_date);
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS idx_usage_account ON usage_records(billing_account_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_user ON usage_records(user_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS billing_invoices (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id UUID NOT NULL REFERENCES billing_accounts(id),
    invoice_number     VARCHAR(50) NOT NULL UNIQUE,
    period_start       TIMESTAMPTZ NOT NULL,
    period_end         TIMESTAMPTZ NOT NULL,
    subtotal           DECIMAL(12,4) NOT NULL,
    tax                DECIMAL(12,4) NOT NULL DEFAULT 0,
    total              DECIMAL(12,4) NOT NULL,
    currency           VARCHAR(3) NOT NULL DEFAULT 'USD',
    status             VARCHAR(20) NOT NULL DEFAULT 'draft',
    payment_method     VARCHAR(50),
    payment_id         VARCHAR(255),
    paid_at            TIMESTAMPTZ,
    line_items         JSONB NOT NULL DEFAULT '[]',
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_account ON billing_invoices(billing_account_id, created_at DESC);

CREATE TRIGGER update_billing_invoices_updated_at
    BEFORE UPDATE ON billing_invoices FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
