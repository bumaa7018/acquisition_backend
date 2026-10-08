-- «Хронос»: олголтын баримтыг (payment_stage > 0) бүх баримтаас шүүж уншихгүйн
-- тулд хэсэгчилсэн индекс — баримт олон болсон ч зөвхөн олголтын мөрүүдийг авна.
CREATE INDEX IF NOT EXISTS idx_document_payment
    ON public.document (id)
    WHERE payment_stage > 0 AND payment_date IS NOT NULL;
