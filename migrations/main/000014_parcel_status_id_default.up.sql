-- parcel_status.id баганад авто-нэмэгддэг sequence + DEFAULT нэмнэ.
--
-- ЯАГААД: parcel_status нь lookup хүснэгт боловч `id` баганад sequence болон
-- DEFAULT байхгүй байсан. Иймд POST /api/v1/parcel-statuses (repo-ийн
-- `INSERT INTO parcel_status (code, name, sort_order) ... RETURNING id`) нь
-- id=NULL болж "null value in column id violates not-null constraint" алдаагаар
-- унадаг байв. Ижил төрлийн acquisition_progress_status хүснэгт нь
-- progress_type_id_seq-тэй (000001 baseline) — parcel_status-д мартагдсаныг нөхнө.

CREATE SEQUENCE IF NOT EXISTS public.parcel_status_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.parcel_status_id_seq OWNED BY public.parcel_status.id;

ALTER TABLE ONLY public.parcel_status
    ALTER COLUMN id SET DEFAULT nextval('public.parcel_status_id_seq'::regclass);

-- Одоо суусан мөрүүдтэй (жишээ: 0..5) давхцахгүйн тулд sequence-г хамгийн их
-- id рүү дэвшүүлнэ. Хүснэгт хоосон бол дараагийн nextval нь 1-ээс эхэлнэ.
SELECT setval(
    'public.parcel_status_id_seq',
    GREATEST((SELECT COALESCE(MAX(id), 0) FROM public.parcel_status), 1),
    (SELECT EXISTS (SELECT 1 FROM public.parcel_status))
);
