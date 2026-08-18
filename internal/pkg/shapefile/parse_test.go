package shapefile_test

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"acquisition_backend/internal/pkg/shapefile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildShp нь заасан цэгүүдээр хялбар .shp файлын байт буцаана.
func buildShp(t *testing.T, pts [][2]float64, shapeType uint32) []byte {
	t.Helper()

	numParts := 1
	numPoints := len(pts)

	contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
	contentWords := uint32(contentLen / 2)
	fileWords := uint32((100 + 8 + contentLen) / 2)

	shpBuf := make([]byte, 100+8+contentLen)

	// File header
	binary.BigEndian.PutUint32(shpBuf[0:], 9994)
	binary.BigEndian.PutUint32(shpBuf[24:], fileWords)
	binary.LittleEndian.PutUint32(shpBuf[28:], 1000)
	binary.LittleEndian.PutUint32(shpBuf[32:], shapeType)

	// Record header
	binary.BigEndian.PutUint32(shpBuf[100:], 1)
	binary.BigEndian.PutUint32(shpBuf[104:], contentWords)

	// Record content
	off := 108
	binary.LittleEndian.PutUint32(shpBuf[off:], shapeType)
	off += 4 + 32
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numParts))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numPoints))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], 0)
	off += 4
	for _, p := range pts {
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[0]))
		off += 8
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[1]))
		off += 8
	}
	return shpBuf
}

var testPts = [][2]float64{
	{106.76, 47.66},
	{106.82, 47.66},
	{106.82, 47.72},
	{106.76, 47.72},
	{106.76, 47.66},
}

func TestParse_Polygon(t *testing.T) {
	wkt, err := shapefile.Parse(buildShp(t, testPts, 5))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(wkt, "POLYGON(("), "expected POLYGON, got: %s", wkt)
	assert.Contains(t, wkt, "106.760000 47.660000")
	assert.Contains(t, wkt, "106.820000 47.720000")
}

func TestParse_PolygonZ(t *testing.T) {
	wkt, err := shapefile.Parse(buildShp(t, testPts, 15))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(wkt, "POLYGON(("))
}

func TestParse_TooShort(t *testing.T) {
	_, err := shapefile.Parse([]byte("short"))
	assert.Error(t, err)
}

func TestParse_TruncatedPolygonRecord(t *testing.T) {
	data := buildShp(t, testPts, 5)
	_, err := shapefile.Parse(data[:len(data)-4])
	assert.Error(t, err)
}

func TestParse_InvalidPartOffset(t *testing.T) {
	data := buildShp(t, testPts, 5)
	// First part offset lives after the record shape type, bbox, numParts, and numPoints.
	binary.LittleEndian.PutUint32(data[108+4+32+4+4:], uint32(len(testPts)))
	_, err := shapefile.Parse(data)
	assert.Error(t, err)
}

func TestParse_WrongShapeType(t *testing.T) {
	_, err := shapefile.Parse(buildShp(t, testPts, 3)) // PolyLine
	assert.ErrorIs(t, err, shapefile.ErrShapeType)
}

func TestParse_MultiRing(t *testing.T) {
	ring1 := [][2]float64{
		{106.70, 47.60}, {106.90, 47.60}, {106.90, 47.80}, {106.70, 47.80}, {106.70, 47.60},
	}
	ring2 := [][2]float64{
		{106.75, 47.65}, {106.80, 47.65}, {106.80, 47.70}, {106.75, 47.70}, {106.75, 47.65},
	}
	allPts := append(ring1, ring2...)

	numParts := 2
	numPoints := len(allPts)
	contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
	shpBuf := make([]byte, 100+8+contentLen)
	binary.BigEndian.PutUint32(shpBuf[0:], 9994)
	binary.BigEndian.PutUint32(shpBuf[24:], uint32((100+8+contentLen)/2))
	binary.LittleEndian.PutUint32(shpBuf[28:], 1000)
	binary.LittleEndian.PutUint32(shpBuf[32:], 5)
	binary.BigEndian.PutUint32(shpBuf[100:], 1)
	binary.BigEndian.PutUint32(shpBuf[104:], uint32(contentLen/2))
	off := 108
	binary.LittleEndian.PutUint32(shpBuf[off:], 5)
	off += 4 + 32
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numParts))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numPoints))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], 0)
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(len(ring1)))
	off += 4
	for _, p := range allPts {
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[0]))
		off += 8
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[1]))
		off += 8
	}

	wkt, err := shapefile.Parse(shpBuf)
	require.NoError(t, err)
	assert.Contains(t, wkt, "),(")
}

