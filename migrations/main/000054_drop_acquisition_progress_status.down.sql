COMMENT ON COLUMN public.land_acquisition.status IS NULL;
COMMENT ON COLUMN public.acquisition_workflow.from_status IS NULL;
COMMENT ON COLUMN public.acquisition_workflow.to_status IS NULL;
COMMENT ON COLUMN public.acquisition_progress.from_status IS NULL;
COMMENT ON COLUMN public.acquisition_progress.to_status IS NULL;

CREATE TABLE IF NOT EXISTS public.acquisition_progress_status (
    id integer NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    sort_order integer
);
CREATE SEQUENCE IF NOT EXISTS public.progress_type_id_seq AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.progress_type_id_seq OWNED BY public.acquisition_progress_status.id;
ALTER TABLE ONLY public.acquisition_progress_status ALTER COLUMN id SET DEFAULT nextval('public.progress_type_id_seq'::regclass);
ALTER TABLE ONLY public.acquisition_progress_status ADD CONSTRAINT progress_type_pkey PRIMARY KEY (id);
