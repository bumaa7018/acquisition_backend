package legacyimport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// ── Shapefile (.shp + .dbf + .prj) ──────────────────────────────────────────
//
// internal/pkg/shapefile нь ЗӨВХӨН нэг полигонтой .shp уншдаг (чөлөөлөлтийн
// хил солих дэлгэц). Импортод олон бичлэг, .dbf атрибут (дугаар, төлөвийн
// код), UTM проекц хэрэгтэй тул энд тусад нь уншина. Проекцийн хөрвүүлэлтийг
// PostGIS (ST_Transform) хийнэ — энд зөвхөн SRID-ийг тодорхойлно.

// feature — shapefile-ийн нэг бичлэг.
type feature struct {
	index int               // 1-ээс
	wkt   string            // ЭХ проекцоор (srid); полигон биш/хоосон бол ""
	props map[string]string // .dbf атрибут (нэрээр)
}

// shapefileSet — нэг .shp-ийн хамтрагч файлууд (өргөтгөлөөр).
type shapefileSet struct {
	base  string
	parts map[string][]byte
}

var requiredShpParts = []string{".shp", ".dbf", ".shx"}

// groupShapefiles — файлуудыг нэрээр (өргөтгөлгүй) бүлэглэнэ. .shp-гүй бүлэг
// (ж нь .qmd) алгасагдана.
func groupShapefiles(files []UploadedFile) []shapefileSet {
	groups := map[string]map[string][]byte{}
	var order []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Name))
		base := strings.TrimSuffix(filepath.Base(f.Name), filepath.Ext(f.Name))
		if _, ok := groups[base]; !ok {
			groups[base] = map[string][]byte{}
			order = append(order, base)
		}
		groups[base][ext] = f.Data
	}
	var sets []shapefileSet
	for _, base := range order {
		if _, ok := groups[base][".shp"]; ok {
			sets = append(sets, shapefileSet{base: base, parts: groups[base]})
		}
	}
	return sets
}

// shapefileIssues — дутуу хэсэг (алдаа) ба .prj байхгүй (анхааруулга).
func (s shapefileSet) issues() []Issue {
	var out []Issue
	var missing []string
	for _, ext := range requiredShpParts {
		if _, ok := s.parts[ext]; !ok {
			missing = append(missing, ext)
		}
	}
	if len(missing) > 0 {
		out = append(out, errIssue("missing_parts",
			fmt.Sprintf("%s: %s файл дутуу — shapefile-ийн бүх хэсгийг хамт сонгоно уу", s.base, strings.Join(missing, ", "))))
	}
	if _, ok := s.parts[".prj"]; !ok {
		out = append(out, warnIssue("missing_prj",
			fmt.Sprintf("%s: .prj файл байхгүй — координатыг WGS84 (EPSG:4326) гэж үзнэ", s.base)))
	}
	return out
}

// readShapefileSet — .shp геометр + .dbf атрибут + .prj-ийн SRID.
func readShapefileSet(s shapefileSet) ([]feature, int, error) {
	srid, err := sridFromPrj(string(s.parts[".prj"]))
	if err != nil {
		return nil, 0, err
	}
	geoms, err := readShp(s.parts[".shp"])
	if err != nil {
		return nil, 0, err
	}
	records, err := readDbf(s.parts[".dbf"], strings.TrimSpace(string(s.parts[".cpg"])))
	if err != nil {
		return nil, 0, err
	}
	features := make([]feature, len(geoms))
	for i, wkt := range geoms {
		props := map[string]string{}
		if i < len(records) {
			props = records[i]
		}
		features[i] = feature{index: i + 1, wkt: wkt, props: props}
	}
	return features, srid, nil
}

var (
	reUTMZone       = regexp.MustCompile(`(?i)UTM[_ ]?zone[_ ]?(\d{1,2})\s*([NS])`)
	reEPSGAuthority = regexp.MustCompile(`AUTHORITY\["EPSG",\s*"?(\d+)"?\]\s*\]\s*$`)
)

