# Ажиллуулах журам — миграц 000031–000033

Гүйцэтгэх: DB админ / deploy хариуцагч.

Эдгээр гурван миграц нь энгийн `make migrate-main-up`-аар өнгөрөх боловч
хоёр нь production-д **нэмэлт болгоомжлол шаардана**. Дараах дарааллаар явна.

---

## 1. 000031 — `parcel_status_history` индекс

```sql
CREATE INDEX idx_psh_parcel_status_date ON parcel_status_history (parcel_id, status_date DESC);
DROP INDEX idx_psh_parcel;
```

### Эрсдэл

`CREATE INDEX` (CONCURRENTLY биш) нь индекс бүтээх ХУГАЦААНД
`parcel_status_history`-д **БИЧИХ үйлдлийг түгжинэ** (унших саадгүй). Нэгж
талбарын төлөв солих бүрд энэ хүснэгтэд мөр нэмэгддэг тул түгжээ нь
хэрэглэгчид шууд мэдрэгдэнэ.

`golang-migrate` нь миграц бүрийг ГҮЙЛГЭЭНД ажиллуулдаг ба
`CREATE INDEX CONCURRENTLY` нь гүйлгээнд ажиллах БОЛОМЖГҮЙ. Тиймээс
миграцын файлыг CONCURRENTLY болгож өөрчилж БОЛОХГҮЙ.

### Хүснэгт жижиг бол (< ~500k мөр)

Шууд ажиллуулна:

```bash
make migrate-main-up
```

Мөрийн тоог урьдчилж шалга:

```sql
SELECT count(*) FROM parcel_status_history;
```

### Хүснэгт том бол

Индексийг миграцаас **ӨМНӨ гараар**, гүйлгээнээс гадуур үүсгэнэ. Дараа нь
миграц дотрох `CREATE INDEX IF NOT EXISTS` нь юу ч хийхгүй өнгөрнө.

```bash
# 1. Гүйлгээнээс ГАДУУР (psql нь өгөгдмөлөөр autocommit).
psql "$MAIN_DSN" -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_psh_parcel_status_date
                       ON public.parcel_status_history USING btree (parcel_id, status_date DESC);"

# 2. Амжилттай эсэхийг ЗААВАЛ шалга: CONCURRENTLY нь бүтэлгүйтэхдээ
#    INVALID индекс үлдээдэг ба тэр нь хайлтад ашиглагдахгүй.
psql "$MAIN_DSN" -c "SELECT indexrelid::regclass, indisvalid
                       FROM pg_index
                      WHERE indexrelid = 'idx_psh_parcel_status_date'::regclass;"
#    indisvalid = false бол: DROP INDEX CONCURRENTLY хийж дахин үүсгэнэ.

# 3. Миграцыг ажиллуулна (индекс аль хэдийн байгаа тул зөвхөн хуучныг хаяна).
make migrate-main-up
```