// ── Стандартын шалгалт ───────────────────────────────────────

func TestParse_RejectsBadFileCode(t *testing.T) {
	data := buildShp(t, testPts, 5)
	binary.BigEndian.PutUint32(data[0:], 1234) // 9994 байх ёстой
	_, err := shapefile.Parse(data)
	assert.ErrorIs(t, err, shapefile.ErrNotShapefile)
}

func TestParse_RejectsBadVersion(t *testing.T) {
	data := buildShp(t, testPts, 5)
	binary.LittleEndian.PutUint32(data[28:], 999) // 1000 байх ёстой
	_, err := shapefile.Parse(data)
	assert.ErrorIs(t, err, shapefile.ErrNotShapefile)
}

// Толгойд заасан урт файлын урттай таарахгүй бол хаана.
func TestParse_RejectsFileLengthMismatch(t *testing.T) {
	data := buildShp(t, testPts, 5)
	binary.BigEndian.PutUint32(data[24:], 99999)
	_, err := shapefile.Parse(data)
	assert.ErrorIs(t, err, shapefile.ErrNotShapefile)
}

// Метр проекц (UTM) — координат градусын хүрээнд байхгүй.
func TestParse_RejectsProjectedCoordinates(t *testing.T) {
	utm := [][2]float64{
		{478000, 5279000}, {478500, 5279000},
		{478500, 5279500}, {478000, 5279500}, {478000, 5279000},
	}
	_, err := shapefile.Parse(buildShp(t, utm, 5))
	assert.ErrorIs(t, err, shapefile.ErrProjection)
}

// Тэнхлэг сольсон (өргөрөг, уртраг) — өргөрөг 90-аас хэтэрнэ.
func TestParse_RejectsSwappedAxes(t *testing.T) {
	swapped := [][2]float64{
		{47.66, 106.76}, {47.66, 106.82},
		{47.72, 106.82}, {47.72, 106.76}, {47.66, 106.76},
	}
	_, err := shapefile.Parse(buildShp(t, swapped, 5))
	assert.ErrorIs(t, err, shapefile.ErrProjection)
}

// WGS84 градус боловч Монголын гадна.
func TestParse_RejectsForeignCoordinates(t *testing.T) {
	berlin := [][2]float64{
		{13.40, 52.51}, {13.41, 52.51}, {13.41, 52.52}, {13.40, 52.52}, {13.40, 52.51},
	}
	_, err := shapefile.Parse(buildShp(t, berlin, 5))
	assert.ErrorIs(t, err, shapefile.ErrOutsideCountry)
}

func TestParse_RejectsNaNCoordinates(t *testing.T) {
	pts := [][2]float64{
		{math.NaN(), 47.66}, {106.82, 47.66},
		{106.82, 47.72}, {106.76, 47.72}, {106.76, 47.66},
	}
	_, err := shapefile.Parse(buildShp(t, pts, 5))
	assert.ErrorIs(t, err, shapefile.ErrNotShapefile)
}

// Хоёр полигон бичлэгтэй файл — аль нь гэдэг тодорхойгүй тул хаана
// (өмнө нь эхнийхийг л авч, бусад нь чимээгүй алдагддаг байв).
func TestParse_RejectsMultipleRecords(t *testing.T) {
	one := buildShp(t, testPts, 5)
	rec := one[100:] // бичлэгийн толгой + агуулга
	two := append(append([]byte{}, one...), rec...)
	// Толгой дээрх файлын уртыг шинэ хэмжээгээр зөв болгоно.
	binary.BigEndian.PutUint32(two[24:], uint32(len(two)/2))
	binary.BigEndian.PutUint32(two[100+len(rec):], 2) // 2-р бичлэгийн дугаар

	_, err := shapefile.Parse(two)
	assert.ErrorIs(t, err, shapefile.ErrMultipleRecords)
}

// ── Геометрийн төрөл / рингийн бүтэц ─────────────────────────

