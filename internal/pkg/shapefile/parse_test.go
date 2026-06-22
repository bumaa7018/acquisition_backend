package shapefile_test

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/pkg/shapefile"
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

func TestParse_WrongShapeType(t *testing.T) {
	_, err := shapefile.Parse(buildShp(t, testPts, 3)) // PolyLine
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Polygon")
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
