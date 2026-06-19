// Package shapefile implements a minimal pure-Go ESRI Shapefile reader.
// Зөвхөн Polygon (type 5) болон PolygonZ (type 15) дэмждэг.
package shapefile

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// Parse reads a raw .shp file and returns the WKT of the first polygon.
func Parse(data []byte) (string, error) {
	return parsePolygon(data)
}

// parsePolygon parses the first polygon record and returns its WKT.
func parsePolygon(data []byte) (string, error) {
	const fileHeaderSize = 100
	const recordHeaderSize = 8

	if len(data) < fileHeaderSize+recordHeaderSize+4 {
		return "", fmt.Errorf("shp file too short (%d bytes)", len(data))
	}

	// first record content starts after file header + record header
	pos := fileHeaderSize + recordHeaderSize

	shapeType := int32(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	if shapeType != 5 && shapeType != 15 {
		return "", fmt.Errorf("expected Polygon (5) or PolygonZ (15), got %d", shapeType)
	}

	// bounding box: Xmin, Ymin, Xmax, Ymax (4 × float64)
	pos += 32

	numParts := int(int32(binary.LittleEndian.Uint32(data[pos:])))
	pos += 4
	numPoints := int(int32(binary.LittleEndian.Uint32(data[pos:])))
	pos += 4

	parts := make([]int, numParts)
	for i := range parts {
		parts[i] = int(int32(binary.LittleEndian.Uint32(data[pos:])))
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

	return toWKT(parts, pts, numParts, numPoints), nil
}

func toWKT(parts []int, pts [][2]float64, numParts, numPoints int) string {
	var b strings.Builder
	b.WriteString("POLYGON(")
	for p := 0; p < numParts; p++ {
		start := parts[p]
		end := numPoints
		if p+1 < numParts {
			end = parts[p+1]
		}
		if p > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		for i := start; i < end; i++ {
			if i > start {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "%.6f %.6f", pts[i][0], pts[i][1])
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	return b.String()
}
