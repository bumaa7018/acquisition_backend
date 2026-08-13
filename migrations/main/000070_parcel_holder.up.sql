-- Нэгж талбарын ОЛОН эзэмшигч.
--
-- parcel_detail.holder_id нь зөвхөн НЭГ эзэмшигчийг хэлдэг (үндсэн хүсэлт
-- гаргагч). Нэг хүсэлт (app_no) дээр эзэмшигчийн төрөл өөр хэд хэдэн хүн
-- байж болох тул бүгдийг энд хадгална.
CREATE TABLE IF NOT EXISTS parcel_holder (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id      UUID        NOT NULL REFERENCES parcel(id) ON DELETE CASCADE,
    acquisition_id UUID        REFERENCES land_acquisition(id) ON DELETE CASCADE,
    holder_id      UUID        NOT NULL REFERENCES right_holder(id),
    app_no         VARCHAR(100),
    main_applicant BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     VARCHAR(255),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by     VARCHAR(255),
    UNIQUE (parcel_id, holder_id)
);

COMMENT ON TABLE  parcel_holder                IS 'Нэгж талбарын эзэмшигчид (дундын сервисийн /application/persons-оос татагдана)';
COMMENT ON COLUMN parcel_holder.main_applicant IS 'Үндсэн хүсэлт гаргагч (ct_application_person_role.main_applicant)';
COMMENT ON COLUMN parcel_holder.app_no         IS 'Аль хүсэлтээс татагдсан (ct_application.app_no)';

CREATE INDEX IF NOT EXISTS idx_parcel_holder_parcel_id      ON parcel_holder (parcel_id);
CREATE INDEX IF NOT EXISTS idx_parcel_holder_acquisition_id ON parcel_holder (acquisition_id);
CREATE INDEX IF NOT EXISTS idx_parcel_holder_holder_id      ON parcel_holder (holder_id);
