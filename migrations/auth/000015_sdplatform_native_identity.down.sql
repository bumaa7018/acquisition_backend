-- Буцаалт: энэ миграц нь таних тэмдгийг uuid → int4 болгож, эрхийн загварыг
-- (нөөц:үйлдэл) → (роль=нөөц, эрх=үйлдэл) руу задалсан ба public дахь HR
-- лавлахыг устгасан. Эдгээрийг АЛДАГДАЛГҮЙ буцаах боломжгүй (хуучин uuid-ууд
-- хадгалагдаагүй) тул энд зөвхөн аппын өөрийн туслах хүснэгтүүдийг арилгана.
--
-- Бүрэн буцаалт шаардлагатай бол миграцын өмнөх нөөц хуулбараас сэргээнэ.

DROP TABLE IF EXISTS sdplatform.user_token_epoch;
DROP TABLE IF EXISTS sdplatform.password_reset_tokens;
DROP TABLE IF EXISTS sdplatform.revoked_tokens;
