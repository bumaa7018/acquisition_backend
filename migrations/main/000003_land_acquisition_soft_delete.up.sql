-- Чөлөөлөлтийг УСТГАХГҮЙ, "устгасан" гэж тэмдэглэдэг болно (soft delete).
--
-- Шалтгаан: land_acquisition-ийг DELETE хийхэд 23 хүснэгт ON DELETE CASCADE-ээр
-- цэвэрлэгддэг (parcel, parcel_holder, compensation, parcel_payment, assets,
-- land_valuation, drone_image гэх мэт) бөгөөд буцаах ямар ч арга байхгүй.
-- Мөн decision_draft.acquisition_id нь SET NULL болж, түүний нэгж талбарын
-- холбоос (decision_draft_parcel) чимээгүй устдаг.
--
-- deleted_at IS NOT NULL мөрийг API-ийн бүх уншилт шүүж хаяна.

ALTER TABLE public.land_acquisition
    ADD COLUMN IF NOT EXISTS deleted_at timestamptz,
    ADD COLUMN IF NOT EXISTS deleted_by text;

-- Уншилт бүр `deleted_at IS NULL`-ээр шүүдэг тул зөвхөн ажиллаж байгаа
-- мөрүүдийг хамарсан хэсэгчилсэн индекс (устгасан мөр индекс хөөхгүй).
CREATE INDEX IF NOT EXISTS idx_land_acquisition_active
    ON public.land_acquisition (id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN public.land_acquisition.deleted_at IS
    'Устгасан хугацаа (soft delete). NULL бол ажиллаж байгаа чөлөөлөлт.';
COMMENT ON COLUMN public.land_acquisition.deleted_by IS
    'Устгасан хэрэглэгчийн ID (soft delete).';

-- Нэгж талбарыг ч мөн "устгасан" гэж тэмдэглэнэ.
--
-- parcel.parcel_id-ийн ГЛОБАЛ UNIQUE constraint нь assets / land_valuation /
-- parcel_status_history / parcel_valuation_submission-ийн FK-ийн бай тул
-- хэсэгчилсэн unique индексээр СОЛИХ БОЛОМЖГҮЙ (FK нь partial индексийг
-- зөвшөөрдөггүй). Тиймээс устгасан нэгж талбар дугаараа "эзэлсээр" үлдэнэ —
-- харин ШИНЭ чөлөөлөлт түүнийг ЭРГҮҮЛЭН ЭЗЭМШИНЭ (insertParcelRow-г үз).
ALTER TABLE public.parcel
    ADD COLUMN IF NOT EXISTS deleted_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_parcel_active
    ON public.parcel (acquisition_id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN public.parcel.deleted_at IS
    'Устгасан хугацаа (чөлөөлөлт soft delete болоход тэмдэглэгдэнэ). NULL бол ажиллаж байгаа.';
