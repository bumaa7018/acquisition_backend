package legacyimport

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ── «ЗӨВХӨН ҮНЭЛГЭЭ» ГОРИМ — CLI-ийн архивын импорттой (archive-map.ts) ИЖИЛ ──
//
// SHP байхгүй тул чөлөөлөлтийг «Бүтээн байгуулалтын ажлын нэр»-ээр бүлэглэж
// ХИЛГҮЙ үүсгэнэ, ангиллыг «Бүтээн байгуулалтын төрөл»-өөс авна. Гар бөглөсөн
// тайлан тул нэг төсөл олон янзаар бичигдсэн байдаг:
//
//	Сэлбэ дэд төв            ↔ СЭЛБЭ ДЭД ТӨВ
//	Сонгинохайрхан …         ↔ Сснгинохайрхан …      (үсгийн алдаа)
//
// АНХААР — ТОО нь төслийн ТАНИХ ТЭМДЭГ («… 2 дугаар блок» ↔ «… 1 дүгээр блок»
// нь ӨӨР төсөл). Бүлэглэх болзол: ТООН ГАРЫН ҮСЭГ ЯГ ИЖИЛ, үсгийн зөрүү ≤6%.

var (
	reQuotes = regexp.MustCompile(`[“”„«»"']`)
	reDigits = regexp.MustCompile(`\d+`)
)

// canonName — хашилт, цэг таслал, зайг арилгаж, жижиг үсгээр.
func canonName(value string) string {
	value = reQuotes.ReplaceAllString(strings.ToLower(value), "")
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nameDigits — нэр доторх бүх тоо дарааллаар ("09" ба "9" ижил).
func nameDigits(value string) string {
	parts := reDigits.FindAllString(value, -1)
	for i, p := range parts {
		if n, err := strconv.Atoi(p); err == nil {
			parts[i] = strconv.Itoa(n)
		}
	}
	return strings.Join(parts, ".")
}

// editDistance — Levenshtein (rune-ээр).
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func nameTolerance(a, b string) int {
	return max(1, max(len([]rune(a)), len([]rune(b)))*6/100)
}

// nameCluster — нэг төсөл: Canonical нь хамгийн олон давтагдсан бичиглэл.
type nameCluster struct {
	Canonical string
	Variants  []string
}

// clusterNames — бичиглэл → мөрийн тоо. Буцаах: бичиглэл → ӨВӨРМӨЦ нэр.
func clusterNames(counts map[string]int) (map[string]string, []nameCluster) {
	type item struct {
		value, key, digits string
		count              int
	}
	items := make([]item, 0, len(counts))
	for v, c := range counts {
		items = append(items, item{value: v, key: canonName(v), digits: nameDigits(v), count: c})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].value < items[j].value
	})
	var groups [][]item
	for _, it := range items {
		placed := false
		for gi, g := range groups {
			for _, other := range g {
				if other.digits == it.digits && (other.key == it.key || editDistance(other.key, it.key) <= nameTolerance(other.key, it.key)) {
					groups[gi] = append(groups[gi], it)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			groups = append(groups, []item{it})
		}
	}
	resolve := map[string]string{}
	clusters := make([]nameCluster, 0, len(groups))
	for _, g := range groups {
		c := nameCluster{Canonical: g[0].value}
		for _, m := range g {
			resolve[m.value] = c.Canonical
			c.Variants = append(c.Variants, m.value)
		}
		clusters = append(clusters, c)
	}
	return resolve, clusters
}

// ── Бүтээн байгуулалтын төрөл → чөлөөлөлтийн ангилал (НЭРЭЭР) ───────────────

type categoryName struct {
	General string `json:"general"`
	Sub     string `json:"sub,omitempty"`
}

const (
	categoryRoad  = "Авто зам, замын байгууламж"
	categoryGer   = "Гэр хорооллын дахин төлөвлөлт"
	categoryGreen = "Ногоон байгууламж, нийтийн эзэмшил"
)

// fallbackCategory — таарахгүй бол «Бусад» (импорт зогсохгүй).
var fallbackCategory = categoryName{General: categoryGer, Sub: "Бусад"}

// typeCategories — canonName-ээр цэвэрлэсэн нэрийн ЭХЛЭЛЭЭР тулгана
// («гэрхорооллыгоронсууцжуулахтөсөлхөтөлбөр» → «гэрхорооллыгоронсууцжуулах»).
// Урт тулгалтыг эхэлж шалгана.
var typeCategories = []struct {
	match    []string
	category categoryName
}{
	{[]string{"автозам"}, categoryName{General: categoryRoad}},
	{[]string{"дэдтөв", "сэлбэдэдтөв", "баянхошуудэдтөв"}, categoryName{General: categoryGer}},
	{[]string{"хэсэгчилсэнтөлөвлөгөө", "дахинтөлөвлөлт", "гэрхорооллыноронсууцжуулах", "гэрхорооллыгоронсууцжуулах"},
		categoryName{General: categoryGer}},
	{[]string{"сургуульцэцэрлэг", "цэцэрлэг", "сургууль"}, categoryName{General: categoryGer, Sub: "Сургууль, цэцэрлэгийн талбай"}},
	{[]string{"инженерийншугамсүлжээ", "инженжрийншугамсүлжээ", "инженершугамсүлжээ", "шугамсүлжээ", "дулаанышугам", "дулааныстанц"},
		categoryName{General: categoryGer, Sub: "Инженерийн дэд бүтэц"}},
	{[]string{"гаднатохижилт"}, categoryName{General: categoryGreen}},
	{[]string{"барилгабайгууламж", "барилга"}, categoryName{General: categoryGer, Sub: "Барилга, байгууламж"}},
	{[]string{"үерхалиатошин"}, categoryName{General: categoryGer, Sub: "Үер, халиа тошин"}},
	{[]string{"бусад"}, categoryName{General: categoryGer, Sub: "Бусад"}},
}

func categoryForConstructionType(label string) (categoryName, bool) {
	key := canonName(label)
	if key == "" {
		return fallbackCategory, false
	}
	best, bestLen := categoryName{}, 0
	for _, e := range typeCategories {
		for _, m := range e.match {
			if (key == m || strings.HasPrefix(key, m)) && len(m) > bestLen {
				best, bestLen = e.category, len(m)
			}
		}
	}
	if bestLen == 0 {
		return fallbackCategory, false
	}
	return best, true
}

// ── Давхардсан дугаар ───────────────────────────────────────────────────────

// parcelIDLimit — parcel.parcel_id varchar(12), глобал unique.
const parcelIDLimit = 12

// duplicateParcelID — нэг дугаарын 2 дахь ба дараагийн бүртгэл: D<n> + дугаар.
// Багтахгүй бол тогтвортой ERR… дугаар (дүн алдагдахгүй).
func duplicateParcelID(parcelID string, occurrence int, rowKey string) string {
	candidate := "D" + strconv.Itoa(occurrence) + parcelID
	if len(candidate) <= parcelIDLimit {
		return candidate
	}
	return syntheticParcelID(rowKey)
}

// valuationRowKey — мөрийн тогтвортой түлхүүр (CLI-ийн rowKey-тэй ижил).
func valuationRowKey(r *ReportRow) string {
	return r.SourceFile + "|" + r.SourceSheet + "|" + strconv.Itoa(r.ExcelRow)
}
