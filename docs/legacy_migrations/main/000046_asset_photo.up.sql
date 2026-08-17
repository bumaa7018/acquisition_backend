-- Хөрөнгө бүрийн зургуудыг нэгтгэсэн PDF-ийг хадгалах талбарууд.
ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS photo_pdf_url  TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS photo_pdf_name TEXT NOT NULL DEFAULT '';
