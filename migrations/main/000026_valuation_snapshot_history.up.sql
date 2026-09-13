CREATE TABLE IF NOT EXISTS public.parcel_valuation_snapshot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL,
    submission_id uuid,
    note text DEFAULT ''::text NOT NULL,
    cancelled_by text DEFAULT ''::text NOT NULL,
    cancelled_at timestamp with time zone DEFAULT now() NOT NULL,
    submitted_by text DEFAULT ''::text NOT NULL,
    submitted_at timestamp with time zone,
    reviewed_by text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    land_valuation jsonb DEFAULT '{}'::jsonb NOT NULL,
    assets jsonb DEFAULT '[]'::jsonb NOT NULL,
    compensations jsonb DEFAULT '[]'::jsonb NOT NULL,
    total_amount numeric(20,2) DEFAULT 0 NOT NULL,
    CONSTRAINT parcel_valuation_snapshot_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_pvsnap_acq_parcel
    ON public.parcel_valuation_snapshot (acquisition_id, parcel_id, valuation_type, cancelled_at DESC);

COMMENT ON TABLE public.parcel_valuation_snapshot IS
    'Баталгаажсан үнэлгээг хүчингүй болгоход тухайн үеийн газар/хөрөнгө/олговрын snapshot хадгална.';
