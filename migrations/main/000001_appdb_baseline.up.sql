-- appdb (main) — БҮХ бүтэц нэг baseline файлд.
--
-- Энэ файл нь хуучин 96 миграцын ЭЦСИЙН үр дүн: холбогдсон өгөгдлийн сангийн
-- бодит бүтцийг (pg_dump --schema-only) шууд авч, идемпотент болгосон.
-- Хуучин файлууд migrations/main/_archive/-д хадгалагдсан (түүх, down зам).
--
-- ХЭРЭГЛЭХ ЖУРАМ:
--   * ШИНЭ өгөгдлийн сан — энэ файл ажиллаж бүх бүтцийг үүсгэнэ.
--   * БАЙГАА appdb (version 96) — энэ файлыг АЖИЛЛУУЛАХГҮЙ, зөвхөн тамга дарна:
--       migrate -path migrations/main -database "$MAIN_DATABASE_URL" force 1
--   * Шинэ өөрчлөлт 000002-оос үргэлжилнэ.
--
-- ЖИЧ: appdb нь АППЫН ӨӨРИЙН сан тул энд өгөгдөл нэмэх/өөрчлөх нь
-- зөвшөөрөгдөнө (authdb/sdplatform-ын дүрэм үүнд хамаарахгүй). Гэхдээ энэ
-- baseline нь зөвхөн бүтэц агуулна — өгөгдөл seeder-т.

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- ХАМГААЛАЛТ: baseline зөвхөн ХООСОН санд ажиллана.
--
-- Доорх 146 ADD CONSTRAINT нь IF NOT EXISTS дэмждэггүй тул аль хэдийн бүтэцтэй
-- санд энэ файлыг ажиллуулбал "column ... does not exist" буюу "multiple
-- primary keys" маягийн ойлгомжгүй алдаа өгөөд schema_migrations нь dirty=1
-- болж үлддэг. Тийм санд ЖУРАМ нь baseline-ыг ажиллуулах биш, тамга дарах:
--     migrate -path migrations/main -database "$MAIN_DATABASE_URL" force 1
-- (Апп өөрөө асахдаа үүнийг автоматаар хийнэ — cmd/api/migrate.go).
--
DO $$
BEGIN
    IF to_regclass('public.parcel') IS NOT NULL THEN
        RAISE EXCEPTION 'appdb-д аппын бүтэц аль хэдийн байна: baseline (000001) ажиллуулж болохгүй'
            USING HINT = 'Тамга дарна уу: migrate -path migrations/main -database "$MAIN_DATABASE_URL" force 1';
    END IF;
END $$;

--
-- Name: base; Type: SCHEMA; Schema: -; Owner: -
--

--
-- Name: codelists; Type: SCHEMA; Schema: -; Owner: -
--

--
-- Name: sdplatform; Type: SCHEMA; Schema: -; Owner: -
--

--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: acquisition_category; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.acquisition_category (
    id integer NOT NULL,
    name character varying(200) NOT NULL,
    parent_id integer,
    sort_order integer DEFAULT 0
);

--
-- Name: acquisition_category_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.acquisition_category_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: acquisition_category_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.acquisition_category_id_seq OWNED BY public.acquisition_category.id;

--
-- Name: acquisition_document; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.acquisition_document (
    acquisition_id uuid NOT NULL,
    document_id uuid NOT NULL
);

