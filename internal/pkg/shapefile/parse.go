// Package shapefile implements a minimal pure-Go ESRI Shapefile reader.
// ЗӨВХӨН .shp файлыг, зөвхөн Polygon (5 / PolygonZ 15 / PolygonM 25) төрлөөр
// уншина.
//
// Уншихаас гадна оруулсан файл нь appdb-ийн СТАНДАРТАД (geometry(Polygon,4326))
// тохирч байгааг шалгана — Errors-ыг үз: файлын бүтэц, геометрийн төрөл,
// полигоны тоо, рингийн бүтэц, проекц (WGS84 градус), координатын хүрээ.
// Стандартаас зөрсөн файлыг WKT болгож БУЦААХГҮЙ — тодорхой алдаа буцаана.
package shapefile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// appdb-ийн бүх геометр багана нь geometry(...,4326) — өөрөөр хэлбэл WGS84
// градусаар (уртраг/өргөрөг) хадгалагдана. Оруулсан файл ӨӨР проекцтой
// (UTM, Гаусс-Крюгер, Mongolia 2000 гэх мэт метр) бол координат нь градус
// биш тул шалгаж хаана.
var (
	// ErrNotShapefile — .shp файлын бүтэц/тэмдэглэгээ таарахгүй (магадгүй
	// огт shapefile биш, эсвэл эвдэрсэн).
	ErrNotShapefile = errors.New("файл нь ESRI Shapefile (.shp) биш эсвэл эвдэрсэн")
	// ErrShapeType — Polygon биш геометр (шугам, цэг гэх мэт).
	ErrShapeType = errors.New("зөвхөн Polygon төрөл дэмжигдэнэ")
	// ErrMultiPolygon — файлд салангид (нэг нь нөгөөдөө агуулагдаагүй) хоёр ба
	// түүнээс дээш полигон байна. appdb-ийн багана нь geometry(Polygon,4326)
	// тул MultiPolygon хадгалагдахгүй.
	ErrMultiPolygon = errors.New("MultiPolygon (салангид олон полигон) дэмжигдэхгүй — geometry(Polygon,4326) стандартад нэг полигон байх ёстой")
	// ErrRingNotClosed — полигоны ринг хаагдаагүй (эхний ба сүүлийн цэг
	// таарахгүй) эсвэл цэг хэтэрхий бага.
	ErrRingNotClosed = errors.New("полигоны ринг хаагдаагүй эсвэл цэг дутуу")
	// ErrProjection — проекц нь WGS84 (EPSG:4326) биш: координат градусын
	// хүрээнд байхгүй (метр проекц) эсвэл тэнхлэг сольсон.
	ErrProjection = errors.New("проекц нь WGS84 (EPSG:4326) биш — градусаар (уртраг/өргөрөг) байх шаардлагатай")
	// ErrOutsideCountry — координат нь WGS84 градус боловч Монгол Улсын
	// хүрээнд байхгүй.
	ErrOutsideCountry = errors.New("координат Монгол Улсын хүрээнд байхгүй")
	// ErrMultipleRecords — файлд нэгээс олон полигон бичлэг байна. appdb нь
	// нэг Polygon хадгалдаг тул аль нь гэдэг нь тодорхойгүй.
	ErrMultipleRecords = errors.New("файлд нэгээс олон полигон байна — нэг полигон агуулсан файл оруулна уу")
)

// Монгол Улсын хүрээ (WGS84 градус), хилийн нарийвчлалын нөөцтэй.
const (
	mnLonMin, mnLonMax = 87.0, 120.5
	mnLatMin, mnLatMax = 41.0, 52.5
)

// isPolygonType — Polygon (5), PolygonZ (15), PolygonM (25). M/Z хувилбарын
// бичлэгийн эхлэл нь Polygon-тай ижил тул адилхан уншигдана.
func isPolygonType(t int32) bool {
	return t == 5 || t == 15 || t == 25
}

// shapeTypeName — татгалзсан үед ЯМАР төрөл ирснийг хэрэглэгчид хэлэхэд.
func shapeTypeName(t int32) string {
	switch t {
	case 0:
		return "Null"
	case 1, 11, 21:
		return "Point (цэг)"
	case 3, 13, 23:
		return "PolyLine (шугам)"
	case 8, 18, 28:
		return "MultiPoint (олон цэг)"
	case 31:
		return "MultiPatch"
	default:
		return fmt.Sprintf("төрөл %d", t)
	}
}

// Parse reads a raw .shp file and returns the WKT of the polygon.
//
// ЗӨВХӨН .shp файлыг уншина (.prj/.dbf/.shx хүлээж авахгүй). Тиймээс проекц
// нь тунхаглалаар БИШ, координатын хүрээгээр шалгагдана (checkCoordinates).
func Parse(data []byte) (string, error) {
	return parsePolygon(data)
}

