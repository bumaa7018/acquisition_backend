// Package legacyimport — газар чөлөөлөлтийн ХУУЧИН МЭДЭЭЛЛИЙН импорт.
//
// Чөлөөлөлтийн хилийн shapefile, нэгж талбарын shapefile, үнэлгээний эксэлийг
// хүлээн авч шалгаад ТҮР хүснэгтэд (legacy_import_stage схем) хадгална.
// Хэрэглэгч «Оруулах» дарсны ДАРАА л системийн хүснэгтүүдэд (land_acquisition,
// parcel, compensation …) НЭГ гүйлгээнд бичнэ. ГУС (middleware) руу зөвхөн энэ
// багц дуудна; хүсэлт бүр цуварч, хоорондоо завсарлагатай явна.
//
// Дүрмүүд нь government-legacy-import (CLI) скрипттэй ИЖИЛ.
package legacyimport

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	reSpaces   = regexp.MustCompile(`\s+`)
	reDigits6  = regexp.MustCompile(`\d{6,}`)
	reDateText = regexp.MustCompile(`(\d{4})\s*[.\-/]\s*(\d{1,2})\s*[.\-/]\s*(\d{1,2})`)
)

// normalize — толгой/утгыг тулгахад ашиглах хэвийн хэлбэр.
func normalize(value string) string {
	value = strings.ReplaceAll(value, " ", " ")
	return strings.TrimSpace(reSpaces.ReplaceAllString(value, " "))
}

// headerKey — толгой тулгах түлхүүр: жижиг үсэг, зөвхөн үсэг/тоо.
func headerKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(normalize(value)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseNumber — "284,350,000", "284 350 000", "1 462 568 000₮" гэх мэтийг тоо
// болгоно. Таарахгүй бол ok=false.
func parseNumber(value string) (float64, bool) {
	raw := normalize(value)
	raw = strings.NewReplacer("₮", "", " ", "", ",", "", "'", "", "`", "").Replace(raw)
	if raw == "" || strings.Trim(raw, "-") == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	return n, true
}

// parseAmount — дүнгийн нүд: 0 ба хоосныг ижилхэн "утгагүй" гэж үзнэ.
func parseAmount(value string) float64 {
	n, ok := parseNumber(value)
	if !ok || n <= 0 {
		return 0
	}
	return n
}

// parseOptionalNumber — хоосон бол nil.
func parseOptionalNumber(value string) *float64 {
	if n, ok := parseNumber(value); ok {
		return &n
	}
	return nil
}

// parseDate — текстээс ЭХНИЙ огноог YYYY-MM-DD болгоно ("2025.01.24-А70").
func parseDate(value string) string {
	m := reDateText.FindStringSubmatch(normalize(value))
	if m == nil {
		return ""
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	if year < 1900 || year > 2200 || month < 1 || month > 12 || day < 1 || day > 31 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// parcelKey — нэгж талбарын дугаарыг цэгцэлнэ ("1461501790 |", "№1461502288"
// гэх мэт бохир нүднээс кадастрын дугаарыг сугална). "0" = байхгүй.
func parcelKey(value string) string {
	text := strings.TrimSuffix(normalize(value), ".0")
	if m := reDigits6.FindString(text); m != "" {
		return m
	}
	if text == "0" {
		return ""
	}
	return text
}

const (
	// oldParcelIDMax — parcel.old_parcel_id varchar(14).
	oldParcelIDMax = 14
	// parcelIDMax — parcel.parcel_id varchar(12).
	parcelIDMax = 12
)

// parseParcelIDs — нүдэн дэх дугаар(ууд): богино = parcel_id, урт = old_parcel_id.
func parseParcelIDs(value string) (parcelID, oldParcelID string, err error) {
	seen := map[string]bool{}
	var ids []string
	for _, id := range reDigits6.FindAllString(normalize(value), -1) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	switch {
	case len(ids) == 0:
		return "", "", nil
	case len(ids) == 1:
		if len(ids[0]) > parcelIDMax {
			return "", ids[0], nil
		}
		return ids[0], "", nil
	case len(ids) != 2 || len(ids[0]) == len(ids[1]):
		return "", "", fmt.Errorf("нэгж талбарын дугааруудыг ялгах боломжгүй: %q", normalize(value))
	}
	sort.Slice(ids, func(i, j int) bool { return len(ids[i]) < len(ids[j]) })
	if len(ids[0]) > parcelIDMax {
		return "", ids[1], nil
	}
	return ids[0], ids[1], nil
}

// cleanRegisterNo — "/ХЛ56072067/" → "ХЛ56072067".
func cleanRegisterNo(value string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimLeft(normalize(value), `/\(`), `/\)`))
}

// syntheticParcelID — дугааргүй мөрийн ТОГТВОРТОЙ зохиомол дугаар: ERR + 9 орон.
// (CLI-ийн FNV-1a хэш — ижил геометрт ижил дугаар гарна.)
func syntheticParcelID(seed string) string {
	hash := uint32(2166136261)
	for _, unit := range utf16Units(seed) {
		hash ^= uint32(unit)
		hash *= 16777619
	}
	return fmt.Sprintf("ERR%09d", hash%1_000_000_000)
}

// utf16Units — JS-ийн charCodeAt-тай ижил код нэгжүүд (CLI-тэй ижил хэш).
func utf16Units(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
			continue
		}
		out = append(out, uint16(r))
	}
	return out
}

// fmtInt — мэдээлэлд харагдах бүхэл тоо ("1,234,567").
func fmtInt(n float64) string {
	v := int64(math.Round(n))
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
