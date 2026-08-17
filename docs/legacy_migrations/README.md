# Хуучин миграцууд (архив)

`main/` — appdb-ийн 000001…000096 миграцууд. Эдгээрийн ЭЦСИЙН үр дүнг
`migrations/main/000001_appdb_baseline.up.sql` баримтжуулсан тул шинэ суулгацад
эдгээр хэрэггүй. Түүх судлах, `down` замыг харах, эсвэл ямар өөрчлөлт хэзээ
орсныг тогтооход хэрэглэнэ.

Байгаа appdb-г baseline руу тамгалах:
    migrate -path migrations/main -database "$MAIN_DATABASE_URL" force 1
