ALTER TABLE ONLY public.parcel_status ALTER COLUMN id DROP DEFAULT;

DROP SEQUENCE IF EXISTS public.parcel_status_id_seq;
