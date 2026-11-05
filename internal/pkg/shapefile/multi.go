package shapefile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

var (
	// ErrNoFeatures — файлд полигон бичлэг огт алга (бүгд Null shape эсвэл хоосон).
	ErrNoFeatures = errors.New("файлд полигон бичлэг алга")
	// ErrPRJNotWGS84 — дагалдах .prj файл нь WGS84 газарзүйн (градус) координат
	// систем биш (ж: UTM, Гаусс-Крюгер, Mongolia 2000 зэрэг метр проекц).
	ErrPRJNotWGS84 = errors.New("координат систем (.prj) нь WGS84 (EPSG:4326) биш — градусаар (уртраг/өргөрөг) байх шаардлагатай")
)

// ParseAll — .shp файлын БҮХ полигон бичлэгийг уншиж, бичлэг тус бүрийг
// MULTIPOLYGON WKT болгон буцаана (буцах дараалал = файлын бичлэгийн дараалал).
//
// Parse-аас ялгаа: олон бичлэг, мөн нэг бичлэг дотор салангид олон полигон
// (MultiPolygon) зөвшөөрнө — «Буулгалтын мэдээлэл» нь нэг файлд олон газар
// агуулдаг. Файлын бүтэц, геометрийн төрөл, рингийн бүтэц, проекц (WGS84
// градус) ба Монгол Улсын хүрээний шалгалт Parse-тай ИЖИЛ.
func ParseAll(data []byte) ([]string, error) {
	const fileHeaderSize = 100
	if len(data) < fileHeaderSize+8+4 {
		return nil, fmt.Errorf("%w: хэт жижиг (%d байт)", ErrNotShapefile, len(data))
	}
	if code := binary.BigEndian.Uint32(data[0:]); code != 9994 {
		return nil, fmt.Errorf("%w: file code %d (9994 байх ёстой)", ErrNotShapefile, code)
	}
	if ver := binary.LittleEndian.Uint32(data[28:]); ver != 1000 {
		return nil, fmt.Errorf("%w: version %d (1000 байх ёстой)", ErrNotShapefile, ver)
	}
	if ht := int32(binary.LittleEndian.Uint32(data[32:])); !isPolygonType(ht) {
		return nil, fmt.Errorf("%w: файл нь %s төрөлтэй", ErrShapeType, shapeTypeName(ht))
	}
	if fileWords := int(binary.BigEndian.Uint32(data[24:])); fileWords*2 != len(data) {
		return nil, fmt.Errorf("%w: толгойд заасан урт (%d байт) файлын урттай (%d) таарахгүй",
			ErrNotShapefile, fileWords*2, len(data))
	}

	var out []string
	for pos, n := fileHeaderSize, 0; pos+8 <= len(data); n++ {
		contentLen := int(binary.BigEndian.Uint32(data[pos+4:])) * 2
		body := pos + 8
		if contentLen <= 0 || body+contentLen > len(data) {
			return nil, fmt.Errorf("%w: %d-р бичлэг тасарсан", ErrNotShapefile, n+1)
		}
		rec := data[body : body+contentLen]
		pos = body + contentLen

		shapeType := int32(binary.LittleEndian.Uint32(rec[0:]))
		if shapeType == 0 {
			continue // Null shape — алгасна
		}
		if !isPolygonType(shapeType) {
			return nil, fmt.Errorf("%w: %d-р бичлэг %s төрөлтэй", ErrShapeType, n+1, shapeTypeName(shapeType))
		}
		rings, err := readPolygonRecord(rec)
		if err != nil {
			return nil, fmt.Errorf("%d-р бичлэг: %w", n+1, err)
		}
		wkt, err := multiPolygonWKT(rings)
		if err != nil {
			return nil, fmt.Errorf("%d-р бичлэг: %w", n+1, err)
		}
		out = append(out, wkt)
	}
	if len(out) == 0 {
		return nil, ErrNoFeatures
	}
	return out, nil
}

// CheckPRJ — дагалдах .prj (WKT) нь WGS84 газарзүйн координат систем эсэх.
// Проекцтой (PROJCS — метр) эсвэл WGS84 биш датумтай бол ErrPRJNotWGS84.
func CheckPRJ(prj string) error {
	s := strings.ToUpper(strings.TrimSpace(prj))
	if s == "" {
		return nil // .prj хоосон — координатын хүрээгээр шалгагдана
	}
	if strings.HasPrefix(s, "PROJCS") || strings.Contains(s, "PROJECTION[") {
		return ErrPRJNotWGS84
	}
	if !strings.HasPrefix(s, "GEOGCS") || (!strings.Contains(s, "WGS_1984") && !strings.Contains(s, "WGS 84") && !strings.Contains(s, "WGS84")) {
		return ErrPRJNotWGS84
	}
	return nil
}

