-- Буцаалт: хуучин нэг баганат индексийг сэргээж, шинийг хасна.
-- Дарааллыг эсрэгээр нь хийнэ — эхлээд орлуулагчийг үүсгэж байж шинийг хасна,
-- эс бөгөөс хоёрын хооронд parcel_id-гаар хайх асуулга индексгүй үлдэнэ.
CREATE INDEX IF NOT EXISTS idx_psh_parcel
    ON public.parcel_status_history USING btree (parcel_id);

DROP INDEX IF EXISTS public.idx_psh_parcel_status_date;
