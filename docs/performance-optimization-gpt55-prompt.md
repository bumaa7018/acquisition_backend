# GPT-5.5-д өгөх хэрэгжүүлэлтийн prompt

Доорх prompt-ийг энэ repository-д хандах боломжтой GPT-5.5 coding agent-д өгнө. Энэ нь одоо хэрэгжүүлэлт эхлүүлэх заавар биш, дараагийн ажлын бэлэн даалгавар юм.

---

Та энэ Go/Gin/PostgreSQL/PostGIS backend-ийн гүйцэтгэлийг оновчлох инженерийн үүрэгтэй. `docs/performance-optimization-plan.md`-ийг бүрэн уншаад төлөвлөгөөг үе шаттай хэрэгжүүл. Зөвхөн санал бичээд зогсохгүй: код, шаардлагатай migration, нийцлийн тест, хэмжилт болон тайланг бэлд.

## Хамгийн чухал шаардлага — ҮР ДҮН ӨӨРЧЛӨГДӨХГҮЙ

**Хурд сайжруулахын тулд API эсвэл дэлгэцийн үр дүнг өөрчлөхийг хориглоно. Нийцэл нь бүх гүйцэтгэлийн зорилтоос давуу.**

Хуучин ба шинэ хэрэгжүүлэлт нь ижил оролт, хэрэглэгчийн эрх, өгөгдлийн төлөв дээр ижил бизнесийн үр дүн гаргах ёстой. Дараахыг хадгал:

- Endpoint, HTTP method, query/body параметр, default, validation болон error response.
- HTTP status, JSON field/төрөл, omitted/null/хоосон массивын ялгаа.
- Одоо буцаж байгаа мөр бүр, давхардсан мөр, тодорхойлсон дараалал, total/page/page_size metadata.
- COUNT/SUM/талбай/дүн, decimal нарийвчлал, round хийх дүрэм, огноо, timezone.
- Жил, байршил, төлөв, сонгосон valuation_type, soft-delete болон бусад шүүлтүүрийн одоогийн логик.
- Authentication, authorization, tenant/organization/assignment scope; `/prof/...` болон external маршрут.
- Transaction, lock, commit/rollback, хэсэгчилсэн амжилт, cancellation, түүх/audit/notification, файл болон гадаад системд үзүүлэх нөлөө.
- Одоогийн synchronous үйлдэл дууссаны дараа одоогийн хариу буцдаг ажиллагаа. Frontend өөрчлөхгүй.

**Одоогийн кодын алдаатай мэт харагдах зан төлөвийг энэ ажлаар засаж болохгүй.** Жишээ: assets жагсаалтын handler pagination авдаг ч repository нь бүх мөрийг буцаадаг. Үүнийг LIMIT нэмээд «засах» нь энэ даалгаврын шаардлагыг зөрчинө. Тусдаа issue болгон тэмдэглээд одоогийн бүх мөрийг хэвээр буцаа.

Хориглох өөрчлөлт:

1. Шинэ pagination/cursor, мөр хасах, field хасах, DTO солих, шүүлтүүр өөрчлөх.
2. TTL cache, хуучирсан materialized summary, ойролцоо COUNT/SUM.
3. `202 Accepted`, job ID, polling, frontend-д шинэ урсгал шаардах.
4. Шинэ 429/503 эсвэл богино timeout-оор амжилттай дуусах ёстой хүсэлтийг тасалж хурдан болсон мэт хэмжих.
5. Geometry simplification, координат/талбай/давхцлын босго өөрчлөх.
6. Тестийн expected-ийг шинэ үр дүнд тааруулж засах, assertion сулруулах, массивыг set болгож мөр/дарааллын ялгааг нуух.

## Эхлээд хийх ажил

