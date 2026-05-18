-- P3a Plugin Signature Support Migration
-- Adds signature and vendor metadata fields to sk_plugins table

-- Step 1: Add new columns (nullable, safe for existing data)
ALTER TABLE sk_plugins ADD COLUMN vendor VARCHAR(128) DEFAULT NULL;
ALTER TABLE sk_plugins ADD COLUMN vendor_url VARCHAR(512) DEFAULT NULL;
ALTER TABLE sk_plugins ADD COLUMN signature_json TEXT DEFAULT NULL;

-- Step 2: Create index on vendor for faster lookups (optional)
CREATE INDEX idx_plugin_vendor ON sk_plugins(vendor);

-- Verification query to check migration success:
-- SELECT COUNT(*) as total_plugins, 
--        COUNT(signature_json) as signed_plugins,
--        COUNT(vendor) as vendor_plugins
-- FROM sk_plugins;

-- Rollback script (if needed):
-- ALTER TABLE sk_plugins DROP COLUMN vendor;
-- ALTER TABLE sk_plugins DROP COLUMN vendor_url;
-- ALTER TABLE sk_plugins DROP COLUMN signature_json;
-- DROP INDEX idx_plugin_vendor ON sk_plugins;
