-- Дундын сервисийн /plan/project хариултад нэмэгдсэн хоёр талбар.
--
-- ДУГААРЛАЛТ: 77-аас 90 болгож ЗОРИУД алгасав. Энэ migration нь зэрэгцээ
-- хөгжүүлэлттэй давхцаж 73→75→76→77 дөрвөн удаа мөргөлдсөн (parcel_holder_role_payment,
-- document_source_code, parcel_mortgage_court, parcel_invoice). 78-89 зайг
-- тэдгээр ажилд үлдээв. Дугаар дараалалгүй байх нь golang-migrate-д асуудалгүй.
--
-- Төлөвлөгөөг одоо ТӨЛӨВЛӨГӨӨНИЙ НЭГЖ ТАЛБАРЫН дугаараар
-- (data_plan.pl_project_parcel.parcel_id) хайдаг болсон. gazner нь тухайн
-- нэгж талбар дээр хийгдэх бүтээн байгуулалтын ажлын нэр.
ALTER TABLE plan
    ADD COLUMN IF NOT EXISTS parcel_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS gazner    TEXT;

COMMENT ON COLUMN plan.parcel_id IS 'Төлөвлөгөөний нэгж талбарын дугаар (data_plan.pl_project_parcel.parcel_id) — төлөвлөгөөг үүгээр хайна';
COMMENT ON COLUMN plan.gazner    IS 'Бүтээн байгуулалтын ажлын нэр, ж: "Дамбадаржаа дэд төвийн бүтээн байгуулалтын ажил"';

-- Хайлт (Suggest) нь plan_code/name-ээс гадна эдгээрээр ч олдоно.
CREATE INDEX IF NOT EXISTS idx_plan_parcel_id ON plan (parcel_id);

-- /plan/project нь ХИЛИЙГ (geometry_wkt) буцаадаг болсон тул 000068 дээрх
-- "хил буцаадаггүй" тайлбар хүчингүй болов. Багана хэвээр NULL байж болно:
-- бүх төлөвлөгөөний нэгж талбарт polygon_geom бүртгэгдээгүй байж болно.
COMMENT ON COLUMN plan.boundary_wkt IS 'PostGIS geometry(Geometry, 4326) — /plan/project-ийн geometry_wkt (pl_project_parcel.polygon_geom). Бүртгэгдээгүй үед NULL';