1. Repository-ийн AGENTS.md, git status, код болон эцэслэсэн төлөвлөгөөг шалга. Байгаа хэрэглэгчийн өөрчлөлтийг хадгал. Энэ ажлаар deploy/push/merge хийхгүй.
2. Өөрчлөлт хийхээс өмнө хуучин хэрэгжүүлэлтийн reference-ийг хадгал. Ижил fixture-тэй тусдаа DB дээр хуучин/шинэ хувилбарыг ажиллуулж харьцуулах аргыг бэлд. Expected утгыг шинэ кодоос үүсгэж болохгүй.
3. Endpoint → handler → service → repository → SQL зураглал болон одоо байгаа индексүүдийг баталгаажуул. Өмнөх судалгааны мөрийн дугаар, query-ийн тоог сохроор үнэн гэж үзэхгүй.
4. Зөвхөн тусгаарласан түр/test DB-д fixture, EXPLAIN ANALYZE, ачааллын тест ажиллуул. Production өгөгдөл өөрчлөхгүй, үндсэн appdb/authdb-г цэвэрлэхгүй, бодит email/notification/external write үүсгэхгүй. Гадаад нөлөөг test double-оор хяна.

## Хэрэгжүүлэх дараалал

### A. Төлөв шалгалт ба индекс

- `IsAcquisitionLocked`, `AcquisitionStatus`, exists шалгалтад бүтэн `GetByID` ашиглахыг хамгийн бага projection-оор соль. Алдааны зан төлөв, soft-delete болон эрхийн шалгалтыг хадгал.
- Query plan-аар шаардлагатай индексийг сонго. Ялангуяа status history-ийн `(parcel_id, status_date DESC)`, compensation-ийн acquisition/parcel/valuation_type, assets болон draft холбоосын lookup-уудыг шалга.
- Индекс давхардуулахгүй; нэмэлт write/disk cost-ийг хэмж. Migration runner-ийн transaction горимд нийцсэн, урт lock үүсгэхгүй нэвтрүүлэх аргыг бэлд.

### B. Dashboard, report summary

- Ижил parcel/compensation/status history өгөгдлийн давтан scan, correlated subquery, GROUP BY-ийг багасга.
- Dashboard-ийн нийт тоо, статус, timeline, valuation, AU code, acquisition list блок бүрийн одоогийн шүүлтүүрийг тусад нь хадгал. Бүгдэд нэг шүүлтүүр хүчээр хэрэглэхгүй.
- Жилийн шүүлтэд хамгийн сүүлийн status_date-ийг дахин дахин тооцохгүй; join scope зөв хэвээр байлга.
- Report summary-ийн 3 давтагдсан aggregation-ийг нэг удаа тооцож ашиглах хувилбарыг хэмж.
- Numeric нийлбэр, COUNT DISTINCT, selected valuation болон хоосон бүлгийн зан төлөвийг хадгал. JOIN-оор мөр үржихийг тестээр шалга.
- LIMIT 1000 болон гүйцэтгэлийн хувийн эрэмбийг өөрчилж болохгүй. Том нэг SQL автоматаар хурдан гэж үзэхгүй; plan/CPU/IO/p95-аар сонго.

### C. Report download, decision draft, acquisition list

- Одоогийн page/sort-оор буцах мөрүүдийн хүрээнд л шаардлагатай нэгтгэлийг хийх боломжийг ашигла.
- Pagination-ийг JOIN/GROUP BY-ийн өмнө шилжүүлэхээс өмнө cardinality-г батал. Эцсийн мөрийн тоо/дараалал өөрчлөгдвөл тухайн rewrite-ийг бүү хэрэглэ.
- Draft-ийн page доторх parcel/funding/compensation мэдээллийг багцаар нэгтгэ. Одоогийн санхүүжилтийн ангилал, string_agg дарааллыг хадгал.
- `sort_by=parcel_count` зэрэг aggregate sort нь pagination-ээс өмнө зөв бодогдсон байх ёстой. Нийт тоог ойролцоогоор орлуулахгүй.

### D. Хуудаслалтгүй жагсаалт ба санах ой

