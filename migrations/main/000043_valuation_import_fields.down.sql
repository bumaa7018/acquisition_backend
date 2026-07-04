ALTER TABLE land_valuation
  DROP COLUMN IF EXISTS ownership_cert_no,
  DROP COLUMN IF EXISTS appraiser_org_name,
  DROP COLUMN IF EXISTS appraiser_reg_no,
  DROP COLUMN IF EXISTS appraiser_state_reg_no,
  DROP COLUMN IF EXISTS appraiser_director,
  DROP COLUMN IF EXISTS appraiser_license,
  DROP COLUMN IF EXISTS appraiser_address,
  DROP COLUMN IF EXISTS appraiser_contact,
  DROP COLUMN IF EXISTS source_file_name,
  DROP COLUMN IF EXISTS source_file_hash;

ALTER TABLE assets
  DROP COLUMN IF EXISTS unit_price;
