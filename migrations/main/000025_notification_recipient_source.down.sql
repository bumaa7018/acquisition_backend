DROP INDEX IF EXISTS public.idx_notification_recipient_source;

ALTER TABLE public.notification
    DROP COLUMN IF EXISTS recipient_source;