- Assets, compensation, valuation snapshots, filter-options-ийн бүх мөр, бүх field-ийг хадгалж индекс, projection, entity/DTO хуулбар, JSON decode/encode-ийн давхардлыг багасга.
- Snapshot nested JSON-ийг raw хэлбэрээр дамжуулах боломжийг шалга; утга/төрөл/нарийвчлал өөрчлөхгүй.
- Шууд streaming хийснээр SQL алдаанаас өмнө HTTP 200 бичигдэх эрсдэл үүсгэхгүй.
- Plan suggest, audit, client-request query-уудын filter/sort/count-ийг хэмжилтээр оновчил.

### E. Discovery, sync, import, geometry, snapshot creation, drone

- Мөр бүрийн SQL-ийг bounded batch/staging/COPY аргаар багасгах боломжийг ашигла. Duplicate, ownership, invalid geometry fallback, generated ID холбоос, history, sort_index, partial success, replacement cleanup болон rollback хэвээр байна.
- Fetch-then-write-ийг streaming write болгож алдаа/цуцлалтын үед өөр өгөгдөл үлдээж болохгүй. Transaction хил, алдааны зан төлөвт нийцэх аргыг сонго.
- Geometry candidate pair-ийг индексээр нарийсгаж, нэг geometry-ийн давтан MakeValid тооцоог дахин ашигла. Raw Intersects ба valid intersection/UTM area-ийн одоогийн ялгааг хадгал. Үндсэн хувилбарт full recomputation хадгал.
- Snapshot-ийг lock авсны дараах зөв өгөгдлөөс үүсгэ. Үнэлгээ цуцлах/илгээх үед stale precompute хэрэглэж болохгүй.
- Drone header/configure/publish-ийн дотоод давхардлыг багасга. Reset-ийн тоо/дарааллыг өөрчлөхийг энэ хүрээнд хийхгүй. External visibility/partial failure өөрчлөх concurrency оруулахгүй.

### F. Зэрэгцээ ачаалал

- Query-уудыг хөнгөлсний дараа хүнд ажлын concurrency-г хэмжилтээр тохируул. Pool size-ийг өсгөхийг үндсэн шийдэл болгохгүй.
- Semaphore/queue хэрэглэх бол DB connection барьж permit хүлээхгүй; request context/cancellation-ийг хадгал; шинэ rejection response нэмэхгүй.
- Олон query-г нэг SQL болгох, lock/queue нэмэх нь concurrent write үед snapshot, харагдах хугацаа, error precedence-д нөлөөлж болно. Үүнийг тусгай тестээр шалга; ижил зан төлөв батлагдахгүй бол хуучин зохион байгуулалтыг хадгалсан өөр оновчлол сонго.

## Тестийн заавал хангах нөхцөл

- Хоосон DB, null/default, олон child row, олон valuation stream, soft-delete, жил солигдох/timezone зааг, дүр бүрийн эрх, эхний/гүн/хоосон page, ижил sort утга, geometry boundary/containment/invalid input, import partial error болон concurrent write-ийг хамруул.
- JSON object key дарааллыг үл тооцож болно; array order/duplicates/null/numeric/date ялгааг нууж болохгүй. Эх кодод тогтворгүй array order байвал baseline дээр нотолж тусад нь тайлбарла, дур мэдэн тогтвортой болгож өөрчлөхгүй.
- Dynamic clock/ID-г хяналттай тогтмолжуул эсвэл холбоос/инвариантаар батал; created_at/status_date/detected_at-ийг бөөнөөр нь хасаж харьцуулахгүй.
- Бичих урсгалд эцсийн DB state, түүх/audit/notification, external нөлөө болон partial failure-ийг харьцуул. Нэг API хариу ижил байх нь дангаараа хангалтгүй.
- Бизнесийн ялгаа гарвал тухайн өөрчлөлтийг засах/буцаах; expected-ийг сулруулахгүй. Нийцэл нь батлагдсан бусад ажлыг үргэлжлүүл.
- Frontend репо эсвэл browser шалгалт боломжгүй бол backend contract баталсан хүрээгээ тайлагна; «дэлгэцийг бүрэн шалгасан» гэж бүү хэл.