### Шалгах

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT MAX(status_date) FROM parcel_status_history WHERE parcel_id = '<аль нэг код>';
```

`Index Only Scan using idx_psh_parcel_status_date` гарч, `Heap Fetches: 0`
байх ёстой.

---

## 2. 000032 — `parcel_status.color`

DDL нь зөвхөн багана + CHECK нэмнэ (өгөгдөл хөндөхгүй). Одоо байгаа
төлөвүүдийн өнгө нь NULL үлдэх тул миграцын ДАРАА seeder ажиллуулна:

```bash
go run ./cmd/seed-sql -db main scripts/seed_parcel_status_color.sql
```

Seeder нь давтан ажиллаж болно: зөвхөн `color IS NULL` мөрийг бөглөх тул
гараар өөрчилсөн өнгийг дарж бичихгүй.

---

## 3. 000033 — `v_parcel_status` харагдац

Онцгой болгоомжлол шаардахгүй. Хандалтын хяналт нь ЭНЭ шилжилтээр
өөрчлөгдөхгүй — доорх шалгалтаар баталгаажсан.

### Хандалтын хяналт хаана байдаг вэ

GeoServer нь **өөрөө эрх шалгадаггүй** бөгөөд `127.0.0.1:8600`-д л
нийтлэгддэг (`government-geoserver/docker-compose.yml`). Түүний `make config`
нь workspace / datastore / featuretype / style л үүсгэдэг — давхаргын
түвшний эрхийн дүрэм (`layers.properties`, role) **байхгүй**.

Бүх хандалт frontend-ийн хоёр гарцаар явна:

| Гарц | Давхарга |
|---|---|
| `government-frontend/src/app/api/geoserver/[...path]` | Дотоод. Нэвтрэлт заавал; цагаан жагсаалт (fail-closed) |
| `government-frontend/src/app/api/public/geoserver/[...path]` | Нээлттэй. `v_parcel_public` болгож **хатуу дардаг** — өөр давхарга гарахгүй |

### Яагаад шилжилт аюулгүй вэ

Дотоод гарц дээр `v_parcel_s0…s5` зургуулаа **`PARCEL_STATUS_LAYERS` гэсэн
НЭГ ангилалд** байсан бөгөөд яг ижилхэн хандлагатай байв — төлвөөр эрх
ялгадаг байсангүй. Хумилт нь **`acquisition_id`-аар** хийгддэг:

```ts
const scope = `acquisition_id IN (${allowedAcquisitionIds…})`
setParamCaseInsensitive(params, 'CQL_FILTER', existing ? `(${existing}) AND ${scope}` : scope)
```

Гурван чухал зүйл:

1. **`status` нь хэзээ ч аюулгүй байдлын хил байгаагүй.** Клиент
   `CQL_FILTER=status=N`-ийг өөрчилж болох боловч өмнө нь ч өөр
   `v_parcel_sN` давхаргыг дуудаад яг тэр үр дүнд хүрч чаддаг байсан.
2. **Жинхэнэ хил нь `acquisition_id`** бөгөөд гарц түүнийг клиентийн
   CQL дээр `AND`-ээр **нэмдэг** (дарж бичдэггүй) тул тойрох боломжгүй.
   `v_parcel_status` нь `acquisition_id` баганатай тул хумилт ижилхэн
   ажиллана.
3. **Өнгө нь клиентээс ирэхгүй.** SLD нь харагдацын `color` баганаас
   (`COALESCE(ps.color, '#94a3b8')`, эх нь `parcel_status` хүснэгт)
   `<PropertyName>color</PropertyName>`-ээр шууд уншина. Клиент өнгийг
   өөрчилж чадахгүй тул зураг бүртгэлтэй ҮРГЭЛЖ тохирно.

Frontend-ийн гарц **аль хэдийн шинэчлэгдсэн**: `v_parcel_status` нь
`PARCEL_STATUS_LAYERS`-д нэмэгдсэн, хуучин `v_parcel_s0…s5` нэрс шилжилтийн
хугацаанд зэрэгцэн зөвшөөрөгдөнө.

### Өнгө хаанаас ирдэг вэ

```
parcel_status.color  (бүртгэл, админ цэсээр засна)
      ↓  LEFT JOIN
v_parcel_status.color  (COALESCE-ээр саарал өгөгдмөл)
      ↓  <ogc:PropertyName>color</ogc:PropertyName>
parcel_status.sld / parcel_status_print.sld
```

Өнгө солиход **зөвхөн бүртгэлийн мөрийг** засна: GeoServer-т ч, frontend-д ч
ямар ч өөрчлөлт, дахин ачаалалт шаардахгүй.

### Гүйцэтгэх

```bash
make migrate-main-up
go run ./cmd/seed-sql -db main scripts/seed_parcel_status_color.sql   # §2, өнгө бөглөнө
cd ../government-geoserver && make config   # v_parcel_status давхарга + parcel_status.sld
```

> **ДАРААЛАЛ ЧУХАЛ:** seeder өнгө бөглөхөөс өмнө `make config` ажиллуулбал
> давхарга үүснэ ч бүх төлөв саарлаар зурагдана (`color` нь NULL тул COALESCE
> өгөгдмөлд унана). Seeder-ийн дараа дахин зурахад засагдана —
> GeoServer-ийг дахин ачаалах шаардлагагүй.

Хуучин `v_parcel_s0…s5` нь ЭНД устгагдахгүй: GeoServer-ийн каталогт
давхарга болж бүртгэлтэй тул урьдчилан устгавал тэдгээр давхарга дээр алдаа
өгнө. Frontend шинэ давхаргад бүрэн шилжиж, GeoServer-ийн каталогоос хуучин
давхаргууд хасагдсаны ДАРАА тусдаа миграцаар устгана. Тэр үед дотоод гарцын
`PARCEL_STATUS_LAYERS`-аас хуучин нэрсийг мөн хасна.

---

## Буцаах

```bash
make migrate-main-down   # нэг алхмаар буцаана, шаардлагатай тоогоор давтана
```

000031-ийн `down` нь хуучин `idx_psh_parcel`-ыг ЭХЛЭЭД сэргээж байж шинийг
хаядаг — хоёрын хооронд `parcel_id`-гаар хайх асуулга индексгүй үлдэхгүй.
Энэ нь мөн энгийн `CREATE INDEX` тул том хүснэгт дээр буцаахад ижил түгжээ
үүснэ.
