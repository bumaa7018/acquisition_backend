package service

import (
	"context"
	"encoding/binary"
	"testing"
)

// tiffEntry — IFD-ийн нэг тагийн бичлэг.
type tiffEntry struct {
	tag, typ uint16
	count    uint32
	value    uint16
}

// buildClassicTIFF — тестийн classic TIFF (little endian). IFD-ийг ТӨГСГӨЛД
// тавьж болно — COG биш файлын бодит хэлбэрийг давтахын тулд.
func buildClassicTIFF(entries []tiffEntry, pad int) []byte {
	ifd := make([]byte, 2+len(entries)*12+4)
	binary.LittleEndian.PutUint16(ifd[0:2], uint16(len(entries)))
	for i, e := range entries {
		off := 2 + i*12
		binary.LittleEndian.PutUint16(ifd[off:off+2], e.tag)
		binary.LittleEndian.PutUint16(ifd[off+2:off+4], e.typ)
		binary.LittleEndian.PutUint32(ifd[off+4:off+8], e.count)
		binary.LittleEndian.PutUint16(ifd[off+8:off+10], e.value)
	}
	head := make([]byte, 16)
	head[0], head[1] = 'I', 'I'
	binary.LittleEndian.PutUint16(head[2:4], 42)
	binary.LittleEndian.PutUint32(head[4:8], uint32(16+pad))
	body := append(head, make([]byte, pad)...)
	return append(body, ifd...)
}

func readerFor(data []byte) rangeReadFunc {
	return func(_ context.Context, offset, length int64) ([]byte, error) {
		if offset >= int64(len(data)) {
			return nil, nil
		}
		end := offset + length
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		return data[offset:end], nil
	}
}

func TestReadDroneRasterInfo(t *testing.T) {
	const short, ifdOffsetPad = 3, 4096

	cases := []struct {
		name     string
		entries  []tiffEntry
		pad      int
		wantAlfa bool
		samples  int
	}{
		{
			// 6a679983a82ed85eb6b2b111.tif — 3 зурваст JPEG/YCbCr, alpha БАЙХГҮЙ.
			name: "3 зурваст RGB — alpha байхгүй",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 6},
				{tagSamplesPerPixel, short, 1, 3},
			},
			wantAlfa: false, samples: 3,
		},
		{
			// 0616hana1.tif — 4 зурваст, ExtraSamples=2 (unassociated alpha).
			name: "4 зурваст RGBA, ExtraSamples=2",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 2},
				{tagSamplesPerPixel, short, 1, 4},
				{tagExtraSamples, short, 1, 2},
			},
			wantAlfa: true, samples: 4,
		},
		{
			name: "4 зурваст RGBA, ExtraSamples=1 (associated)",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 2},
				{tagSamplesPerPixel, short, 1, 4},
				{tagExtraSamples, short, 1, 1},
			},
			wantAlfa: true, samples: 4,
		},
		{
			// ExtraSamples тагийг бичээгүй хөтөлбөрүүд ч байдаг.
			name: "4 зурваст RGB, ExtraSamples таггүй",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 2},
				{tagSamplesPerPixel, short, 1, 4},
			},
			wantAlfa: true, samples: 4,
		},
		{
			// ExtraSamples=0 (unspecified) нь alpha БИШ.
			name: "5 зурваст, ExtraSamples=0 — alpha биш",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 2},
				{tagSamplesPerPixel, short, 1, 5},
				{tagExtraSamples, short, 1, 0},
			},
			wantAlfa: false, samples: 5,
		},
		{
			// COG биш файл — IFD нь толгойгоос хол, төгсгөлд байна.
			name: "IFD төгсгөлд (COG биш)",
			entries: []tiffEntry{
				{tagPhotometric, short, 1, 2},
				{tagSamplesPerPixel, short, 1, 4},
				{tagExtraSamples, short, 1, 2},
			},
			pad:      ifdOffsetPad,
			wantAlfa: true, samples: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := buildClassicTIFF(tc.entries, tc.pad)
			info, err := readDroneRasterInfo(context.Background(), readerFor(data))
			if err != nil {
				t.Fatalf("readDroneRasterInfo: %v", err)
			}
			if info.HasAlpha != tc.wantAlfa {
				t.Errorf("HasAlpha = %v, хүсэн байсан %v", info.HasAlpha, tc.wantAlfa)
			}
			if info.Samples != tc.samples {
				t.Errorf("Samples = %d, хүсэн байсан %d", info.Samples, tc.samples)
			}
		})
	}
}

func TestReadDroneRasterInfoBigTIFF(t *testing.T) {
	// BigTIFF — IFD-ийн тоолуур 8 байт, бичлэг 20 байт.
	entries := []tiffEntry{
		{tagPhotometric, 3, 1, 2},
		{tagSamplesPerPixel, 3, 1, 4},
		{tagExtraSamples, 3, 1, 2},
	}
	ifd := make([]byte, 8+len(entries)*20+8)
	binary.LittleEndian.PutUint64(ifd[0:8], uint64(len(entries)))
	for i, e := range entries {
		off := 8 + i*20
		binary.LittleEndian.PutUint16(ifd[off:off+2], e.tag)
		binary.LittleEndian.PutUint16(ifd[off+2:off+4], e.typ)
		binary.LittleEndian.PutUint64(ifd[off+4:off+12], uint64(e.count))
		binary.LittleEndian.PutUint16(ifd[off+12:off+14], e.value)
	}
	head := make([]byte, 16)
	head[0], head[1] = 'I', 'I'
	binary.LittleEndian.PutUint16(head[2:4], 43)
	binary.LittleEndian.PutUint16(head[4:6], 8) // offset size
	binary.LittleEndian.PutUint64(head[8:16], 16)
	data := append(head, ifd...)

	info, err := readDroneRasterInfo(context.Background(), readerFor(data))
	if err != nil {
		t.Fatalf("readDroneRasterInfo: %v", err)
	}
	if !info.HasAlpha || info.Samples != 4 {
		t.Errorf("BigTIFF: HasAlpha=%v Samples=%d, хүсэн байсан true/4", info.HasAlpha, info.Samples)
	}
}

func TestReadDroneRasterInfoRejectsNonTIFF(t *testing.T) {
	for name, data := range map[string][]byte{
		"хоосон":         {},
		"хэт богино":     []byte("II*"),
		"BOM буруу":      append([]byte("XY*\x00"), make([]byte, 12)...),
		"magic буруу":    append([]byte{'I', 'I', 99, 0}, make([]byte, 12)...),
		"IFD offset тэг": append([]byte{'I', 'I', 42, 0, 0, 0, 0, 0}, make([]byte, 8)...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readDroneRasterInfo(context.Background(), readerFor(data)); err == nil {
				t.Error("алдаа хүлээж байсан, nil буцлаа")
			}
		})
	}
}
