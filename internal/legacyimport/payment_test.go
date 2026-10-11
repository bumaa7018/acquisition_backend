package legacyimport

import "testing"

// Архивын хүснэгтийн дүн ÷ батлагдсан нийт олговор → олголтын шат.
func TestPaidStage(t *testing.T) {
	for _, tc := range []struct {
		name        string
		paid, total float64
		want        int
	}{
		{"бүтэн олгосон", 166_937_000, 166_937_000, 100},
		{"бүхэл төгрөгийн зөрүү — бүтэн", 166_936_999.6, 166_937_000, 100},
		{"хүснэгтийн дүн илүү — бүтэн", 200, 150, 100},
		{"60% олгосон", 60, 100, 60},
		{"60-аас дээш, бүтэн биш — 60%", 80, 100, 60},
		{"60-аас доош — олгоогүй", 59, 100, 0},
		{"хүснэгтэд дүн байхгүй", 0, 100, 0},
		{"батлагдсан олговоргүй", 100, 0, 0},
	} {
		if got := paidStage(tc.paid, tc.total); got != tc.want {
			t.Errorf("%s: paidStage(%v, %v) = %d, хүлээгдэж буй %d", tc.name, tc.paid, tc.total, got, tc.want)
		}
	}
}

// Эхний явцын огноо — төлөвлөгөө батлагдсан огноо (текст), буруу бол хоосон.
func TestPlanApprovedDate(t *testing.T) {
	for in, want := range map[string]string{
		"2025-12-05":           "2025-12-05",
		"2025.12.05":           "2025-12-05",
		"2025-12-05T00:00:00Z": "2025-12-05",
		"2025-02-31":           "", // байхгүй өдөр — ::date алдаа гаргахгүй
		"":                     "",
		"тодорхойгүй":          "",
	} {
		if got := planApprovedDate(in); got != want {
			t.Errorf("planApprovedDate(%q) = %q, хүлээгдэж буй %q", in, got, want)
		}
	}
}