--
-- Name: acquisition_progress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.acquisition_progress (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    acquisition_id uuid NOT NULL,
    from_status integer NOT NULL,
    to_status integer NOT NULL,
    note text,
    changed_by text DEFAULT 'system'::text NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: acquisition_progress_status; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.acquisition_progress_status (
    id integer NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    sort_order integer
);

--
-- Name: acquisition_workflow; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.acquisition_workflow (
    id integer NOT NULL,
    from_status integer,
    to_status integer NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: acquisition_workflow_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.acquisition_workflow_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: acquisition_workflow_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.acquisition_workflow_id_seq OWNED BY public.acquisition_workflow.id;

--
-- Name: asset_calc_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.asset_calc_type (
    id integer NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    default_unit text DEFAULT ''::text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    grp text DEFAULT ''::text NOT NULL
);

--
-- Name: asset_calc_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.asset_calc_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: asset_calc_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.asset_calc_type_id_seq OWNED BY public.asset_calc_type.id;

--
-- Name: asset_calculation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.asset_calculation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_id uuid NOT NULL,
    calc_type_id integer NOT NULL,
    unit text DEFAULT ''::text NOT NULL,
    value numeric(18,4) DEFAULT 0 NOT NULL,
    sort_index integer DEFAULT 0 NOT NULL
);

--
-- Name: asset_spec; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.asset_spec (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_id uuid NOT NULL,
    spec_type_id integer NOT NULL,
    value text DEFAULT ''::text NOT NULL
);

--
-- Name: asset_spec_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.asset_spec_type (
    id integer NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: asset_spec_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.asset_spec_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: asset_spec_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.asset_spec_type_id_seq OWNED BY public.asset_spec_type.id;

--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.assets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    asset_number text DEFAULT ''::text NOT NULL,
    asset_type character varying(20) DEFAULT 'real_state'::character varying NOT NULL,
    asset_name text DEFAULT ''::text NOT NULL,
    floor_count integer DEFAULT 0 NOT NULL,
    area_m2 numeric(18,4) DEFAULT 0 NOT NULL,
    owner_name text DEFAULT ''::text NOT NULL,
    address text DEFAULT ''::text NOT NULL,
    notes text DEFAULT ''::text NOT NULL,
    geometry public.geometry(Polygon,4326),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unit text DEFAULT ''::text NOT NULL,
    capacity text DEFAULT ''::text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    unit_price numeric(18,4) DEFAULT 0 NOT NULL,
    photo_pdf_url text DEFAULT ''::text NOT NULL,
    photo_pdf_name text DEFAULT ''::text NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL
);

--
-- Name: au1; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.au1 (
    code character varying(3) NOT NULL,
    name character varying(50) NOT NULL,
    area_m2 numeric,
    geometry public.geometry(MultiPolygon,4326),
    name_en character varying(100)
);

--
-- Name: au2; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.au2 (
    code character varying(5) NOT NULL,
    name character varying(50) NOT NULL,
    au1_code character varying(3),
    area_m2 numeric,
    geometry public.geometry(Polygon,4326),
    name_en character varying(100)
);

--
-- Name: au3; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.au3 (
    code character varying(8),
    name character varying(50),
    au1_code character varying(3),
    au2_code character varying(5),
    area_m2 numeric,
    geometry public.geometry(MultiPolygon,4326),
    name_desc text,
    old_code character varying(8),
    name_en character varying(100)
);

--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.audit_log (
    id uuid NOT NULL,
    actor_id integer,
    actor_name text DEFAULT ''::text NOT NULL,
    actor_position text DEFAULT ''::text NOT NULL,
    actor_roles jsonb DEFAULT '[]'::jsonb NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text DEFAULT ''::text NOT NULL,
    acquisition_id uuid,
    parcel_id uuid,
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    ip_address text DEFAULT ''::text NOT NULL,
    user_agent text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: authorized_representatives; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.authorized_representatives (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id text DEFAULT ''::text NOT NULL,
    last_name text DEFAULT ''::text NOT NULL,
    first_name text DEFAULT ''::text NOT NULL,
    register_no text DEFAULT ''::text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    address text DEFAULT ''::text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL
);

--
-- Name: client_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.client_request (
    id uuid NOT NULL,
    method text NOT NULL,
    host text NOT NULL,
    path text NOT NULL,
    request_body text,
    request_response text,
    status_code integer,
    error text,
    duration_ms bigint,
    requested_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: compensation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.compensation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(50) DEFAULT ''::character varying NOT NULL,
    asset_id uuid,
    compensation_type character varying(20) DEFAULT 'cash'::character varying NOT NULL,
    coverage_percent numeric(5,2) DEFAULT 100 NOT NULL,
    amount numeric(20,2) DEFAULT 0 NOT NULL,
    compensation_date date,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    target_type character varying(20) DEFAULT 'asset'::character varying NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    review_note text DEFAULT ''::text NOT NULL,
    reviewed_by text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    valuation_report_url text DEFAULT ''::text NOT NULL,
    valuation_report_name text DEFAULT ''::text NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL
);

--
-- Name: compensation_grant; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.compensation_grant (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    amount numeric(20,2) DEFAULT 0 NOT NULL,
    grant_date date,
    note text DEFAULT ''::text NOT NULL,
    land_area_m2 numeric(15,2) DEFAULT 0 NOT NULL,
    land_price numeric(20,2) DEFAULT 0 NOT NULL,
    land_location text DEFAULT ''::text NOT NULL,
    land_purpose text DEFAULT ''::text NOT NULL,
    land_use_type text DEFAULT ''::text NOT NULL,
    parcel_number character varying(50) DEFAULT ''::character varying NOT NULL,
    decree_number character varying(100) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    compensation_id uuid
);

--
-- Name: compensation_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.compensation_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    compensation_id uuid NOT NULL,
    acquisition_id uuid NOT NULL,
    target_type text NOT NULL,
    parcel_id text DEFAULT ''::text NOT NULL,
    asset_id uuid,
    compensation_type text NOT NULL,
    coverage_percent numeric(10,4) DEFAULT 0 NOT NULL,
    amount numeric(20,2) DEFAULT 0 NOT NULL,
    compensation_date date,
    note text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'rejected'::text NOT NULL,
    review_note text DEFAULT ''::text NOT NULL,
    reviewed_by text DEFAULT ''::text NOT NULL,
    reviewed_at timestamp with time zone,
    archived_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: construction_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.construction_type (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: construction_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.construction_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: construction_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.construction_type_id_seq OWNED BY public.construction_type.id;

--
-- Name: decision_budget; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_budget (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: decision_budget_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.decision_budget_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: decision_budget_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.decision_budget_id_seq OWNED BY public.decision_budget.id;

--
-- Name: decision_draft; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_draft (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    proposal_no text NOT NULL,
    decree_number character varying(100) DEFAULT ''::character varying NOT NULL,
    decision_date date,
    location text DEFAULT ''::text NOT NULL,
    duration_year integer,
    status smallint DEFAULT 1 NOT NULL,
    acquisition_id uuid,
    work_type_id integer,
    budget_id integer,
    confirmed_at timestamp with time zone,
    confirmed_by text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    current_progress_history_id uuid
);

--
-- Name: decision_draft_funding_source; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_draft_funding_source (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    decision_draft_id uuid NOT NULL,
    funding_source_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    amount numeric(18,2) DEFAULT 0 NOT NULL
);

--
-- Name: decision_draft_parcel; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_draft_parcel (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    decision_draft_id uuid NOT NULL,
    parcel_id uuid NOT NULL,
    linked_at timestamp with time zone DEFAULT now() NOT NULL,
    linked_by text DEFAULT ''::text NOT NULL,
    removed_at timestamp with time zone,
    removed_by text DEFAULT ''::text NOT NULL,
    funding_link_id uuid
);

--
-- Name: decision_draft_progress_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_draft_progress_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    decision_draft_id uuid NOT NULL,
    progress_type character varying(30) NOT NULL,
    recipient text DEFAULT ''::text NOT NULL,
    progress_date date NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT chk_ddph_progress_type CHECK (((progress_type)::text = ANY ((ARRAY['reviewing'::character varying, 'confirming'::character varying])::text[])))
);

--
-- Name: decision_work_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.decision_work_type (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: decision_work_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.decision_work_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: decision_work_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.decision_work_type_id_seq OWNED BY public.decision_work_type.id;

--
-- Name: document; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.document (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    file_type text NOT NULL,
    file_url text NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    note text,
    uploaded_by text DEFAULT 'system'::text NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    document_type_id integer,
    source_doc_id text,
    source_ftp_id text,
    source_doc_code character varying(20)
);

--
-- Name: document_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.document_type (
    id integer NOT NULL,
    type character varying(100) NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_by character varying(255),
    target character varying(20) DEFAULT 'both'::character varying NOT NULL
);

--
-- Name: document_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.document_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: document_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.document_type_id_seq OWNED BY public.document_type.id;

--
-- Name: drone_image; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.drone_image (
    id uuid NOT NULL,
    acquisition_id uuid NOT NULL,
    file_name text NOT NULL,
    original_name text NOT NULL,
    content_type text DEFAULT ''::text NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    min_x double precision,
    min_y double precision,
    max_x double precision,
    max_y double precision,
    published boolean DEFAULT false NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    uploaded_by integer,
    file_url text DEFAULT ''::text NOT NULL
);

--
-- Name: funding_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.funding_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid,
    organization_name text NOT NULL,
    source_type text NOT NULL,
    amount numeric(20,2),
    currency text DEFAULT 'MNT'::text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL
);

--
-- Name: land_acquisition; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_acquisition (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    plan_id uuid,
    plan_code character varying(100) NOT NULL,
    plan_name text,
    plan_geom public.geometry(Polygon,4326),
    plan_area_m2 numeric(18,4),
    geometry public.geometry(Polygon,4326),
    area_m2 numeric(18,4),
    status smallint DEFAULT 1 NOT NULL,
    start_date date,
    end_date date,
    acquisition_name text DEFAULT ''::text NOT NULL,
    implementing_org text,
    reason text,
    responsible_org text,
    funding_source text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255),
    construction_type_id integer,
    decree_number character varying(100) DEFAULT ''::character varying NOT NULL,
    decree_date date,
    general_category_id integer,
    sub_category_id integer,
    professional_org_id integer
);

--
-- Name: land_acquisition_assignee; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_acquisition_assignee (
    acquisition_id uuid NOT NULL,
    user_id integer NOT NULL,
    user_name text DEFAULT ''::text NOT NULL,
    user_position text DEFAULT ''::text NOT NULL,
    assigned_by integer NOT NULL,
    assigned_by_name text DEFAULT ''::text NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: land_acquisition_au; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_acquisition_au (
    land_acquisition_id uuid NOT NULL,
    au1_code character varying(10),
    au1_name character varying(100),
    au2_code character varying(10),
    au2_name character varying(100),
    au3_code character varying(10) NOT NULL,
    au3_name character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255)
);

--
-- Name: land_acquisition_boundary_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_acquisition_boundary_history (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    land_acquisition_id uuid NOT NULL,
    old_geometry public.geometry(Polygon,4326),
    new_geometry public.geometry(Polygon,4326) NOT NULL,
    changed_by character varying(255),
    changed_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: land_acquisition_document; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_acquisition_document (
    land_acquisition_id uuid NOT NULL,
    document_id uuid NOT NULL
);

--
-- Name: land_valuation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.land_valuation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    land_area_m2 numeric(18,4) DEFAULT 0 NOT NULL,
    base_price_per_m2 numeric(18,4) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ownership_cert_no text DEFAULT ''::text NOT NULL,
    appraiser_org_name text DEFAULT ''::text NOT NULL,
    appraiser_reg_no text DEFAULT ''::text NOT NULL,
    appraiser_state_reg_no text DEFAULT ''::text NOT NULL,
    appraiser_director text DEFAULT ''::text NOT NULL,
    appraiser_license text DEFAULT ''::text NOT NULL,
    appraiser_address text DEFAULT ''::text NOT NULL,
    appraiser_contact text DEFAULT ''::text NOT NULL,
    source_file_name text DEFAULT ''::text NOT NULL,
    source_file_hash text DEFAULT ''::text NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL
);

--
-- Name: notification; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.notification (
    id uuid NOT NULL,
    recipient_id integer NOT NULL,
    actor_id integer,
    actor_name text DEFAULT ''::text NOT NULL,
    type text NOT NULL,
    title text NOT NULL,
    body text DEFAULT ''::text NOT NULL,
    resource_type text DEFAULT ''::text NOT NULL,
    resource_id text DEFAULT ''::text NOT NULL,
    acquisition_id uuid,
    parcel_id uuid,
    is_read boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    read_at timestamp with time zone
);

--
-- Name: parcel; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id character varying(12) NOT NULL,
    old_parcel_id character varying(14),
    acquisition_id uuid NOT NULL,
    valid_from date,
    valid_till date,
    au1_code character varying(10),
    au2_code character varying(10),
    au3_code character varying(10),
    geometry public.geometry(Polygon,4326),
    acquisition_geom public.geometry(Polygon,4326),
    acquisition_area_m2 numeric(18,4),
    right_type integer,
    landuse character varying(100),
    area_m2 numeric(18,4),
    compensation_paid boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255),
    remaining_area_m2 numeric(15,2),
    remaining_geom public.geometry(Polygon,4326),
    db_changed boolean DEFAULT false NOT NULL,
    changed_parcel_id character varying(50),
    status integer DEFAULT 0 NOT NULL,
    independent_org_id integer,
    selected_valuation_type text,
    landuse_name text,
    address_khashaa text,
    address_streetname text,
    property_no character varying(100)
);

