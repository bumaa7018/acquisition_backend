ALTER TABLE public.parcel_status_history
    DROP COLUMN IF EXISTS attachment_url,
    DROP COLUMN IF EXISTS attachment_name;
