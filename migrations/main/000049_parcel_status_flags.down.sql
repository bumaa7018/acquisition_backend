ALTER TABLE public.parcel_status
    DROP COLUMN IF EXISTS requires_reason,
    DROP COLUMN IF EXISTS is_released,
    DROP COLUMN IF EXISTS is_valuation_stage,
    DROP COLUMN IF EXISTS locks_progress,
    DROP COLUMN IF EXISTS is_final;