// buildMultiPartShp — олон part (ринг) бүхий Polygon .shp байт бүтээнэ.
func buildMultiPartShp(t *testing.T, rings [][][2]float64, shapeType uint32) []byte {
	t.Helper()

	var allPts [][2]float64
	offsets := make([]int, 0, len(rings))
	for _, r := range rings {
		offsets = append(offsets, len(allPts))
		allPts = append(allPts, r...)
	}
	numParts, numPoints := len(rings), len(allPts)

	contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
	buf := make([]byte, 100+8+contentLen)
	binary.BigEndian.PutUint32(buf[0:], 9994)
	binary.BigEndian.PutUint32(buf[24:], uint32((100+8+contentLen)/2))
	binary.LittleEndian.PutUint32(buf[28:], 1000)
	binary.LittleEndian.PutUint32(buf[32:], shapeType)
	binary.BigEndian.PutUint32(buf[100:], 1)
	binary.BigEndian.PutUint32(buf[104:], uint32(contentLen/2))

	off := 108
	binary.LittleEndian.PutUint32(buf[off:], shapeType)
	off += 4 + 32
	binary.LittleEndian.PutUint32(buf[off:], uint32(numParts))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], uint32(numPoints))
	off += 4
	for _, o := range offsets {
		binary.LittleEndian.PutUint32(buf[off:], uint32(o))
		off += 4
	}
	for _, p := range allPts {
		binary.LittleEndian.PutUint64(buf[off:], math.Float64bits(p[0]))
		off += 8
		binary.LittleEndian.PutUint64(buf[off:], math.Float64bits(p[1]))
		off += 8
	}
	return buf
}

var (
	// outerRing — гадна ринг.
	outerRing = [][2]float64{
		{106.70, 47.60}, {106.90, 47.60}, {106.90, 47.80}, {106.70, 47.80}, {106.70, 47.60},
	}
	// holeRing — outerRing ДОТОР байрлах цоорхой.
	holeRing = [][2]float64{
		{106.75, 47.65}, {106.80, 47.65}, {106.80, 47.70}, {106.75, 47.70}, {106.75, 47.65},
	}
	// islandRing — outerRing-ээс САЛАНГИД (өөр) полигон.
	islandRing = [][2]float64{
		{107.10, 47.60}, {107.20, 47.60}, {107.20, 47.70}, {107.10, 47.70}, {107.10, 47.60},
	}
)

// Салангид хоёр полигон = MultiPolygon → geometry(Polygon,4326)-д
// хадгалагдахгүй тул хаана. (Өмнө нь 2-р рингийг ЦООРХОЙ гэж уншиж,
// "Hole lies outside shell" гэсэн НЕВАЛИД геометр хадгалагддаг байв.)
func TestParse_RejectsMultiPolygon(t *testing.T) {
	data := buildMultiPartShp(t, [][][2]float64{outerRing, islandRing}, 5)
	_, err := shapefile.Parse(data)
	assert.ErrorIs(t, err, shapefile.ErrMultiPolygon)
}

// Цоорхой (нэг гадна ринг + дотор ринг) нь MultiPolygon БИШ — уншигдана.
func TestParse_AcceptsPolygonWithHole(t *testing.T) {
	data := buildMultiPartShp(t, [][][2]float64{outerRing, holeRing}, 5)
	wkt, err := shapefile.Parse(data)
	require.NoError(t, err)
	assert.Contains(t, wkt, "),(")
}

// Файлд цоорхой нь ЭХНИЙ part байсан ч WKT-д гадна ринг эхэлнэ.
func TestParse_PutsShellFirstInWKT(t *testing.T) {
	data := buildMultiPartShp(t, [][][2]float64{holeRing, outerRing}, 5)
	wkt, err := shapefile.Parse(data)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(wkt, "POLYGON((106.700000 47.600000"),
		"гадна ринг эхэнд байх ёстой, гарсан: %s", wkt)
}

// PolygonM (25) нь мөн полигон тул уншигдана.
func TestParse_AcceptsPolygonM(t *testing.T) {
	wkt, err := shapefile.Parse(buildShp(t, testPts, 25))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(wkt, "POLYGON(("))
}

// Point / MultiPoint / MultiPatch — төрөл нь мессежид тодорхой заагдана.
func TestParse_RejectsNonPolygonTypes(t *testing.T) {
	for _, tc := range []struct {
		shapeType uint32
		want      string
	}{
		{1, "Point"},
		{3, "PolyLine"},
		{8, "MultiPoint"},
		{31, "MultiPatch"},
	} {
		_, err := shapefile.Parse(buildShp(t, testPts, tc.shapeType))
		assert.ErrorIs(t, err, shapefile.ErrShapeType)
		assert.Contains(t, err.Error(), tc.want)
	}
}

// Хаагдаагүй ринг (эхний ≠ сүүлийн цэг) — PostGIS дээр 500 болохоос
// сэргийлж эндээ хаана.
func TestParse_RejectsUnclosedRing(t *testing.T) {
	open := [][2]float64{
		{106.76, 47.66}, {106.82, 47.66}, {106.82, 47.72}, {106.76, 47.72},
	}
	_, err := shapefile.Parse(buildShp(t, open, 5))
	assert.ErrorIs(t, err, shapefile.ErrRingNotClosed)
}