// readPolygonRecord — нэг Polygon бичлэгийн (record content) рингүүд.
// Координатыг (WGS84 градус, Монгол Улсын хүрээ) шалгана.
func readPolygonRecord(rec []byte) ([][][2]float64, error) {
	const maxParts = 4096
	const maxPoints = 1_000_000
	pos := 4 + 32 // shape type + bbox
	if len(rec) < pos+8 {
		return nil, fmt.Errorf("%w: polygon толгой хэт жижиг", ErrNotShapefile)
	}
	numParts := int(int32(binary.LittleEndian.Uint32(rec[pos:])))
	numPoints := int(int32(binary.LittleEndian.Uint32(rec[pos+4:])))
	pos += 8
	if numParts < 1 || numParts > maxParts {
		return nil, fmt.Errorf("%w: parts count %d", ErrNotShapefile, numParts)
	}
	if numPoints < 1 || numPoints > maxPoints {
		return nil, fmt.Errorf("%w: points count %d", ErrNotShapefile, numPoints)
	}
	if need := pos + numParts*4 + numPoints*16; need > len(rec) {
		return nil, fmt.Errorf("%w: бичлэг тасарсан", ErrNotShapefile)
	}
	parts := make([]int, numParts)
	for i := range parts {
		parts[i] = int(int32(binary.LittleEndian.Uint32(rec[pos:])))
		if parts[i] < 0 || parts[i] >= numPoints || (i > 0 && parts[i] <= parts[i-1]) {
			return nil, fmt.Errorf("%w: part offset %d", ErrNotShapefile, parts[i])
		}
		pos += 4
	}
	pts := make([][2]float64, numPoints)
	for i := range pts {
		x := math.Float64frombits(binary.LittleEndian.Uint64(rec[pos:]))
		y := math.Float64frombits(binary.LittleEndian.Uint64(rec[pos+8:]))
		pos += 16
		pts[i] = [2]float64{x, y}
	}
	if err := checkCoordinates(pts); err != nil {
		return nil, err
	}
	rings := splitRings(parts, pts, numParts, numPoints)
	for _, r := range rings {
		if len(r) < 4 || r[0] != r[len(r)-1] {
			return nil, fmt.Errorf("%w (%d цэг)", ErrRingNotClosed, len(r))
		}
	}
	return rings, nil
}

// multiPolygonWKT — рингүүдийг агуулагдах байдлаар нь гадна ринг (shell) ба
// цоорхой (hole) болгож MULTIPOLYGON WKT бүтээнэ. Цоорхой нь өөрийг нь
// агуулсан ЭХНИЙ гадна рингэд оноогдоно.
func multiPolygonWKT(rings [][][2]float64) (string, error) {
	isHole := make([]bool, len(rings))
	for i, r := range rings {
		for j, other := range rings {
			if i != j && pointInRing(r[0], other) {
				isHole[i] = true
				break
			}
		}
	}
	type poly struct {
		shell int
		holes []int
	}
	var polys []poly
	for i := range rings {
		if !isHole[i] {
			polys = append(polys, poly{shell: i})
		}
	}
	if len(polys) == 0 {
		return "", fmt.Errorf("%w: рингийн бүтэц тодорхойгүй", ErrNotShapefile)
	}
	for i := range rings {
		if !isHole[i] {
			continue
		}
		for k := range polys {
			if pointInRing(rings[i][0], rings[polys[k].shell]) {
				polys[k].holes = append(polys[k].holes, i)
				break
			}
		}
	}

	var b strings.Builder
	writeRing := func(r [][2]float64) {
		b.WriteString("(")
		for i, pt := range r {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "%.6f %.6f", pt[0], pt[1])
		}
		b.WriteString(")")
	}
	b.WriteString("MULTIPOLYGON(")
	for k, p := range polys {
		if k > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		writeRing(rings[p.shell])
		for _, h := range p.holes {
			b.WriteString(",")
			writeRing(rings[h])
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	return b.String(), nil
}