// sridFromPrj — .prj (ESRI WKT) → EPSG SRID. Хоосон бол 4326 (WGS84).
func sridFromPrj(prj string) (int, error) {
	prj = strings.TrimSpace(prj)
	if prj == "" {
		return 4326, nil
	}
	if m := reEPSGAuthority.FindStringSubmatch(prj); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, nil
	}
	upper := strings.ToUpper(prj)
	wgs84 := strings.Contains(upper, "WGS_1984") || strings.Contains(upper, "WGS 84") || strings.Contains(upper, "WGS84")
	if !strings.HasPrefix(upper, "PROJCS") {
		if strings.HasPrefix(upper, "GEOGCS") && wgs84 {
			return 4326, nil
		}
		return 0, fmt.Errorf("проекц танигдсангүй (%s) — файлыг EPSG:4326 руу хөрвүүлж оруулна уу", prjName(prj))
	}
	if m := reUTMZone.FindStringSubmatch(prj); m != nil && wgs84 {
		zone, _ := strconv.Atoi(m[1])
		if zone >= 1 && zone <= 60 {
			if strings.EqualFold(m[2], "N") {
				return 32600 + zone, nil
			}
			return 32700 + zone, nil
		}
	}
	return 0, fmt.Errorf("проекц танигдсангүй (%s) — WGS84 эсвэл WGS84/UTM биш бол EPSG:4326 руу хөрвүүлж оруулна уу", prjName(prj))
}

var rePrjName = regexp.MustCompile(`^[A-Z]+\["([^"]*)"`)

func prjName(prj string) string {
	if m := rePrjName.FindStringSubmatch(prj); m != nil {
		return m[1]
	}
	return "нэргүй"
}

// readShp — полигон бичлэг бүрийг WKT (POLYGON / MULTIPOLYGON) болгоно.
func readShp(data []byte) ([]string, error) {
	if len(data) < 100 || binary.BigEndian.Uint32(data[0:4]) != 9994 {
		return nil, errors.New(".shp файл эвдэрсэн эсвэл shapefile биш")
	}
	var out []string
	for off := 100; off+8 <= len(data); {
		contentLen := int(binary.BigEndian.Uint32(data[off+4:off+8])) * 2
		start, end := off+8, off+8+contentLen
		if end > len(data) || contentLen < 4 {
			return nil, errors.New(".shp файлын бичлэг таслагдсан")
		}
		wkt, err := polygonWKT(data[start:end])
		if err != nil {
			return nil, fmt.Errorf("%d-р бичлэг: %w", len(out)+1, err)
		}
		out = append(out, wkt)
		off = end
	}
	return out, nil
}

type point struct{ x, y float64 }

func polygonWKT(rec []byte) (string, error) {
	shapeType := binary.LittleEndian.Uint32(rec[0:4])
	switch shapeType {
	case 0:
		return "", nil // Null shape
	case 5, 15, 25:
	default:
		return "", fmt.Errorf("геометрийн төрөл полигон биш (shape type %d)", shapeType)
	}
	if len(rec) < 44 {
		return "", errors.New("полигоны бичлэг дутуу")
	}
	numParts := int(binary.LittleEndian.Uint32(rec[36:40]))
	numPoints := int(binary.LittleEndian.Uint32(rec[40:44]))
	partsOff := 44
	pointsOff := partsOff + 4*numParts
	if numParts <= 0 || numPoints <= 0 || pointsOff+16*numPoints > len(rec) {
		return "", errors.New("полигоны бичлэг дутуу")
	}
	starts := make([]int, numParts)
	for i := range starts {
		starts[i] = int(binary.LittleEndian.Uint32(rec[partsOff+4*i:]))
	}
	var rings [][]point
	for i, s := range starts {
		e := numPoints
		if i+1 < numParts {
			e = starts[i+1]
		}
		if s < 0 || e > numPoints || e-s < 4 {
			return "", errors.New("полигоны ринг хаагдаагүй эсвэл цэг дутуу")
		}
		ring := make([]point, 0, e-s)
		for p := s; p < e; p++ {
			o := pointsOff + 16*p
			ring = append(ring, point{
				x: math.Float64frombits(binary.LittleEndian.Uint64(rec[o:])),
				y: math.Float64frombits(binary.LittleEndian.Uint64(rec[o+8:])),
			})
		}
		rings = append(rings, ring)
	}
	// Shapefile-ийн стандарт: гадна ринг ЦАГИЙН зүүний дагуу, нүх эсрэг.
	var polygons [][][]point
	for _, ring := range rings {
		if clockwise(ring) || len(polygons) == 0 {
			polygons = append(polygons, [][]point{ring})
			continue
		}
		last := len(polygons) - 1
		polygons[last] = append(polygons[last], ring)
	}
	var b strings.Builder
	if len(polygons) == 1 {
		b.WriteString("POLYGON")
		writePolygon(&b, polygons[0])
		return b.String(), nil
	}
	b.WriteString("MULTIPOLYGON(")
	for i, poly := range polygons {
		if i > 0 {
			b.WriteString(",")
		}
		writePolygon(&b, poly)
	}
	b.WriteString(")")
	return b.String(), nil
}

