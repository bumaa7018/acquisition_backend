ALTER TABLE users
    ADD COLUMN IF NOT EXISTS position TEXT NOT NULL DEFAULT '';

UPDATE users
SET position = 'Ахлах мэргэжилтэн'
WHERE username = 'admin'
  AND COALESCE(position, '') = '';

UPDATE users
SET position = 'Ажилтан'
WHERE username = 'employee'
  AND COALESCE(position, '') = '';
