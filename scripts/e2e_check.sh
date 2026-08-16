#!/bin/bash
# sdplatform шилжилтийн дараах бүрэн E2E шалгалт
BASE=http://localhost:8099/api/v1
AP=$(grep '^ADMIN_PASSWORD=' .env | cut -d= -f2-)
PASS=0; FAIL=0
declare -a FAILED

chk() { # chk "нэр" "хүлээгдэх" "бодит"
  if [ "$2" = "$3" ]; then PASS=$((PASS+1)); printf "  ✓ %-52s %s\n" "$1" "$3"
  else FAIL=$((FAIL+1)); FAILED+=("$1 (хүлээсэн $2, ирсэн $3)"); printf "  ✗ %-52s %s (хүлээсэн %s)\n" "$1" "$3" "$2"; fi
}
req() { curl -s -o /tmp/e2e.json -w '%{http_code}' "$@"; }
val() { python3 -c "import json;d=json.load(open('/tmp/e2e.json'));print($1)" 2>/dev/null || echo ""; }

echo "═══ 1. НЭВТРЭЛТ ═══"
c=$(req -X POST $BASE/auth/login -H 'Content-Type: application/json' -d "{\"username\":\"admin@example.com\",\"password\":\"$AP\"}")
chk "POST /auth/login (admin)" 200 "$c"
TOK=$(val 'd["data"]["access_token"]'); RTOK=$(val 'd["data"]["refresh_token"]')
H=(-H "Authorization: Bearer $TOK" -H 'Content-Type: application/json')

python3 -c "
import json,base64
t='$TOK'.split('.')[1]; t+='='*(-len(t)%4)
c=json.loads(base64.urlsafe_b64decode(t))
assert isinstance(c['user_id'],int), 'user_id int биш'
assert 'admin' in c['roles'], 'admin роль алга'
assert 'land:read' in c['permissions'], 'land:read эрх алга'
print('  ✓ JWT: user_id=%s (int), roles=%s, %d эрх' % (c['user_id'], c['roles'], len(c['permissions'])))
" || { FAIL=$((FAIL+1)); FAILED+=("JWT бүтэц"); }

chk "POST /auth/login (буруу нууц үг)" 401 "$(req -X POST $BASE/auth/login -H 'Content-Type: application/json' -d '{"username":"admin@example.com","password":"wrong"}')"
chk "POST /auth/refresh" 200 "$(req -X POST $BASE/auth/refresh -H 'Content-Type: application/json' -d "{\"refresh_token\":\"$RTOK\"}")"
chk "GET  без токен (401)" 401 "$(req $BASE/users)"
chk "GET  буруу токен (401)" 401 "$(req $BASE/users -H 'Authorization: Bearer bad.token.here')"

