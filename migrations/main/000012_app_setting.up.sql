-- Системийн тохиргоо (key → JSON утга).
--
-- Төлөвлөгөөний 2.6.1 ажил: газрын зургийн СУУРЬ зургийн хаягийг гараас
-- тохируулах. Тохируулаагүй бол дэлгэц өөрийн үндсэн суурь зургаа хэрэглэнэ
-- (мөр огт байхгүй = үндсэн суурь зураг).
--
-- Яагаад key/value: цаашид өөр нэг цонхны тохиргоо нэмэхэд хүснэгт үүсгэхгүй
-- байхаар. Утга нь jsonb тул тохиргоо тус бүр өөрийн бүтэцтэй байж болно
-- (basemap → {type, url, layer, max_zoom, extent, attribution}).
--
-- Уншилт нэвтэрсэн бүх хэрэглэгчид, БИЧИХ нь admin:update эрхтэйд (router).

CREATE TABLE IF NOT EXISTS public.app_setting (
    key text NOT NULL,
    value jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'app_setting_pkey') THEN
        ALTER TABLE ONLY public.app_setting
            ADD CONSTRAINT app_setting_pkey PRIMARY KEY (key);
    END IF;
END $$;

COMMENT ON TABLE public.app_setting IS
    'Системийн тохиргоо key→jsonb (ж: basemap — газрын зургийн суурь зургийн хаяг).';
