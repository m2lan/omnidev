DROP TRIGGER IF EXISTS update_billing_invoices_updated_at ON billing_invoices;
DROP TRIGGER IF EXISTS update_billing_accounts_updated_at ON billing_accounts;
DROP TABLE IF EXISTS billing_invoices CASCADE;
DROP TABLE IF EXISTS usage_records CASCADE;
DROP TABLE IF EXISTS billing_accounts CASCADE;