func clockwise(ring []point) bool {
	sum := 0.0
	for i := 0; i+1 < len(ring); i++ {
		sum += (ring[i+1].x - ring[i].x) * (ring[i+1].y + ring[i].y)
	}
	return sum > 0
}

func writePolygon(b *strings.Builder, rings [][]point) {
	b.WriteString("(")
	for i, ring := range rings {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		for j, p := range ring {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString(strconv.FormatFloat(p.x, 'f', -1, 64))
			b.WriteString(" ")
			b.WriteString(strconv.FormatFloat(p.y, 'f', -1, 64))
		}
		b.WriteString(")")
	}
	b.WriteString(")")
}

// readDbf — dBase III атрибутууд. .cpg-гүй эсвэл UTF-8 бол UTF-8, 1251 бол
// кирилл Windows-1251 гэж уншина.
func readDbf(data []byte, cpg string) ([]map[string]string, error) {
	if len(data) < 32 {
		return nil, errors.New(".dbf файл эвдэрсэн")
	}
	numRecords := int(binary.LittleEndian.Uint32(data[4:8]))
	headerLen := int(binary.LittleEndian.Uint16(data[8:10]))
	recordLen := int(binary.LittleEndian.Uint16(data[10:12]))
	type field struct {
		name   string
		kind   byte
		length int
	}
	var fields []field
	for off := 32; off+32 <= headerLen && off < len(data) && data[off] != 0x0D; off += 32 {
		name := string(bytes.TrimRight(data[off:off+11], "\x00 "))
		fields = append(fields, field{name: name, kind: data[off+11], length: int(data[off+16])})
	}
	decode := func(b []byte) string {
		if strings.Contains(cpg, "1251") {
			if s, err := charmap.Windows1251.NewDecoder().Bytes(b); err == nil {
				return string(s)
			}
		}
		if !utf8.Valid(b) {
			if s, err := charmap.Windows1251.NewDecoder().Bytes(b); err == nil {
				return string(s)
			}
		}
		return string(b)
	}
	records := make([]map[string]string, 0, numRecords)
	for r := 0; r < numRecords; r++ {
		off := headerLen + r*recordLen
		if off+recordLen > len(data) {
			break
		}
		rec := data[off : off+recordLen]
		pos := 1 // эхний байт — устгасан эсэх
		props := make(map[string]string, len(fields))
		for _, f := range fields {
			if pos+f.length > len(rec) {
				break
			}
			raw := strings.TrimSpace(decode(bytes.TrimRight(rec[pos:pos+f.length], "\x00")))
			pos += f.length
			if raw == "" || (f.kind == 'N' || f.kind == 'F') && strings.Trim(raw, "*") == "" {
				continue
			}
			props[f.name] = raw
		}
		records = append(records, props)
	}
	return records, nil
}

// attr — атрибутыг олон нэрээр (том/жижиг үсэг ялгахгүй) хайна. dbf-ийн нэр
// 10 тэмдэгтээр тасардаг тул хэд хэдэн хувилбараар өгнө.
func attr(props map[string]string, names ...string) string {
	for _, n := range names {
		if v, ok := props[n]; ok && v != "" {
			return v
		}
	}
	for _, n := range names {
		for k, v := range props {
			if strings.EqualFold(k, n) && v != "" {
				return v
			}
		}
	}
	return ""
}

func hasAttr(features []feature, names ...string) bool {
	for _, f := range features {
		if attr(f.props, names...) != "" {
			return true
		}
	}
	return false
}

func attrNames(features []feature) string {
	if len(features) == 0 || len(features[0].props) == 0 {
		return "(атрибутгүй)"
	}
	var names []string
	for k := range features[0].props {
		names = append(names, k)
	}
	return strings.Join(names, ", ")
}

// Атрибутын нэрс (CLI-тэй ижил).
var (
	planParcelFields = []string{"parcel_id", "plan_parcel_id", "plan_parce", "parcelid", "pl_parcel"}
	parcelIDFields   = []string{"parcel_id", "parcelid", "parcel"}
	statusCodeFields = []string{"code", "status_code", "status"}
)

// intAttr — "3" / "3.000" → 3.
func intAttr(props map[string]string, names ...string) *int {
	n, ok := parseNumber(attr(props, names...))
	if !ok {
		return nil
	}
	v := int(n)
	return &v
}