## Хурд хэмжих арга

Өмнөх локал суурь нь 100 acquisition, 10,000 parcel, 30,000 compensation, 10,000 asset, 50,000 status history, 10,000 valuation submission байсан. 10 зэрэгцээ repository дуудалтын дундаж/хамгийн удаан: dashboard 3.00/6.87 сек; жилтэй dashboard 8.20/10.04 сек; report summary 2.10/9.11 сек; report download эхний 20 мөр 0.390/0.735 сек. Энэ нь ARM дээр amd64 PostgreSQL image ажиллуулсан локал тест; production SLA биш. Үүнийг одоогийн өөр орчны benchmark-тай шууд харьцуулахгүй, хуучин/шинэ хувилбарыг нэг орчинд дахин хэмж.

- 1 ба 10 зэрэгцээ хэрэглэгч, дан endpoint болон холимог хүсэлт, warm/cold cache, ердийн болон муу query plan гардаг шүүлтүүдээр харьцуул.
- Repository болон HTTP-ийн бүтэн хугацааг тусад нь хэмж. Frontend-ийн request mix мэдэгдэхгүй бол таамагласан хувилбар гэж тодорхой тэмдэглэ.
- 10 зэрэгцээ горимд дор хаяж 300 хүсэлтийн p50/p95/p99, throughput, error/timeout, pool wait, SQL count/time, CPU/RAM/temp disk, response bytes-ийг бүртгэ.
- Өмнөх fixture дээр хүнд уншилтын p95-ийг 50%-иас дээш бууруулахыг зорь. Үр дүнгийн нийцэл зөрчиж энэ зорилтыг биелүүлэхгүй.
- Test-only safety timeout хэрэглэж болно. Timeout болсон хүсэлтийг амжилттай дууссан latency-д оруулахгүй; бүх failure-ийг тайлагна. Бодит API-ийн timeout-ийг энэ шалтагтай өөрчилж болохгүй.
- Бүх мөрийг буцаах сүлжээний зардал, N² хүртэл өсөх geometry pair, гадаад системийн хүлээлтийн үлдэх хязгаарыг ил тод тайлагна.

## Ажлыг дуусгах нөхцөл ба тайлан

Нийцэл батлагдаагүй өөрчлөлтийг «дууссан» гэж бүү үз. Шалгах боломжгүй хэсэгт хуучин зан төлөвийг хадгалж, хязгаарлалтаа тодорхой тэмдэглэ. Бусад хэрэгжүүлж болох ажлыг дуусга; шаардлагагүй зөвшөөрөл давтан бүү асуу.

Эцэст нь дараахыг өг:

1. Өөрчилсөн API/файл, арилгасан давтан ажил, migration ба rollback арга.
2. Хуучин/шинэ үр дүн ижил гэдгийг нотлох тестийн жагсаалт, бодит ажиллуулсан команд ба үр дүн.
3. Ижил орчинд 1/10 хэрэглэгчийн өмнөх/дараах хэмжилтийн хүснэгт, dataset ба тоног төхөөрөмжийн нөхцөл.
4. Батлагдаагүй зүйл, үлдсэн өртөг, шаардлага зөрчих тул хийгээгүй санал, тусдаа илрүүлсэн бизнесийн алдаа.
5. Үр дүнгийн ялгаа байгаа эсэхийг шууд хэл. Тест/ачааллын хэмжилт хийгээгүй бол хийсэн мэт эсвэл секундийн таамгийг хэмжилт мэт тайлагнахгүй.

Одоо төлөвлөгөө болон repository-г уншаад P0 нийцлийн хамгаалалтаас эхэлж, дарааллаар хэрэгжүүл.
