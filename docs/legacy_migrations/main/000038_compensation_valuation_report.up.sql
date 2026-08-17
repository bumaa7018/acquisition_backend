ALTER TABLE compensation
    ADD COLUMN IF NOT EXISTS valuation_report_url  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS valuation_report_name TEXT NOT NULL DEFAULT '';
