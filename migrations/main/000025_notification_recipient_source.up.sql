-- Мэдэгдлийг ХҮЛЭЭН АВАГЧИЙН ЭХ СУРВАЛЖААР тусгаарлана.
--
-- ЯАГААД: 000024-өөс хойш нэвтрэх бүртгэл ХОЁР санд байна — sdplatform.sd_user
-- (auth) ба appdb.valuation_org_employee (valuation_org). Хоёулаа өөрийн
-- serial дараалалтай тул recipient_id = 3 нь ХОЁР өөр хүнийг заана.
-- Эх сурвалжгүйгээр /notifications нь нөгөө хүний мэдэгдлийг харуулна.
--
-- Анхдагч нь 'auth' — одоо байгаа бүх мөр дотоод ажилтных.
ALTER TABLE public.notification
    ADD COLUMN IF NOT EXISTS recipient_source character varying(16)
        DEFAULT 'auth'::character varying NOT NULL;

-- Уншилт ҮРГЭЛЖ (recipient_source, recipient_id)-ээр шүүгдэнэ.
CREATE INDEX IF NOT EXISTS idx_notification_recipient_source
    ON public.notification USING btree (recipient_source, recipient_id, created_at DESC);

COMMENT ON COLUMN public.notification.recipient_source IS
    'Хүлээн авагчийн эх сурвалж: auth (sdplatform.sd_user) | valuation_org (valuation_org_employee). recipient_id зөвхөн үүнтэй хосоороо утгатай.';
