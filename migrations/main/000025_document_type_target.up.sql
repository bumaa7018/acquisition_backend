ALTER TABLE document_type ADD COLUMN IF NOT EXISTS target VARCHAR(20) NOT NULL DEFAULT 'both';

-- Нэгж талбарын баримт бичгүүд
UPDATE document_type SET target = 'parcel' WHERE type IN (
  'photo_valuation', 'tax_receipt', 'engineering_blueprint', 'id_card',
  'company_certificate', 'land_certificate', 'cadastral_map', 'bank_statement',
  'power_of_attorney', 'inheritance_agreement', 'death_certificate',
  'disability_certificate', 'signature_guarantee'
);

-- Чөлөөлөлтийн баримт бичгүүд
UPDATE document_type SET target = 'acquisition' WHERE type IN (
  'control_card', 'land_handover_act', 'payment_table', 'decree_draft',
  'settlement_act', 'monitoring_report', 'contract_closing_act'
);

-- БУСАД — аль алинд харагдана
UPDATE document_type SET target = 'both' WHERE type = 'other';
