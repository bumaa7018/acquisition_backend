package service

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
)

// Дроны GeoTIFF-ийн толгойг (IFD) уншиж РАСТРЫН ЗУРВАСЫН БҮТЦИЙГ тодорхойлно.
//
// ЯАГААД ХЭРЭГТЭЙ: GeoServer-т тунгалаг байдлыг хэрхэн тохируулах нь файлд
// alpha зурвас БАЙГАА ЭСЭХЭЭС шалтгаална (geoserver.Client.Configure-ийн
// тайлбарыг үзнэ үү) — alpha-тай файлд `InputTransparentColor` тавих нь alpha-г
// УСТГАЖ, фонг бүтэн өнгөөр (цагаанаар) харуулдаг.
//
// GeoServer-ийн REST нь зурвасын тоог (dimensions) бидний үүсгэсэн coverage
// дээр буцаадаггүй тул толгойг ӨӨРСДӨӨ уншина. Бүтэн файл татахгүй: 2-3 жижиг
// Range хүсэлт (толгой нь COG биш файлд төгсгөлд байж болно).

// TIFF-ийн тагууд.
const (
	tagSamplesPerPixel = 277
	tagExtraSamples    = 338
	tagPhotometric     = 262
)

// tiffMaxIFDEntries — эрүүл саруул IFD-д байх боломжтой тагийн дээд тоо.
// Эвдэрсэн/хорлонтой файл асар их санах ой шаардахаас сэргийлнэ.
const tiffMaxIFDEntries = 512

// droneRasterInfo — публикацид шаардлагатай растрын шинж.
type droneRasterInfo struct {
	Samples  int  // зурвасын тоо (SamplesPerPixel)
	HasAlpha bool // alpha зурвас байгаа эсэх
}

// rangeReadFunc — объектын байтын хүрээ унших. Тестэд орлуулж болно.
type rangeReadFunc func(ctx context.Context, offset, length int64) ([]byte, error)

// readDroneRasterInfo — GeoTIFF-ийн IFD0-оос зурвасын бүтцийг уншина.
//
// Алдаа буцвал дуудагч ЭНГИЙН (alpha-гүй) гэж үзэх нь ЗӨВ шийдэл: тэр үед
// `InputTransparentColor` хэрэглэгдэж, хамгийн доод тал нь хар фон арилна.
func readDroneRasterInfo(ctx context.Context, read rangeReadFunc) (*droneRasterInfo, error) {
	head, err := read(ctx, 0, 16)
	if err != nil {
		return nil, err
	}
	if len(head) < 16 {
		return nil, errors.New("tiff header too short")
	}

	var bo binary.ByteOrder
	switch {
	case head[0] == 'I' && head[1] == 'I':
		bo = binary.LittleEndian
	case head[0] == 'M' && head[1] == 'M':
		bo = binary.BigEndian
	default:
		return nil, errors.New("not a tiff: bad byte order mark")
	}

	// BigTIFF (magic 43) ба classic TIFF (magic 42) нь IFD-ийн хаяглалт,
	// тагийн бичлэгийн урлаараа ЗӨРНӨ.
	magic := bo.Uint16(head[2:4])
	big := magic == 43
	if magic != 42 && magic != 43 {
		return nil, fmt.Errorf("not a tiff: magic %d", magic)
	}

	var ifdOffset int64
	if big {
		ifdOffset = int64(bo.Uint64(head[8:16]))
	} else {
		ifdOffset = int64(bo.Uint32(head[4:8]))
	}
	if ifdOffset <= 0 {
		return nil, errors.New("tiff: bad ifd offset")
	}

	// Тагийн ТОО — дараа нь бичлэгүүдийг нэг хүсэлтээр авахын тулд.
	countLen := int64(2)
	if big {
		countLen = 8
	}
	cntBuf, err := read(ctx, ifdOffset, countLen)
	if err != nil {
		return nil, err
	}
	if int64(len(cntBuf)) < countLen {
		return nil, errors.New("tiff: truncated ifd count")
	}
	var entries int64
	if big {
		entries = int64(bo.Uint64(cntBuf))
	} else {
		entries = int64(bo.Uint16(cntBuf))
	}
	if entries <= 0 || entries > tiffMaxIFDEntries {
		return nil, fmt.Errorf("tiff: implausible ifd entry count %d", entries)
	}

	entryLen := int64(12)
	if big {
		entryLen = 20
	}
	body, err := read(ctx, ifdOffset+countLen, entries*entryLen)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) < entries*entryLen {
		return nil, errors.New("tiff: truncated ifd body")
	}

	info := &droneRasterInfo{}
	var photometric, extraCount int
	var extraInline uint64
	var haveExtra bool

	for i := int64(0); i < entries; i++ {
		e := body[i*entryLen : (i+1)*entryLen]
		tag := bo.Uint16(e[0:2])
		count := int64(bo.Uint32(e[4:8]))
		if big {
			count = int64(bo.Uint64(e[4:12]))
		}
		// Утга нь бичлэг дотор БАГТААД байвал шууд эндээс уншигдана. Багтахгүй
		// бол энэ нь хаяг болно — alpha-г тодорхойлоход тэр хэрэггүй (доорх
		// тайлбарыг үзнэ үү) тул хөөж уншихгүй.
		valField := e[8:12]
		if big {
			valField = e[12:20]
		}
		switch tag {
		case tagSamplesPerPixel:
			info.Samples = int(bo.Uint16(valField[0:2]))
		case tagPhotometric:
			photometric = int(bo.Uint16(valField[0:2]))
		case tagExtraSamples:
			haveExtra = true
			extraCount = int(count)
			// ExtraSamples нь SHORT. Ортофотод энэ нь бараг үргэлж 1 утга
			// (count=1) тул бичлэг дотор багтана.
			extraInline = uint64(bo.Uint16(valField[0:2]))
		}
	}

	// ExtraSamples = 1 (associated alpha) эсвэл 2 (unassociated alpha).
	// 0 (unspecified) бол alpha ГЭЖ ҮЗЭХГҮЙ — тэр нь ердийн нэмэлт зурвас.
	if haveExtra && extraCount >= 1 && (extraInline == 1 || extraInline == 2) {
		info.HasAlpha = true
	}
	// ExtraSamples тагийг БИЧДЭГГҮЙ хөтөлбөрүүд ч байдаг: RGB (photometric=2)
	// дээр 4 зурвас байвал 4 дэх нь бодит практикт үргэлж alpha.
	if !info.HasAlpha && info.Samples == 4 && photometric == 2 {
		info.HasAlpha = true
	}
	return info, nil
}