// parsePolygon parses the polygon record and returns its WKT.
func parsePolygon(data []byte) (string, error) {
	const fileHeaderSize = 100
	const recordHeaderSize = 8
	const maxParts = 4096
	const maxPoints = 1_000_000

	if len(data) < fileHeaderSize+recordHeaderSize+4 {
		return "", fmt.Errorf("%w: хэт жижиг (%d байт)", ErrNotShapefile, len(data))
	}

	// Файлын толгойн ТЭМДЭГЛЭГЭЭ: file code 9994 (big-endian), version 1000.
	// Өмнө нь шалгагдаагүй тул ямар ч файл "shapefile" болж уншигдаж байв.
	if code := binary.BigEndian.Uint32(data[0:]); code != 9994 {
		return "", fmt.Errorf("%w: file code %d (9994 байх ёстой)", ErrNotShapefile, code)
	}
	if ver := binary.LittleEndian.Uint32(data[28:]); ver != 1000 {
		return "", fmt.Errorf("%w: version %d (1000 байх ёстой)", ErrNotShapefile, ver)
	}
	// Толгой дээрх геометрийн төрөл (бичлэг дээрхтэй адил байх ёстой).
	if ht := int32(binary.LittleEndian.Uint32(data[32:])); !isPolygonType(ht) {
		return "", fmt.Errorf("%w: файл нь %s төрөлтэй", ErrShapeType, shapeTypeName(ht))
	}
	// Толгойд заасан урт (16 битийн үгээр) нь файлын биттэй таарах ёстой.
	if fileWords := int(binary.BigEndian.Uint32(data[24:])); fileWords*2 != len(data) {
		return "", fmt.Errorf("%w: толгойд заасан урт (%d байт) файлын урттай (%d) таарахгүй",
			ErrNotShapefile, fileWords*2, len(data))
	}

	// Нэгээс олон полигон бичлэгтэй эсэхийг шалгана: appdb нь НЭГ Polygon
	// хадгалдаг тул үлдсэнийг чимээгүй хаях нь өгөгдөл алдагдуулна.
	if err := ensureSingleRecord(data); err != nil {
		return "", err
	}

	// first record content starts after file header + record header
	pos := fileHeaderSize + recordHeaderSize

	shapeType := int32(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	if !isPolygonType(shapeType) {
		return "", fmt.Errorf("%w: бичлэг нь %s төрөлтэй", ErrShapeType, shapeTypeName(shapeType))
	}

	// bounding box: Xmin, Ymin, Xmax, Ymax (4 × float64)
	if len(data[pos:]) < 32+4+4 {
		return "", fmt.Errorf("%w: polygon толгой хэт жижиг", ErrNotShapefile)
	}
	pos += 32

	numParts := int(int32(binary.LittleEndian.Uint32(data[pos:])))
	pos += 4
	numPoints := int(int32(binary.LittleEndian.Uint32(data[pos:])))
	pos += 4
	if numParts < 1 || numParts > maxParts {
		return "", fmt.Errorf("%w: parts count %d", ErrNotShapefile, numParts)
	}
	if numPoints < 1 || numPoints > maxPoints {
		return "", fmt.Errorf("%w: points count %d", ErrNotShapefile, numPoints)
	}
	requiredLen := pos + numParts*4 + numPoints*16
	if requiredLen < pos || requiredLen > len(data) {
		return "", fmt.Errorf("%w: бичлэг тасарсан", ErrNotShapefile)
	}

	parts := make([]int, numParts)
	for i := range parts {
		parts[i] = int(int32(binary.LittleEndian.Uint32(data[pos:])))
		if parts[i] < 0 || parts[i] >= numPoints {
			return "", fmt.Errorf("%w: part offset %d", ErrNotShapefile, parts[i])
		}
		if i > 0 && parts[i] <= parts[i-1] {
			return "", fmt.Errorf("%w: part offset-ууд өсөх ёстой", ErrNotShapefile)
		}
		pos += 4
	}

	pts := make([][2]float64, numPoints)
	for i := range pts {
		x := math.Float64frombits(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8
		y := math.Float64frombits(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8
		pts[i] = [2]float64{x, y}
	}

	if err := checkCoordinates(pts); err != nil {
		return "", err
	}

	rings := splitRings(parts, pts, numParts, numPoints)
	shell, err := checkRings(rings)
	if err != nil {
		return "", err
	}

	return toWKT(rings, shell), nil
}

// splitRings — part-уудыг ринг болгон хуваана.
func splitRings(parts []int, pts [][2]float64, numParts, numPoints int) [][][2]float64 {
	rings := make([][][2]float64, 0, numParts)
	for p := 0; p < numParts; p++ {
		end := numPoints
		if p+1 < numParts {
			end = parts[p+1]
		}
		rings = append(rings, pts[parts[p]:end])
	}
	return rings
}

// checkRings — рингүүд стандартад тохирч байгааг шалгаж, ГАДНА рингийн
// индексийг буцаана.
//
// Shapefile-д MultiPolygon гэсэн ТУСДАА төрөл байдаггүй: салангид полигонууд
// ч, цоорхой (hole) ч адилхан "олон part" болж бичигддэг. Тиймээс part-уудын
// АГУУЛАГДАХ байдлаар (нэг нь нөгөө дотор орсон эсэх) хоёрыг ялгана:
//
//   - Бүх нэмэлт ринг гадна ринг дотор → цоорхой бүхий НЭГ полигон (зөв).
//   - Хоёр ба түүнээс дээш ринг хаана ч агуулагдаагүй → MultiPolygon
//     (geometry(Polygon,4326) багананд хадгалагдахгүй) → ErrMultiPolygon.
//
// Агуулагдахыг ray casting-аар шалгах нь рингийн эргэлтийн чиглэлээс (CW/CCW)
// ХАМААРАХГҮЙ — бодит файлууд стандарт чиглэлийг байнга зөрчдөг тул чиглэлээр
// шалгах нь найдваргүй.
func checkRings(rings [][][2]float64) (int, error) {
	for _, r := range rings {
		if len(r) < 4 || r[0] != r[len(r)-1] {
			return 0, fmt.Errorf("%w (%d цэг)", ErrRingNotClosed, len(r))
		}
	}

	shell := -1
	outer := 0
	for i, r := range rings {
		contained := false
		for j, other := range rings {
			if i != j && pointInRing(r[0], other) {
				contained = true
				break
			}
		}
		if !contained {
			outer++
			if shell < 0 {
				shell = i
			}
		}
	}
	if outer > 1 {
		return 0, fmt.Errorf("%w (%d салангид полигон)", ErrMultiPolygon, outer)
	}
	if shell < 0 {
		// Бүх ринг бие бие дотор — тодорхойлох боломжгүй бүтэц.
		return 0, fmt.Errorf("%w: рингийн бүтэц тодорхойгүй", ErrNotShapefile)
	}
	return shell, nil
}

// pointInRing — ray casting: цэг рингийн дотор эсэх.
func pointInRing(p [2]float64, ring [][2]float64) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		if (yi > p[1]) != (yj > p[1]) &&
			p[0] < (xj-xi)*(p[1]-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

// ensureSingleRecord — файлд полигон бичлэг ЯГ нэг байгааг шалгана.
// Бичлэгийн толгой: дугаар (4 байт BE) + агуулгын урт (4 байт BE, 16 битийн
// үгээр). Null shape (type 0) бичлэгийг тоохгүй.
func ensureSingleRecord(data []byte) error {
	const fileHeaderSize = 100
	count := 0
	for pos := fileHeaderSize; pos+8 <= len(data); {
		contentWords := int(binary.BigEndian.Uint32(data[pos+4:]))
		if contentWords <= 0 {
			break // эвдэрсэн/хоосон бичлэг — цаашид уншихгүй
		}
		contentLen := contentWords * 2
		body := pos + 8
		if body+4 > len(data) || body+contentLen > len(data) {
			break // тасарсан сүүл — үндсэн бичлэгийн шалгалт доор хийгдэнэ
		}
		if int32(binary.LittleEndian.Uint32(data[body:])) != 0 {
			count++
			if count > 1 {
				return ErrMultipleRecords
			}
		}
		pos = body + contentLen
	}
	return nil
}

// checkCoordinates — координат нь WGS84 градус бөгөөд Монгол Улсын хүрээнд
// байгааг шалгана.
//
// Метр проекцтой (UTM, Гаусс-Крюгер, Mongolia 2000) файлын координат нь
// зуун мянгаараа тоологддог тул градусын хүрээнд ОРОХГҮЙ — ингэж проекц
// зөрснийг .prj байхгүй ч илрүүлнэ. Тэнхлэг сольсон (өргөрөг, уртраг)
// тохиолдол ч өргөрөг > 90 болж мөн энд баригдана.
func checkCoordinates(pts [][2]float64) error {
	for _, p := range pts {
		lon, lat := p[0], p[1]
		if math.IsNaN(lon) || math.IsNaN(lat) || math.IsInf(lon, 0) || math.IsInf(lat, 0) {
			return fmt.Errorf("%w: координат тоо биш (NaN/Inf)", ErrNotShapefile)
		}
		if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
			return fmt.Errorf("%w (координат: %.3f %.3f — метр проекц эсвэл тэнхлэг сольсон байж магадгүй)",
				ErrProjection, lon, lat)
		}
	}
	for _, p := range pts {
		lon, lat := p[0], p[1]
		if lon < mnLonMin || lon > mnLonMax || lat < mnLatMin || lat > mnLatMax {
			return fmt.Errorf("%w (координат: %.3f %.3f)", ErrOutsideCountry, lon, lat)
		}
	}
	return nil
}

// toWKT — POLYGON WKT. WKT-д ГАДНА ринг ХАМГИЙН ЭХЭНД байх ёстой тул
// shell-ийг эхлүүлж, дараа нь цоорхойнуудыг бичнэ (файлд гадна ринг эхний
// part биш байж болно).
func toWKT(rings [][][2]float64, shell int) string {
	order := make([]int, 0, len(rings))
	order = append(order, shell)
	for i := range rings {
		if i != shell {
			order = append(order, i)
		}
	}

	var b strings.Builder
	b.WriteString("POLYGON(")
	for n, idx := range order {
		if n > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		for i, pt := range rings[idx] {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "%.6f %.6f", pt[0], pt[1])
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	return b.String()
}
