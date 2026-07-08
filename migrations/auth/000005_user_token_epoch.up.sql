-- Хэрэглэгчийн нэвтрэлтийн "эх цэг": энэ хугацаанаас өмнө олгогдсон бүх токен
-- хүчингүй болно. Нууц үг сэргээх / албадан гаргах үед NOW() болгож тэмдэглэнэ.
ALTER TABLE users ADD COLUMN IF NOT EXISTS tokens_valid_from TIMESTAMPTZ;