echo "═══ 2. ХЭРЭГЛЭГЧ ═══"
chk "GET  /users" 200 "$(req "${H[@]}" "$BASE/users?page=1&page_size=5")"
chk "GET  /users/me" 200 "$(req "${H[@]}" $BASE/users/me)"
MYID=$(val 'd["data"]["id"]')
MEID=$MYID
MYEMP=$(val 'd["data"]["employee_id"]')
chk "  /users/me id нь бүхэл тоо" True "$(python3 -c "import json;print(isinstance(json.load(open('/tmp/e2e.json'))['data']['id'],int))")"
chk "GET  /users/:id" 200 "$(req "${H[@]}" $BASE/users/$MEID)"
chk "GET  /users?search=admin" 200 "$(req "${H[@]}" "$BASE/users?search=admin")"
chk "GET  /users?role=admin" 200 "$(req "${H[@]}" "$BASE/users?role=admin")"
NU="e2e_$(date +%s)"
req "${H[@]}" "$BASE/positions?page=1&page_size=1" >/dev/null; UPOS=$(val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
req "${H[@]}" "$BASE/departments?page=1&page_size=1" >/dev/null; UDEP=$(val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
UREG="V$(date +%s | tail -c 9)"
c=$(req -X POST "${H[@]}" $BASE/users -d "{\"username\":\"$NU\",\"email\":\"$NU@e2e.mn\",\"password\":\"Testpass123!\",\"first_name\":\"E2E\",\"last_name\":\"Тест\",\"employee\":{\"person\":{\"person_type\":\"citizen\",\"register_no\":\"$UREG\",\"last_name\":\"E2E\",\"first_name\":\"Хэрэглэгч\"},\"department_id\":$UDEP,\"position_id\":$UPOS}}")
chk "POST /users (ажилтантай)" 201 "$c"; NEWU=$(val 'd["data"]["id"]')
req "${H[@]}" $BASE/users/me >/dev/null; MYEMP=$(val 'd["data"]["employee_id"]')
req "${H[@]}" $BASE/users/me >/dev/null; MYEMP=$(val 'd["data"]["employee_id"]')
DUP=$(req -X POST "${H[@]}" $BASE/users -d "{\"username\":\"admin\",\"email\":\"dup@e2e.mn\",\"password\":\"Testpass123!\",\"first_name\":\"A\",\"last_name\":\"B\",\"employee_id\":$MYEMP}")
chk "POST /users (давхардсан нэр → 409)" 409 "$DUP"
chk "PUT  /users/:id (users:update алга → 403)" 403 "$(req -X PUT "${H[@]}" $BASE/users/$NEWU -d '{"first_name":"Засав","last_name":"Тест","is_active":true}')"

echo "═══ 3. РОЛЬ, ЭРХ ═══"
chk "GET  /roles" 200 "$(req "${H[@]}" $BASE/roles)"
RID=$(val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
chk "GET  /roles/:id" 200 "$(req "${H[@]}" $BASE/roles/$RID)"
python3 -c "
import json;d=json.load(open('/tmp/e2e.json'))['data']
assert d.get('resource'), 'resource алга'
p=d.get('permissions') or []
assert all(':' in x['name'] for x in p), 'эрх resource:action биш'
print('  ✓ роль: %s [resource=%s], эрх: %s' % (d['name'], d['resource'], [x['name'] for x in p]))
" || { FAIL=$((FAIL+1)); FAILED+=("роль бүтэц"); }
chk "GET  /permissions" 200 "$(req "${H[@]}" $BASE/permissions)"
chk "POST /users/:id/roles (roles:update алга → 403)" 403 "$(req -X POST "${H[@]}" $BASE/users/$NEWU/roles -d "{\"role_id\":\"$RID\"}")"
chk "DELETE /users/:id/roles/:rid (403)" 403 "$(req -X DELETE "${H[@]}" $BASE/users/$NEWU/roles/$RID)"

echo "═══ 4. ЦЭС (шинэ) ═══"
chk "GET  /menus" 200 "$(req "${H[@]}" $BASE/menus)"
req "${H[@]}" $BASE/roles/$RID/menus >/dev/null; MID=$(val 'd["data"][0]["id"]')
chk "GET  /menus/me" 200 "$(req "${H[@]}" $BASE/menus/me)"
chk "GET  /roles/:id/menus" 200 "$(req "${H[@]}" $BASE/roles/$RID/menus)"
chk "DELETE /roles/:id/menus/:mid" 204 "$(req -X DELETE "${H[@]}" $BASE/roles/$RID/menus/$MID)"
chk "POST /roles/:id/menus" 201 "$(req -X POST "${H[@]}" $BASE/roles/$RID/menus -d "{\"menu_id\":$MID}")"

echo "═══ 5. HR ЛАВЛАХ ═══"
for ep in organizations departments positions persons employees; do
  chk "GET  /$ep" 200 "$(req "${H[@]}" "$BASE/$ep?page=1&page_size=5")"
done
DEP=$(req "${H[@]}" "$BASE/departments?page=1&page_size=1" >/dev/null; val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
req "${H[@]}" "$BASE/positions?page=1&page_size=1" >/dev/null; POS=$(val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
req "${H[@]}" "$BASE/departments?page=1&page_size=1" >/dev/null; DEP=$(val 'd["data"][0]["id"] if isinstance(d["data"],list) else d["data"]["items"][0]["id"]')
R10="U$(date +%s | tail -c 9)"
c=$(req -X POST "${H[@]}" $BASE/employees -d "{\"person\":{\"person_type\":\"citizen\",\"register_no\":\"$R10\",\"last_name\":\"Тестов\",\"first_name\":\"Тест\",\"gender\":\"M\",\"civil_id\":\"9911\",\"address\":\"УБ\"},\"department_id\":$DEP,\"position_id\":$POS,\"work_email\":\"e2e@e.mn\"}")
chk "POST /employees (хүнтэй)" 201 "$c"; EMP=$(val 'd["data"]["id"]')
chk "GET  /employees/:id" 200 "$(req "${H[@]}" $BASE/employees/$EMP)"
python3 -c "
import json;d=json.load(open('/tmp/e2e.json'))['data'];p=d['person']
assert d.get('organization_name'), 'байгууллага хэлтсээр гарахгүй байна'
assert p['gender']=='M' and p['civil_id']=='9911' and p['address']=='УБ'
print('  ✓ ажилтан: %s | %s | %s | pos=%r' % (d['person_name'],d['organization_name'],d['department_name'],d.get('position_name','')))
" || { FAIL=$((FAIL+1)); FAILED+=("ажилтны талбарууд"); }
chk "POST /employees (регистр >10 → 400)" 400 "$(req -X POST "${H[@]}" $BASE/employees -d "{\"person\":{\"person_type\":\"citizen\",\"register_no\":\"УБ12345678901\",\"last_name\":\"A\",\"first_name\":\"B\"},\"department_id\":$DEP,\"position_id\":$POS}")"
chk "DELETE /employees/:id (hr:delete алга → 403)" 403 "$(req -X DELETE "${H[@]}" $BASE/employees/$EMP)"
chk "GET  ажилтан хэвээр (200)" 200 "$(req "${H[@]}" $BASE/employees/$EMP)"
PN="E2E тушаал $(date +%s)"
chk "POST /positions" 201 "$(req -X POST "${H[@]}" $BASE/positions -d "{\"name\":\"$PN\"}")"
NPOS=$(val 'd["data"]["id"]')
chk "PUT  /positions/:id (hr:update алга → 403)" 403 "$(req -X PUT "${H[@]}" $BASE/positions/$NPOS -d '{"name":"Засав"}')"

echo "═══ 6. ГАЗАР ЧӨЛӨӨЛӨЛТ ═══"
for ep in land-acquisitions parcels compensations acquisition-categories document-types parcel-statuses acquisition-progress-statuses funding-sources construction-types; do
  chk "GET  /$ep" 200 "$(req "${H[@]}" "$BASE/$ep?page=1&page_size=3")"
done
chk "GET  /plans/suggest" 200 "$(req "${H[@]}" "$BASE/plans/suggest?q=LA")"
chk "GET  /dashboard" 200 "$(req "${H[@]}" $BASE/dashboard)"
chk "GET  /report/summary" 200 "$(req "${H[@]}" $BASE/report/summary)"
req "${H[@]}" "$BASE/land-acquisitions?page=1&page_size=1" >/dev/null
ACQ=$(val 'd["data"]["items"][0]["id"] if isinstance(d["data"],dict) else d["data"][0]["id"]')
if [ -n "$ACQ" ]; then
  chk "GET  /land-acquisitions/:id" 200 "$(req "${H[@]}" $BASE/land-acquisitions/$ACQ)"
  chk "GET  /land-acquisitions/:id/parcels" 200 "$(req "${H[@]}" "$BASE/land-acquisitions/$ACQ/parcels?page=1&page_size=3")"
  chk "GET  /land-acquisitions/:id/documents" 200 "$(req "${H[@]}" $BASE/land-acquisitions/$ACQ/documents)"
  chk "GET  /land-acquisitions/:id/assets" 200 "$(req "${H[@]}" "$BASE/land-acquisitions/$ACQ/assets?page=1&page_size=3")"
  chk "GET  /land-acquisitions/:id/compensations" 200 "$(req "${H[@]}" $BASE/land-acquisitions/$ACQ/compensations)"
fi

echo "═══ 7. МЭДЭГДЭЛ, АУДИТ ═══"
chk "GET  /notifications" 200 "$(req "${H[@]}" "$BASE/notifications?page=1&page_size=5")"
chk "GET  /notifications/unread-count" 200 "$(req "${H[@]}" $BASE/notifications/unread-count)"
chk "GET  /audit-logs (энэ DB-д admin-д audit:read алга)" 403 "$(req "${H[@]}" "$BASE/audit-logs?page=1&page_size=5")"

echo "═══ 8. ЭРХИЙН ХЯЗГААРЛАЛТ ═══"
c=$(req -X POST $BASE/auth/login -H 'Content-Type: application/json' -d "{\"username\":\"senior.specialist@example.mn\",\"password\":\"$AP\"}")
if [ "$c" = "200" ]; then
  ET=$(val 'd["data"]["access_token"]'); EH=(-H "Authorization: Bearer $ET")
  chk "senior: GET /land-acquisitions (200)" 200 "$(req "${EH[@]}" "$BASE/land-acquisitions?page=1&page_size=1")"
  chk "senior: GET /users (users:read бий → 200)" 200 "$(req "${EH[@]}" $BASE/users)"
  chk "senior: POST /users (403)" 403 "$(req "${EH[@]}" -H 'Content-Type: application/json' -X POST $BASE/users -d '{"username":"x","email":"x@x.mn","password":"Testpass123!","first_name":"a","last_name":"b"}')"
  chk "senior: GET /menus/me (200)" 200 "$(req "${EH[@]}" $BASE/menus/me)"
else echo "  ⚠ senior_specialist нэвтэрч чадсангүй ($c)"; fi

echo "═══ 9. ЦЭВЭРЛЭГЭЭ ═══"
chk "DELETE /users/:id (users:delete алга → 403)" 403 "$(req -X DELETE "${H[@]}" $BASE/users/$NEWU)"
chk "POST /auth/logout" 200 "$(req -X POST "${H[@]}" $BASE/auth/logout -d "{\"refresh_token\":\"$RTOK\"}")"

echo
echo "════════════════════════════════════"
echo "  ДҮН:  $PASS амжилттай / $FAIL алдаа"
[ $FAIL -gt 0 ] && { echo "  Алдаанууд:"; for f in "${FAILED[@]}"; do echo "   - $f"; done; }
echo "════════════════════════════════════"
