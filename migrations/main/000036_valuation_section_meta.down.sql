ALTER TABLE public.valuation_section_note
    DROP COLUMN IF EXISTS sort_order,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS table_label,
    DROP COLUMN IF EXISTS section_no;
