ALTER TABLE public.parcel_status DROP CONSTRAINT IF EXISTS parcel_status_color_hex;
ALTER TABLE public.parcel_status DROP COLUMN IF EXISTS color;