--
-- Name: parcel_court_decision; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_court_decision (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    source_id text NOT NULL,
    app_no text,
    start_period text,
    end_period text,
    court_status text,
    status_name text,
    court_decision_no text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: parcel_detail; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_detail (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid,
    acquisition_id uuid,
    right_type integer,
    holder_id uuid,
    app_no character varying(100),
    app_timestamps timestamp with time zone,
    decision_no character varying(100),
    decision_date date,
    contract_no character varying(100),
    contract_date date,
    certificate_no character varying(100),
    certificate_date date,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255),
    valuation_zone character varying(200),
    base_price_per_ha numeric(18,4),
    auction_coeff numeric(10,6),
    auction_price numeric(18,4),
    app_type text,
    app_status text,
    contract_begin date,
    contract_end date,
    contract_property_no character varying(100),
    contract_status text,
    record_no character varying(100),
    record_date date,
    record_certificate_no character varying(100),
    record_status text
);

--
-- Name: parcel_document; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_document (
    parcel_id uuid NOT NULL,
    document_id uuid NOT NULL
);

--
-- Name: parcel_holder; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_holder (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    holder_id uuid NOT NULL,
    app_no character varying(100),
    main_applicant boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255),
    holder_role character varying(20) DEFAULT 'holder'::character varying NOT NULL,
    payment_recipient boolean DEFAULT false NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    CONSTRAINT parcel_holder_holder_role_check CHECK (((holder_role)::text = ANY ((ARRAY['holder'::character varying, 'representative'::character varying])::text[])))
);

