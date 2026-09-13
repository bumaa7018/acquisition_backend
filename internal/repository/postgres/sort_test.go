package postgres

import (
	"strings"
	"testing"
)

// Хэрэглэгчийн өгсөн эрэмбийн түлхүүр SQL-д ШУУД ОРОХГҮЙ: whitelist-д
// байхгүй бүхэн үндсэн эрэмбэ рүү унана.
func TestOrderClause_RejectsUnknownAndInjection(t *testing.T) {
	allowed := map[string]string{"parcel_id": "p.parcel_id"}
	// Үндсэн эрэмбэ + тогтвортой хоёрдогч эрэмбэ.
	const want = "p.created_at DESC, p.id"

	for _, key := range []string{
		"", "unknown_column", "p.parcel_id", "1",
		"parcel_id; DROP TABLE parcel", "parcel_id--", "(SELECT 1)",
	} {
		got := orderClause(allowed, key, "asc", "p.created_at DESC", "p.id")
		if got != want {
			t.Fatalf("key=%q: хүлээсэн %q, гарсан %q", key, want, got)
		}
	}
}

func TestOrderClause_BuildsWhitelistedColumn(t *testing.T) {
	allowed := map[string]string{"area_m2": "p.area_m2"}

	got := orderClause(allowed, "area_m2", "asc", "p.created_at DESC", "p.id")
	if got != "p.area_m2 ASC NULLS LAST, p.id" {
		t.Fatalf("өсөх эрэмбэ буруу: %q", got)
	}
	// Чиглэл нь ЗӨВХӨН asc/desc — том жижиг үсэг, зай хамаарахгүй.
	got = orderClause(allowed, "area_m2", " DESC ", "p.created_at DESC", "p.id")
	if got != "p.area_m2 DESC NULLS LAST, p.id" {
		t.Fatalf("буурах эрэмбэ буруу: %q", got)
	}
	// Танигдаагүй чиглэл — өсөхөөр тооцно (буруу утга алдаа үүсгэхгүй).
	got = orderClause(allowed, "area_m2", "; DELETE FROM parcel", "p.created_at DESC", "p.id")
	if got != "p.area_m2 ASC NULLS LAST, p.id" {
		t.Fatalf("буруу чиглэлийг өсөхөөр авах ёстой: %q", got)
	}
}

// Хуудаслалттай жагсаалтад тогтвортой хоёрдогч эрэмбэ ЗААВАЛ байна — эс
// бөгөөс ижил утгатай мөрүүд хуудас хооронд үсэрч, давхардаж эсвэл алга болно.
func TestSortColumnMaps_HaveNoRawUserInput(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"acquisition":   acquisitionSortColumns,
		"global_parcel": globalParcelSortColumns,
		"parcel":        parcelSortColumns,
	} {
		for key, expr := range m {
			if key != strings.ToLower(key) {
				t.Errorf("%s: %q түлхүүр жижиг үсгээр байх ёстой", name, key)
			}
			if strings.Contains(expr, ";") {
				t.Errorf("%s.%s: илэрхийлэлд ';' байж болохгүй", name, key)
			}
		}
	}
}
