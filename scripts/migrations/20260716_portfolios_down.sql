-- Roll back the transaction-ledger portfolio schema.
-- Destructive by design: take a backup before running in production.
BEGIN;
DROP TABLE IF EXISTS portfolio_positions;
DROP TABLE IF EXISTS portfolio_transactions;
DROP TABLE IF EXISTS portfolios;
COMMIT;
