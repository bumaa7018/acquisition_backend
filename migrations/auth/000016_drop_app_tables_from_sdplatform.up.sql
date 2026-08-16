-- sdplatform схемд аппын өөрийн ЯМАР Ч хүснэгт үлдээхгүй.
--
-- Эдгээр нь өмнө нь sdplatform дотор үүсгэгдэж байсан ба одоо үндсэн (main)
-- өгөгдлийн санд шилжсэн (migrations/main/000094). Ингэснээр энэ холболтыг
-- жинхэнэ sdplatform DB рүү залгахад бид тэр санд ЮУ Ч бичихгүй — зөвхөн
-- өгөгдсөн хүснэгтүүдийг уншиж/бичнэ.
--
-- Токенууд түр зуурынх тул зөөх шаардлагагүй (хэрэглэгчид дахин нэвтэрнэ).

DROP TABLE IF EXISTS sdplatform.password_reset_tokens;
DROP TABLE IF EXISTS sdplatform.revoked_tokens;
DROP TABLE IF EXISTS sdplatform.user_token_epoch;
