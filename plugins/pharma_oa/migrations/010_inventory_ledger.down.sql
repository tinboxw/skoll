DROP TABLE IF EXISTS {{table:stock_balances}};
DROP TABLE IF EXISTS {{table:stock_ledger}};
DROP TABLE IF EXISTS {{table:inventory_lots}};
ALTER TABLE {{table:purchase_inbounds}} DROP COLUMN request_hash;
