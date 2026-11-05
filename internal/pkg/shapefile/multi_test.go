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

// buildMultiShp — бичлэг бүр нь рингүүдийн жагсаалт (part бүр нэг ринг).
func buildMultiShp(records [][][][2]float64) []byte {
	var body []byte
	for n, rings := range records {
		numParts := len(rings)
		numPoints := 0
		for _, r := range rings {
			numPoints += len(r)
		}
		contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
		rec := make([]byte, 8+contentLen)
		binary.BigEndian.PutUint32(rec[0:], uint32(n+1))
		binary.BigEndian.PutUint32(rec[4:], uint32(contentLen/2))
		off := 8
		binary.LittleEndian.PutUint32(rec[off:], 5)
		off += 4 + 32
		binary.LittleEndian.PutUint32(rec[off:], uint32(numParts))
		binary.LittleEndian.PutUint32(rec[off+4:], uint32(numPoints))
		off += 8
		start := 0
		for _, r := range rings {
			binary.LittleEndian.PutUint32(rec[off:], uint32(start))
			off += 4
			start += len(r)
		}
		for _, r := range rings {
			for _, p := range r {
				binary.LittleEndian.PutUint64(rec[off:], math.Float64bits(p[0]))
				binary.LittleEndian.PutUint64(rec[off+8:], math.Float64bits(p[1]))
				off += 16
			}
		}
		body = append(body, rec...)
	}
	buf := make([]byte, 100+len(body))
	binary.BigEndian.PutUint32(buf[0:], 9994)
	binary.BigEndian.PutUint32(buf[24:], uint32(len(buf)/2))
	binary.LittleEndian.PutUint32(buf[28:], 1000)
	binary.LittleEndian.PutUint32(buf[32:], 5)
	copy(buf[100:], body)
	return buf
}

func square(x, y, d float64) [][2]float64 {
	return [][2]float64{{x, y}, {x + d, y}, {x + d, y + d}, {x, y + d}, {x, y}}
}

func TestParseAll_MultipleRecords(t *testing.T) {
	data := buildMultiShp([][][][2]float64{
		{square(106.90, 47.90, 0.01)},
		{square(106.95, 47.90, 0.01)},
		{square(107.00, 47.90, 0.01)},
	})
	wkts, err := shapefile.ParseAll(data)
	require.NoError(t, err)
	require.Len(t, wkts, 3)
	for _, w := range wkts {
		assert.True(t, strings.HasPrefix(w, "MULTIPOLYGON((("), w)
	}
}

func TestParseAll_RecordWithTwoShellsAndHole(t *testing.T) {
	data := buildMultiShp([][][][2]float64{{
		square(106.90, 47.90, 0.02),
		square(106.905, 47.905, 0.005), // цоорхой — эхний дотор
		square(107.00, 47.90, 0.01),    // салангид хоёр дахь полигон
	}})
	wkts, err := shapefile.ParseAll(data)
	require.NoError(t, err)
	require.Len(t, wkts, 1)
	// Хоёр полигон; эхнийх нь цоорхойтой.
	assert.Equal(t, 2, strings.Count(wkts[0], "((")-0)
}

func TestParseAll_RejectsMeterProjection(t *testing.T) {
	data := buildMultiShp([][][][2]float64{{square(478000, 5279000, 1000)}})
	_, err := shapefile.ParseAll(data)
	assert.ErrorIs(t, err, shapefile.ErrProjection)
}

func TestParseAll_RejectsOutsideMongolia(t *testing.T) {
	data := buildMultiShp([][][][2]float64{{square(2.30, 48.85, 0.05)}})
	_, err := shapefile.ParseAll(data)
	assert.ErrorIs(t, err, shapefile.ErrOutsideCountry)
}

func TestCheckPRJ(t *testing.T) {
	assert.NoError(t, shapefile.CheckPRJ(`GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137.0,298.257223563]],PRIMEM["Greenwich",0.0],UNIT["Degree",0.0174532925199433]]`))
	assert.NoError(t, shapefile.CheckPRJ(""))
	assert.ErrorIs(t, shapefile.CheckPRJ(`PROJCS["WGS_1984_UTM_Zone_48N",GEOGCS["GCS_WGS_1984"],PROJECTION["Transverse_Mercator"]]`), shapefile.ErrPRJNotWGS84)
	assert.ErrorIs(t, shapefile.CheckPRJ(`GEOGCS["GCS_Beijing_1954",DATUM["D_Beijing_1954"]]`), shapefile.ErrPRJNotWGS84)
}