--
-- Name: parcel_invoice; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_invoice (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    invoice_no text NOT NULL,
    description text,
    amount numeric(20,2),
    paid_amount numeric(20,2),
    status_id integer,
    status_name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: parcel_monitoring; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_monitoring (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid NOT NULL,
    monitoring_id text NOT NULL,
    page_no text,
    status_name text,
    company_name text,
    source_created_at text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: parcel_mortgage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_mortgage (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    acquisition_id uuid,
    source_id text NOT NULL,
    source_parent_id text,
    app_no text,
    start_mortgage_period text,
    end_mortgage_period text,
    loan_contract_no text,
    mortgage_contract_no text,
    loan_contract_amount numeric(20,2),
    monetary_unit_value numeric(20,2),
    unit_type text,
    mortgage_type text,
    status_id text,
    status_name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: parcel_payment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_payment (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    parcel_id uuid NOT NULL,
    amount numeric(15,2) DEFAULT 0 NOT NULL,
    currency character varying(10) DEFAULT 'MNT'::character varying NOT NULL,
    paid_at date,
    note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(100)
);

--
-- Name: parcel_status; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_status (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: parcel_status_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_status_history (
    id integer NOT NULL,
    parcel_id character varying(12) NOT NULL,
    acquisition_id uuid NOT NULL,
    status_id integer NOT NULL,
    status_date timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255) DEFAULT 'system'::character varying NOT NULL
);

--
-- Name: parcel_status_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.parcel_status_history_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: parcel_status_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.parcel_status_history_id_seq OWNED BY public.parcel_status_history.id;

--
-- Name: parcel_valuation_submission; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_valuation_submission (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    status text DEFAULT 'draft'::text NOT NULL,
    submitted_by character varying(255) DEFAULT ''::character varying NOT NULL,
    submitted_at timestamp with time zone,
    reviewed_by character varying(255) DEFAULT ''::character varying NOT NULL,
    reviewed_at timestamp with time zone,
    last_note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL
);

--
-- Name: parcel_valuation_submission_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_valuation_submission_history (
    id integer NOT NULL,
    acquisition_id uuid NOT NULL,
    parcel_id character varying(12) NOT NULL,
    action text NOT NULL,
    from_status text DEFAULT ''::text NOT NULL,
    to_status text NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_by character varying(255) DEFAULT 'system'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    valuation_type text DEFAULT 'asset'::text NOT NULL
);

--
-- Name: parcel_valuation_submission_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.parcel_valuation_submission_history_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: parcel_valuation_submission_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.parcel_valuation_submission_history_id_seq OWNED BY public.parcel_valuation_submission_history.id;

--
-- Name: parcel_workflow; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.parcel_workflow (
    id integer NOT NULL,
    from_status_id integer,
    to_status_id integer NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);

--
-- Name: parcel_workflow_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.parcel_workflow_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: parcel_workflow_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.parcel_workflow_id_seq OWNED BY public.parcel_workflow.id;

--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_source character varying(16) DEFAULT 'auth'::character varying NOT NULL,
    user_id integer NOT NULL,
    token character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: plan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.plan (
    id uuid NOT NULL,
    plan_code character varying(100) NOT NULL,
    project_id character varying(100),
    name text DEFAULT ''::text NOT NULL,
    plan_type_name text,
    boundary_wkt public.geometry(Geometry,4326),
    area_m2 numeric(18,4),
    status smallint DEFAULT 0 NOT NULL,
    start_data text,
    end_data text,
    approved_date text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    parcel_id character varying(100),
    gazner text
);

--
-- Name: progress_type_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.progress_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: progress_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.progress_type_id_seq OWNED BY public.acquisition_progress_status.id;

--
-- Name: revoked_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.revoked_tokens (
    jti character varying(255) NOT NULL,
    user_source character varying(16) DEFAULT 'auth'::character varying NOT NULL,
    user_id integer NOT NULL,
    token_type character varying(32) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: right_holder; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.right_holder (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    last_name character varying(100),
    name character varying(100) NOT NULL,
    register_no character varying(20),
    address text,
    phone character varying(20),
    email character varying(255),
    holder_type text,
    civil_id character varying(50),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by character varying(255)
);

--
-- Name: seed_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.seed_history (
    name text NOT NULL,
    seeded_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: user_token_epoch; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.user_token_epoch (
    user_source character varying(16) DEFAULT 'auth'::character varying NOT NULL,
    user_id integer NOT NULL,
    valid_from timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: valuation_org_profile; Type: TABLE; Schema: public; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS public.valuation_org_profile_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.valuation_org_profile (
    id integer DEFAULT nextval('public.valuation_org_profile_id_seq'::regclass) NOT NULL,
    name character varying(250) DEFAULT ''::character varying NOT NULL,
    short_name character varying(250) DEFAULT ''::character varying NOT NULL,
    register_no character varying(20) DEFAULT ''::character varying NOT NULL,
    license_no character varying(50) DEFAULT ''::character varying NOT NULL,
    license_issued_at date,
    license_expires_at date,
    phone character varying(50) DEFAULT ''::character varying NOT NULL,
    email character varying(250) DEFAULT ''::character varying NOT NULL,
    address character varying(500) DEFAULT ''::character varying NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER SEQUENCE public.valuation_org_profile_id_seq
    OWNED BY public.valuation_org_profile.id;

--
-- Name: acquisition_category id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_category ALTER COLUMN id SET DEFAULT nextval('public.acquisition_category_id_seq'::regclass);

--
-- Name: acquisition_progress_status id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_progress_status ALTER COLUMN id SET DEFAULT nextval('public.progress_type_id_seq'::regclass);

--
-- Name: acquisition_workflow id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_workflow ALTER COLUMN id SET DEFAULT nextval('public.acquisition_workflow_id_seq'::regclass);

--
-- Name: asset_calc_type id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calc_type ALTER COLUMN id SET DEFAULT nextval('public.asset_calc_type_id_seq'::regclass);

--
-- Name: asset_spec_type id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec_type ALTER COLUMN id SET DEFAULT nextval('public.asset_spec_type_id_seq'::regclass);

--
-- Name: construction_type id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.construction_type ALTER COLUMN id SET DEFAULT nextval('public.construction_type_id_seq'::regclass);

--
-- Name: decision_budget id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_budget ALTER COLUMN id SET DEFAULT nextval('public.decision_budget_id_seq'::regclass);

--
-- Name: decision_work_type id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_work_type ALTER COLUMN id SET DEFAULT nextval('public.decision_work_type_id_seq'::regclass);

--
-- Name: document_type id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_type ALTER COLUMN id SET DEFAULT nextval('public.document_type_id_seq'::regclass);

--
-- Name: parcel_status_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status_history ALTER COLUMN id SET DEFAULT nextval('public.parcel_status_history_id_seq'::regclass);

--
-- Name: parcel_valuation_submission_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission_history ALTER COLUMN id SET DEFAULT nextval('public.parcel_valuation_submission_history_id_seq'::regclass);

--
-- Name: parcel_workflow id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_workflow ALTER COLUMN id SET DEFAULT nextval('public.parcel_workflow_id_seq'::regclass);

--
-- Name: acquisition_category acquisition_category_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_category
    ADD CONSTRAINT acquisition_category_pkey PRIMARY KEY (id);

--
-- Name: acquisition_document acquisition_document_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_document
    ADD CONSTRAINT acquisition_document_pkey PRIMARY KEY (acquisition_id, document_id);

--
-- Name: acquisition_progress acquisition_progress_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_progress
    ADD CONSTRAINT acquisition_progress_pkey PRIMARY KEY (id);

--
-- Name: acquisition_workflow acquisition_workflow_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_workflow
    ADD CONSTRAINT acquisition_workflow_pkey PRIMARY KEY (id);

--
-- Name: asset_calc_type asset_calc_type_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calc_type
    ADD CONSTRAINT asset_calc_type_code_key UNIQUE (code);

--
-- Name: asset_calc_type asset_calc_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calc_type
    ADD CONSTRAINT asset_calc_type_pkey PRIMARY KEY (id);

--
-- Name: asset_calculation asset_calculation_asset_id_calc_type_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calculation
    ADD CONSTRAINT asset_calculation_asset_id_calc_type_id_key UNIQUE (asset_id, calc_type_id);

--
-- Name: asset_calculation asset_calculation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calculation
    ADD CONSTRAINT asset_calculation_pkey PRIMARY KEY (id);

--
-- Name: asset_spec asset_spec_asset_id_spec_type_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec
    ADD CONSTRAINT asset_spec_asset_id_spec_type_id_key UNIQUE (asset_id, spec_type_id);

--
-- Name: asset_spec asset_spec_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec
    ADD CONSTRAINT asset_spec_pkey PRIMARY KEY (id);

--
-- Name: asset_spec_type asset_spec_type_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec_type
    ADD CONSTRAINT asset_spec_type_code_key UNIQUE (code);

--
-- Name: asset_spec_type asset_spec_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec_type
    ADD CONSTRAINT asset_spec_type_pkey PRIMARY KEY (id);

--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);

--
-- Name: au1 au_level1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.au1
    ADD CONSTRAINT au_level1_pkey PRIMARY KEY (code);

--
-- Name: au2 au_level2_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.au2
    ADD CONSTRAINT au_level2_pkey PRIMARY KEY (code);

--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);

--
-- Name: authorized_representatives authorized_representatives_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authorized_representatives
    ADD CONSTRAINT authorized_representatives_pkey PRIMARY KEY (id);

--
-- Name: client_request client_request_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_request
    ADD CONSTRAINT client_request_pkey PRIMARY KEY (id);

--
-- Name: compensation_grant compensation_grant_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation_grant
    ADD CONSTRAINT compensation_grant_pkey PRIMARY KEY (id);

--
-- Name: compensation_history compensation_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation_history
    ADD CONSTRAINT compensation_history_pkey PRIMARY KEY (id);

--
-- Name: compensation compensation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation
    ADD CONSTRAINT compensation_pkey PRIMARY KEY (id);

--
-- Name: construction_type construction_type_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.construction_type
    ADD CONSTRAINT construction_type_code_key UNIQUE (code);

--
-- Name: construction_type construction_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.construction_type
    ADD CONSTRAINT construction_type_pkey PRIMARY KEY (id);

--
-- Name: decision_budget decision_budget_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_budget
    ADD CONSTRAINT decision_budget_code_key UNIQUE (code);

--
-- Name: decision_budget decision_budget_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_budget
    ADD CONSTRAINT decision_budget_pkey PRIMARY KEY (id);

--
-- Name: decision_draft_funding_source decision_draft_funding_source_decision_draft_id_funding_sou_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_decision_draft_id_funding_sou_key UNIQUE (decision_draft_id, funding_source_id);

--
-- Name: decision_draft_funding_source decision_draft_funding_source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_pkey PRIMARY KEY (id);

--
-- Name: decision_draft_parcel decision_draft_parcel_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_parcel
    ADD CONSTRAINT decision_draft_parcel_pkey PRIMARY KEY (id);

--
-- Name: decision_draft decision_draft_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft
    ADD CONSTRAINT decision_draft_pkey PRIMARY KEY (id);

--
-- Name: decision_draft_progress_history decision_draft_progress_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_progress_history
    ADD CONSTRAINT decision_draft_progress_history_pkey PRIMARY KEY (id);

--
-- Name: decision_work_type decision_work_type_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_work_type
    ADD CONSTRAINT decision_work_type_code_key UNIQUE (code);

--
-- Name: decision_work_type decision_work_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_work_type
    ADD CONSTRAINT decision_work_type_pkey PRIMARY KEY (id);

--
-- Name: document document_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document
    ADD CONSTRAINT document_pkey PRIMARY KEY (id);

--
-- Name: document_type document_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_type
    ADD CONSTRAINT document_type_pkey PRIMARY KEY (id);

--
-- Name: document_type document_type_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_type
    ADD CONSTRAINT document_type_type_key UNIQUE (type);

--
-- Name: drone_image drone_image_file_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drone_image
    ADD CONSTRAINT drone_image_file_name_key UNIQUE (file_name);

--
-- Name: drone_image drone_image_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drone_image
    ADD CONSTRAINT drone_image_pkey PRIMARY KEY (id);

--
-- Name: funding_sources funding_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.funding_sources
    ADD CONSTRAINT funding_sources_pkey PRIMARY KEY (id);

--
-- Name: land_acquisition_assignee land_acquisition_assignee_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_assignee
    ADD CONSTRAINT land_acquisition_assignee_pkey PRIMARY KEY (acquisition_id, user_id);

--
-- Name: land_acquisition_au land_acquisition_au_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_au
    ADD CONSTRAINT land_acquisition_au_pkey PRIMARY KEY (land_acquisition_id, au3_code);

--
-- Name: land_acquisition_boundary_history land_acquisition_boundary_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_boundary_history
    ADD CONSTRAINT land_acquisition_boundary_history_pkey PRIMARY KEY (id);

--
-- Name: land_acquisition_document land_acquisition_document_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_document
    ADD CONSTRAINT land_acquisition_document_pkey PRIMARY KEY (land_acquisition_id, document_id);

--
-- Name: land_acquisition land_acquisition_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition
    ADD CONSTRAINT land_acquisition_pkey PRIMARY KEY (id);

--
-- Name: land_valuation land_valuation_acq_parcel_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_valuation
    ADD CONSTRAINT land_valuation_acq_parcel_type_key UNIQUE (acquisition_id, parcel_id, valuation_type);

--
-- Name: land_valuation land_valuation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_valuation
    ADD CONSTRAINT land_valuation_pkey PRIMARY KEY (id);

--
-- Name: notification notification_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification
    ADD CONSTRAINT notification_pkey PRIMARY KEY (id);

--
-- Name: parcel_court_decision parcel_court_decision_parcel_id_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_court_decision
    ADD CONSTRAINT parcel_court_decision_parcel_id_source_id_key UNIQUE (parcel_id, source_id);

--
-- Name: parcel_court_decision parcel_court_decision_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_court_decision
    ADD CONSTRAINT parcel_court_decision_pkey PRIMARY KEY (id);

--
-- Name: parcel_detail parcel_detail_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_detail
    ADD CONSTRAINT parcel_detail_pkey PRIMARY KEY (id);

--
-- Name: parcel_document parcel_document_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_document
    ADD CONSTRAINT parcel_document_pkey PRIMARY KEY (parcel_id, document_id);

--
-- Name: parcel_holder parcel_holder_parcel_id_holder_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_holder
    ADD CONSTRAINT parcel_holder_parcel_id_holder_id_key UNIQUE (parcel_id, holder_id);

--
-- Name: parcel_holder parcel_holder_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_holder
    ADD CONSTRAINT parcel_holder_pkey PRIMARY KEY (id);

--
-- Name: parcel_invoice parcel_invoice_parcel_id_invoice_no_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_invoice
    ADD CONSTRAINT parcel_invoice_parcel_id_invoice_no_key UNIQUE (parcel_id, invoice_no);

--
-- Name: parcel_invoice parcel_invoice_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_invoice
    ADD CONSTRAINT parcel_invoice_pkey PRIMARY KEY (id);

--
-- Name: parcel_monitoring parcel_monitoring_monitoring_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_monitoring
    ADD CONSTRAINT parcel_monitoring_monitoring_id_key UNIQUE (monitoring_id);

--
-- Name: parcel_monitoring parcel_monitoring_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_monitoring
    ADD CONSTRAINT parcel_monitoring_pkey PRIMARY KEY (id);

--
-- Name: parcel_mortgage parcel_mortgage_parcel_id_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_mortgage
    ADD CONSTRAINT parcel_mortgage_parcel_id_source_id_key UNIQUE (parcel_id, source_id);

--
-- Name: parcel_mortgage parcel_mortgage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_mortgage
    ADD CONSTRAINT parcel_mortgage_pkey PRIMARY KEY (id);

--
-- Name: parcel parcel_parcel_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel
    ADD CONSTRAINT parcel_parcel_id_key UNIQUE (parcel_id);

--
-- Name: parcel_payment parcel_payment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_payment
    ADD CONSTRAINT parcel_payment_pkey PRIMARY KEY (id);

--
-- Name: parcel parcel_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel
    ADD CONSTRAINT parcel_pkey PRIMARY KEY (id);

--
-- Name: parcel_status parcel_status_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status
    ADD CONSTRAINT parcel_status_code_key UNIQUE (code);

--
-- Name: parcel_status_history parcel_status_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status_history
    ADD CONSTRAINT parcel_status_history_pkey PRIMARY KEY (id);

--
-- Name: parcel_status parcel_status_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status
    ADD CONSTRAINT parcel_status_pkey PRIMARY KEY (id);

--
-- Name: parcel_valuation_submission_history parcel_valuation_submission_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission_history
    ADD CONSTRAINT parcel_valuation_submission_history_pkey PRIMARY KEY (id);

--
-- Name: parcel_valuation_submission parcel_valuation_submission_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission
    ADD CONSTRAINT parcel_valuation_submission_pkey PRIMARY KEY (id);

--
-- Name: parcel_workflow parcel_workflow_from_status_id_to_status_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_workflow
    ADD CONSTRAINT parcel_workflow_from_status_id_to_status_id_key UNIQUE (from_status_id, to_status_id);

--
-- Name: parcel_workflow parcel_workflow_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_workflow
    ADD CONSTRAINT parcel_workflow_pkey PRIMARY KEY (id);

--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);

--
-- Name: password_reset_tokens password_reset_tokens_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_token_key UNIQUE (token);

--
-- Name: plan plan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);

--
-- Name: plan plan_plan_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_plan_code_key UNIQUE (plan_code);

--
-- Name: acquisition_progress_status progress_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_progress_status
    ADD CONSTRAINT progress_type_pkey PRIMARY KEY (id);

--
-- Name: parcel_valuation_submission pvs_acq_parcel_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission
    ADD CONSTRAINT pvs_acq_parcel_type_key UNIQUE (acquisition_id, parcel_id, valuation_type);

--
-- Name: revoked_tokens revoked_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revoked_tokens
    ADD CONSTRAINT revoked_tokens_pkey PRIMARY KEY (jti);

--
-- Name: right_holder right_holder_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.right_holder
    ADD CONSTRAINT right_holder_pkey PRIMARY KEY (id);

--
-- Name: right_holder right_holder_register_no_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.right_holder
    ADD CONSTRAINT right_holder_register_no_key UNIQUE (register_no);

--
-- Name: seed_history seed_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seed_history
    ADD CONSTRAINT seed_history_pkey PRIMARY KEY (name);

--
-- Name: user_token_epoch user_token_epoch_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_token_epoch
    ADD CONSTRAINT user_token_epoch_pkey PRIMARY KEY (user_source, user_id);

--
-- Name: valuation_org_profile valuation_org_profile_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.valuation_org_profile
    ADD CONSTRAINT valuation_org_profile_pkey PRIMARY KEY (id);

--
-- Name: au_level1_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS au_level1_code_idx ON public.au1 USING btree (code);

--
-- Name: au_level2_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS au_level2_code_idx ON public.au2 USING btree (code);

--
-- Name: au_level3_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS au_level3_code_idx ON public.au3 USING btree (code);

--
-- Name: idx_acq_assignee_acq_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_acq_assignee_acq_id ON public.land_acquisition_assignee USING btree (acquisition_id);

--
-- Name: idx_acq_assignee_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_acq_assignee_user_id ON public.land_acquisition_assignee USING btree (user_id);

--
-- Name: idx_acq_doc_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_acq_doc_acquisition ON public.acquisition_document USING btree (acquisition_id);

--
-- Name: idx_acq_progress_acq_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_acq_progress_acq_id ON public.acquisition_progress USING btree (acquisition_id);

--
-- Name: idx_acquisition_workflow_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_acquisition_workflow_from ON public.acquisition_workflow USING btree (from_status);

--
-- Name: idx_assets_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_assets_acquisition_id ON public.assets USING btree (acquisition_id);

--
-- Name: idx_assets_geometry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_assets_geometry ON public.assets USING gist (geometry);

--
-- Name: idx_assets_parcel_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_assets_parcel_id ON public.assets USING btree (parcel_id);

--
-- Name: idx_assets_valuation_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_assets_valuation_type ON public.assets USING btree (valuation_type);

--
-- Name: idx_audit_log_acquisition_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_acquisition_created ON public.audit_log USING btree (acquisition_id, created_at DESC) WHERE (acquisition_id IS NOT NULL);

--
-- Name: idx_audit_log_action_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_action_created ON public.audit_log USING btree (action, created_at DESC);

--
-- Name: idx_audit_log_actor_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_actor_created ON public.audit_log USING btree (actor_id, created_at DESC);

--
-- Name: idx_audit_log_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_created ON public.audit_log USING btree (created_at DESC);

--
-- Name: idx_audit_log_parcel_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_parcel_created ON public.audit_log USING btree (parcel_id, created_at DESC) WHERE (parcel_id IS NOT NULL);

--
-- Name: idx_audit_log_resource_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_resource_created ON public.audit_log USING btree (resource_type, resource_id, created_at DESC);

--
-- Name: idx_audit_log_resource_type_action_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_resource_type_action_created ON public.audit_log USING btree (resource_type, action, created_at DESC);

--
-- Name: idx_audit_log_resource_type_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_resource_type_created ON public.audit_log USING btree (resource_type, created_at DESC);

--
-- Name: idx_authorized_representatives_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_authorized_representatives_acquisition_id ON public.authorized_representatives USING btree (acquisition_id);

--
-- Name: idx_authorized_representatives_parcel_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_authorized_representatives_parcel_id ON public.authorized_representatives USING btree (parcel_id);

--
-- Name: idx_client_request_host; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_client_request_host ON public.client_request USING btree (host);

--
-- Name: idx_client_request_requested_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_client_request_requested_at ON public.client_request USING btree (requested_at DESC);

--
-- Name: idx_compensation_acq_parcel_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_compensation_acq_parcel_type ON public.compensation USING btree (acquisition_id, parcel_id, compensation_type);

--
-- Name: idx_compensation_asset_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_compensation_asset_id ON public.compensation USING btree (asset_id) WHERE (asset_id IS NOT NULL);

--
-- Name: idx_compensation_grant_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_compensation_grant_acquisition ON public.compensation_grant USING btree (acquisition_id);

--
-- Name: idx_compensation_history_comp_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_compensation_history_comp_id ON public.compensation_history USING btree (compensation_id);

--
-- Name: idx_compensation_valuation_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_compensation_valuation_type ON public.compensation USING btree (valuation_type);

--
-- Name: idx_ddfs_draft; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddfs_draft ON public.decision_draft_funding_source USING btree (decision_draft_id);

--
-- Name: idx_ddfs_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddfs_source ON public.decision_draft_funding_source USING btree (funding_source_id);

--
-- Name: idx_ddp_draft; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddp_draft ON public.decision_draft_parcel USING btree (decision_draft_id);

--
-- Name: idx_ddp_funding_link; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddp_funding_link ON public.decision_draft_parcel USING btree (funding_link_id);

--
-- Name: idx_ddp_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddp_parcel ON public.decision_draft_parcel USING btree (parcel_id);

--
-- Name: idx_ddph_draft_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ddph_draft_date ON public.decision_draft_progress_history USING btree (decision_draft_id, progress_date DESC, created_at DESC);

--
-- Name: idx_decision_draft_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_decision_draft_acquisition ON public.decision_draft USING btree (acquisition_id);

--
-- Name: idx_decision_draft_current_progress; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_decision_draft_current_progress ON public.decision_draft USING btree (current_progress_history_id);

--
-- Name: idx_decision_draft_decree_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_decision_draft_decree_number ON public.decision_draft USING btree (decree_number);

--
-- Name: idx_decision_draft_proposal_no; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_decision_draft_proposal_no ON public.decision_draft USING btree (proposal_no);

--
-- Name: idx_decision_draft_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_decision_draft_status ON public.decision_draft USING btree (status);

--
-- Name: idx_document_source_doc_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_document_source_doc_code ON public.document USING btree (source_doc_code) WHERE (source_doc_code IS NOT NULL);

--
-- Name: idx_drone_image_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_drone_image_acquisition ON public.drone_image USING btree (acquisition_id, uploaded_at DESC);

--
-- Name: idx_funding_sources_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_funding_sources_acquisition_id ON public.funding_sources USING btree (acquisition_id);

--
-- Name: idx_funding_sources_standalone; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_funding_sources_standalone ON public.funding_sources USING btree (created_at) WHERE (acquisition_id IS NULL);

--
-- Name: idx_la_boundary_history_acq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_acq ON public.land_acquisition_boundary_history USING btree (land_acquisition_id, changed_at DESC);

--
-- Name: idx_la_boundary_history_new_geom; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_new_geom ON public.land_acquisition_boundary_history USING gist (new_geometry);

--
-- Name: idx_la_boundary_history_old_geom; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_la_boundary_history_old_geom ON public.land_acquisition_boundary_history USING gist (old_geometry);

--
-- Name: idx_la_doc_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_la_doc_acquisition ON public.land_acquisition_document USING btree (land_acquisition_id);

--
-- Name: idx_la_professional_org; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_la_professional_org ON public.land_acquisition USING btree (professional_org_id);

--
-- Name: idx_land_acquisition_acq_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_acq_name_trgm ON public.land_acquisition USING gin (acquisition_name public.gin_trgm_ops);

--
-- Name: idx_land_acquisition_construction_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_construction_type ON public.land_acquisition USING btree (construction_type_id);

--
-- Name: idx_land_acquisition_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_created_at ON public.land_acquisition USING btree (created_at DESC);

--
-- Name: idx_land_acquisition_decree_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_decree_date ON public.land_acquisition USING btree (decree_date);

--
-- Name: idx_land_acquisition_decree_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_decree_year ON public.land_acquisition USING btree (EXTRACT(year FROM COALESCE(decree_date, start_date)));

--
-- Name: idx_land_acquisition_geometry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_geometry ON public.land_acquisition USING gist (geometry);

--
-- Name: idx_land_acquisition_plan_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_code ON public.land_acquisition USING btree (plan_code);

--
-- Name: idx_land_acquisition_plan_code_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_code_trgm ON public.land_acquisition USING gin (plan_code public.gin_trgm_ops);

--
-- Name: idx_land_acquisition_plan_geom; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_plan_geom ON public.land_acquisition USING gist (plan_geom);

--
-- Name: idx_land_acquisition_start_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_start_date ON public.land_acquisition USING btree (start_date);

--
-- Name: idx_land_acquisition_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_acquisition_status ON public.land_acquisition USING btree (status);

--
-- Name: idx_land_valuation_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_valuation_acquisition ON public.land_valuation USING btree (acquisition_id);

--
-- Name: idx_land_valuation_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_land_valuation_parcel ON public.land_valuation USING btree (parcel_id);

--
-- Name: idx_notification_recipient_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_notification_recipient_created ON public.notification USING btree (recipient_id, created_at DESC);

--
-- Name: idx_notification_recipient_unread; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_notification_recipient_unread ON public.notification USING btree (recipient_id) WHERE (is_read = false);

--
-- Name: idx_parcel_acquisition_geom; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_acquisition_geom ON public.parcel USING gist (acquisition_geom);

--
-- Name: idx_parcel_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_acquisition_id ON public.parcel USING btree (acquisition_id);

--
-- Name: idx_parcel_au_codes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_au_codes ON public.parcel USING btree (au1_code, au2_code, au3_code);

--
-- Name: idx_parcel_court_decision_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_court_decision_parcel ON public.parcel_court_decision USING btree (parcel_id);

--
-- Name: idx_parcel_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_created_at ON public.parcel USING btree (created_at DESC);

--
-- Name: idx_parcel_detail_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_detail_acquisition_id ON public.parcel_detail USING btree (acquisition_id);

--
-- Name: idx_parcel_detail_holder_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_detail_holder_id ON public.parcel_detail USING btree (holder_id);

--
-- Name: idx_parcel_detail_parcel_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_detail_parcel_id ON public.parcel_detail USING btree (parcel_id);

--
-- Name: idx_parcel_doc_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_doc_parcel ON public.parcel_document USING btree (parcel_id);

--
-- Name: idx_parcel_geometry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_geometry ON public.parcel USING gist (geometry);

--
-- Name: idx_parcel_holder_acquisition_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_holder_acquisition_id ON public.parcel_holder USING btree (acquisition_id);

--
-- Name: idx_parcel_holder_holder_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_holder_holder_id ON public.parcel_holder USING btree (holder_id);

--
-- Name: idx_parcel_holder_parcel_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_holder_parcel_id ON public.parcel_holder USING btree (parcel_id);

--
-- Name: idx_parcel_holder_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_holder_role ON public.parcel_holder USING btree (parcel_id, holder_role);

--
-- Name: idx_parcel_independent_org; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_independent_org ON public.parcel USING btree (independent_org_id);

--
-- Name: idx_parcel_invoice_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_invoice_parcel ON public.parcel_invoice USING btree (parcel_id);

--
-- Name: idx_parcel_landuse; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_landuse ON public.parcel USING btree (landuse);

--
-- Name: idx_parcel_landuse_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_landuse_trgm ON public.parcel USING gin (landuse public.gin_trgm_ops);

--
-- Name: idx_parcel_monitoring_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_monitoring_acquisition ON public.parcel_monitoring USING btree (acquisition_id);

--
-- Name: idx_parcel_monitoring_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_monitoring_parcel ON public.parcel_monitoring USING btree (parcel_id);

--
-- Name: idx_parcel_mortgage_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_mortgage_parcel ON public.parcel_mortgage USING btree (parcel_id);

--
-- Name: idx_parcel_mortgage_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_mortgage_parent ON public.parcel_mortgage USING btree (parcel_id, source_parent_id);

--
-- Name: idx_parcel_parcel_id_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_parcel_id_trgm ON public.parcel USING gin (parcel_id public.gin_trgm_ops);

--
-- Name: idx_parcel_payment_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_payment_parcel ON public.parcel_payment USING btree (parcel_id);

--
-- Name: idx_parcel_right_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_right_type ON public.parcel USING btree (right_type);

--
-- Name: idx_parcel_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_parcel_status ON public.parcel USING btree (status);

--
-- Name: idx_plan_boundary_wkt; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_boundary_wkt ON public.plan USING gist (boundary_wkt);

--
-- Name: idx_plan_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_name ON public.plan USING btree (name);

--
-- Name: idx_plan_parcel_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_parcel_id ON public.plan USING btree (parcel_id);

--
-- Name: idx_prt_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_prt_expires_at ON public.password_reset_tokens USING btree (expires_at);

--
-- Name: idx_prt_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_prt_token ON public.password_reset_tokens USING btree (token);

--
-- Name: idx_psh_acquisition; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_psh_acquisition ON public.parcel_status_history USING btree (acquisition_id);

--
-- Name: idx_psh_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_psh_parcel ON public.parcel_status_history USING btree (parcel_id);

--
-- Name: idx_pvs_acq_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_pvs_acq_parcel ON public.parcel_valuation_submission USING btree (acquisition_id, parcel_id);

--
-- Name: idx_pvsh_acq_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_pvsh_acq_parcel ON public.parcel_valuation_submission_history USING btree (acquisition_id, parcel_id, created_at DESC);

--
-- Name: idx_revoked_tokens_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON public.revoked_tokens USING btree (expires_at);

--
-- Name: idx_revoked_tokens_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user ON public.revoked_tokens USING btree (user_source, user_id);

--
-- Name: idx_valuation_org_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_valuation_org_active ON public.valuation_org_profile USING btree (is_active);

--
-- Name: st_idx_au_level1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS st_idx_au_level1 ON public.au1 USING gist (geometry);

--
-- Name: st_idx_au_level2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS st_idx_au_level2 ON public.au2 USING gist (geometry);

--
-- Name: st_idx_au_level3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS st_idx_au_level3 ON public.au3 USING gist (geometry);

--
-- Name: uq_ddp_active_parcel; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS uq_ddp_active_parcel ON public.decision_draft_parcel USING btree (parcel_id) WHERE (removed_at IS NULL);

--
-- Name: uq_parcel_holder_payment_recipient; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS uq_parcel_holder_payment_recipient ON public.parcel_holder USING btree (parcel_id) WHERE payment_recipient;

--
-- Name: uq_valuation_org_register; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS uq_valuation_org_register ON public.valuation_org_profile USING btree (register_no) WHERE ((register_no)::text <> ''::text);

--
-- Name: ux_document_source_doc_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS ux_document_source_doc_id ON public.document USING btree (source_doc_id) WHERE (source_doc_id IS NOT NULL);

--
-- Name: acquisition_category acquisition_category_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_category
    ADD CONSTRAINT acquisition_category_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.acquisition_category(id);

--
-- Name: acquisition_document acquisition_document_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_document
    ADD CONSTRAINT acquisition_document_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: acquisition_document acquisition_document_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_document
    ADD CONSTRAINT acquisition_document_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.document(id) ON DELETE CASCADE;

--
-- Name: acquisition_progress acquisition_progress_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acquisition_progress
    ADD CONSTRAINT acquisition_progress_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: asset_calculation asset_calculation_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calculation
    ADD CONSTRAINT asset_calculation_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE CASCADE;

--
-- Name: asset_calculation asset_calculation_calc_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_calculation
    ADD CONSTRAINT asset_calculation_calc_type_id_fkey FOREIGN KEY (calc_type_id) REFERENCES public.asset_calc_type(id);

--
-- Name: asset_spec asset_spec_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec
    ADD CONSTRAINT asset_spec_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE CASCADE;

--
-- Name: asset_spec asset_spec_spec_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_spec
    ADD CONSTRAINT asset_spec_spec_type_id_fkey FOREIGN KEY (spec_type_id) REFERENCES public.asset_spec_type(id);

--
-- Name: assets assets_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: assets assets_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE;

--
-- Name: authorized_representatives authorized_representatives_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authorized_representatives
    ADD CONSTRAINT authorized_representatives_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: compensation compensation_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation
    ADD CONSTRAINT compensation_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: compensation compensation_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation
    ADD CONSTRAINT compensation_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE SET NULL;

--
-- Name: compensation_grant compensation_grant_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation_grant
    ADD CONSTRAINT compensation_grant_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: compensation_grant compensation_grant_compensation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compensation_grant
    ADD CONSTRAINT compensation_grant_compensation_id_fkey FOREIGN KEY (compensation_id) REFERENCES public.compensation(id) ON DELETE CASCADE;

--
-- Name: decision_draft decision_draft_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft
    ADD CONSTRAINT decision_draft_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE SET NULL;

--
-- Name: decision_draft decision_draft_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft
    ADD CONSTRAINT decision_draft_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.decision_budget(id) ON DELETE SET NULL;

--
-- Name: decision_draft decision_draft_current_progress_history_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft
    ADD CONSTRAINT decision_draft_current_progress_history_id_fkey FOREIGN KEY (current_progress_history_id) REFERENCES public.decision_draft_progress_history(id) ON DELETE SET NULL;

--
-- Name: decision_draft_funding_source decision_draft_funding_source_decision_draft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_decision_draft_id_fkey FOREIGN KEY (decision_draft_id) REFERENCES public.decision_draft(id) ON DELETE CASCADE;

--
-- Name: decision_draft_funding_source decision_draft_funding_source_funding_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_funding_source
    ADD CONSTRAINT decision_draft_funding_source_funding_source_id_fkey FOREIGN KEY (funding_source_id) REFERENCES public.funding_sources(id) ON DELETE CASCADE;

--
-- Name: decision_draft_parcel decision_draft_parcel_decision_draft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_parcel
    ADD CONSTRAINT decision_draft_parcel_decision_draft_id_fkey FOREIGN KEY (decision_draft_id) REFERENCES public.decision_draft(id) ON DELETE CASCADE;

--
-- Name: decision_draft_parcel decision_draft_parcel_funding_link_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_parcel
    ADD CONSTRAINT decision_draft_parcel_funding_link_id_fkey FOREIGN KEY (funding_link_id) REFERENCES public.decision_draft_funding_source(id) ON DELETE RESTRICT;

--
-- Name: decision_draft_parcel decision_draft_parcel_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_parcel
    ADD CONSTRAINT decision_draft_parcel_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: decision_draft_progress_history decision_draft_progress_history_decision_draft_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft_progress_history
    ADD CONSTRAINT decision_draft_progress_history_decision_draft_id_fkey FOREIGN KEY (decision_draft_id) REFERENCES public.decision_draft(id) ON DELETE CASCADE;

--
-- Name: decision_draft decision_draft_work_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.decision_draft
    ADD CONSTRAINT decision_draft_work_type_id_fkey FOREIGN KEY (work_type_id) REFERENCES public.decision_work_type(id) ON DELETE SET NULL;

--
-- Name: document document_document_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document
    ADD CONSTRAINT document_document_type_id_fkey FOREIGN KEY (document_type_id) REFERENCES public.document_type(id);

--
-- Name: drone_image drone_image_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drone_image
    ADD CONSTRAINT drone_image_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: assets fk_assets_parcel_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT fk_assets_parcel_id FOREIGN KEY (parcel_id) REFERENCES public.parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE NOT VALID;

--
-- Name: funding_sources funding_sources_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.funding_sources
    ADD CONSTRAINT funding_sources_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_acquisition_assignee land_acquisition_assignee_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_assignee
    ADD CONSTRAINT land_acquisition_assignee_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_acquisition_au land_acquisition_au_land_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_au
    ADD CONSTRAINT land_acquisition_au_land_acquisition_id_fkey FOREIGN KEY (land_acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_acquisition_boundary_history land_acquisition_boundary_history_land_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_boundary_history
    ADD CONSTRAINT land_acquisition_boundary_history_land_acquisition_id_fkey FOREIGN KEY (land_acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_acquisition land_acquisition_construction_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition
    ADD CONSTRAINT land_acquisition_construction_type_id_fkey FOREIGN KEY (construction_type_id) REFERENCES public.construction_type(id) ON DELETE SET NULL;

--
-- Name: land_acquisition_document land_acquisition_document_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_document
    ADD CONSTRAINT land_acquisition_document_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.document(id) ON DELETE CASCADE;

--
-- Name: land_acquisition_document land_acquisition_document_land_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition_document
    ADD CONSTRAINT land_acquisition_document_land_acquisition_id_fkey FOREIGN KEY (land_acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_acquisition land_acquisition_general_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition
    ADD CONSTRAINT land_acquisition_general_category_id_fkey FOREIGN KEY (general_category_id) REFERENCES public.acquisition_category(id);

--
-- Name: land_acquisition land_acquisition_sub_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_acquisition
    ADD CONSTRAINT land_acquisition_sub_category_id_fkey FOREIGN KEY (sub_category_id) REFERENCES public.acquisition_category(id);

--
-- Name: land_valuation land_valuation_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_valuation
    ADD CONSTRAINT land_valuation_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: land_valuation land_valuation_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.land_valuation
    ADD CONSTRAINT land_valuation_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE;

--
-- Name: parcel parcel_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel
    ADD CONSTRAINT parcel_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_court_decision parcel_court_decision_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_court_decision
    ADD CONSTRAINT parcel_court_decision_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_court_decision parcel_court_decision_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_court_decision
    ADD CONSTRAINT parcel_court_decision_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_detail parcel_detail_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_detail
    ADD CONSTRAINT parcel_detail_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_detail parcel_detail_holder_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_detail
    ADD CONSTRAINT parcel_detail_holder_id_fkey FOREIGN KEY (holder_id) REFERENCES public.right_holder(id);

--
-- Name: parcel_detail parcel_detail_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_detail
    ADD CONSTRAINT parcel_detail_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_document parcel_document_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_document
    ADD CONSTRAINT parcel_document_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.document(id) ON DELETE CASCADE;

--
-- Name: parcel_document parcel_document_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_document
    ADD CONSTRAINT parcel_document_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_holder parcel_holder_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_holder
    ADD CONSTRAINT parcel_holder_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_holder parcel_holder_holder_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_holder
    ADD CONSTRAINT parcel_holder_holder_id_fkey FOREIGN KEY (holder_id) REFERENCES public.right_holder(id);

--
-- Name: parcel_holder parcel_holder_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_holder
    ADD CONSTRAINT parcel_holder_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_invoice parcel_invoice_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_invoice
    ADD CONSTRAINT parcel_invoice_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_invoice parcel_invoice_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_invoice
    ADD CONSTRAINT parcel_invoice_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_monitoring parcel_monitoring_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_monitoring
    ADD CONSTRAINT parcel_monitoring_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_monitoring parcel_monitoring_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_monitoring
    ADD CONSTRAINT parcel_monitoring_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_mortgage parcel_mortgage_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_mortgage
    ADD CONSTRAINT parcel_mortgage_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_mortgage parcel_mortgage_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_mortgage
    ADD CONSTRAINT parcel_mortgage_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel_payment parcel_payment_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_payment
    ADD CONSTRAINT parcel_payment_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(id) ON DELETE CASCADE;

--
-- Name: parcel parcel_status_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel
    ADD CONSTRAINT parcel_status_fkey FOREIGN KEY (status) REFERENCES public.parcel_status(id);

--
-- Name: parcel_status_history parcel_status_history_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status_history
    ADD CONSTRAINT parcel_status_history_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_status_history parcel_status_history_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status_history
    ADD CONSTRAINT parcel_status_history_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(parcel_id) ON DELETE CASCADE;

--
-- Name: parcel_status_history parcel_status_history_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_status_history
    ADD CONSTRAINT parcel_status_history_status_id_fkey FOREIGN KEY (status_id) REFERENCES public.parcel_status(id);

--
-- Name: parcel_valuation_submission parcel_valuation_submission_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission
    ADD CONSTRAINT parcel_valuation_submission_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_valuation_submission_history parcel_valuation_submission_history_acquisition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission_history
    ADD CONSTRAINT parcel_valuation_submission_history_acquisition_id_fkey FOREIGN KEY (acquisition_id) REFERENCES public.land_acquisition(id) ON DELETE CASCADE;

--
-- Name: parcel_valuation_submission parcel_valuation_submission_parcel_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_valuation_submission
    ADD CONSTRAINT parcel_valuation_submission_parcel_id_fkey FOREIGN KEY (parcel_id) REFERENCES public.parcel(parcel_id) ON UPDATE CASCADE ON DELETE CASCADE;

--
-- Name: parcel_workflow parcel_workflow_from_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_workflow
    ADD CONSTRAINT parcel_workflow_from_status_id_fkey FOREIGN KEY (from_status_id) REFERENCES public.parcel_status(id);

--
-- Name: parcel_workflow parcel_workflow_to_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.parcel_workflow
    ADD CONSTRAINT parcel_workflow_to_status_id_fkey FOREIGN KEY (to_status_id) REFERENCES public.parcel_status(id);
