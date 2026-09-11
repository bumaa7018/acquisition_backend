package service

import (
	"context"
	"math"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// CalculateParcelEstimatedValue uses saved fee rates and one percentage for all
// zones. Affected area is distributed proportionally to the recorded fee areas;
// this is an estimate, since fee-zone intersection geometries are not stored.
//
// ӨМЧЛӨХ эрхийн газарт (right_type=3) итгэлцүүр ХЭРЭГЛЭХГҮЙ: суурь үнэ нь
// газрын зах зээлийн жишиг үнэ бөгөөс өртсөн талбайгаар шууд үржигдэнэ.
// Дуудлага худалдааны анхны үнийн итгэлцүүр нь ЭЗЭМШИХ/АШИГЛАХ эрхийн газрын
// суурь төлбөрийг зах зээлийн үнэ болгож хувиргах зорилготой — өмчлөлд тэр
// хувиргалт утгагүй. Client итгэлцүүр илгээсэн ч ЭНД алгасана (эрхийн төрлөөр
// шийднэ, client-ийн хүсэлтээр биш).
//
// ХАВСРАЛТ (attachment) нь ЗААВАЛ БИШ: тооцооллыг гэрчлэх баримт (зах зээлийн
// жишиг үнийн лавлагаа, судалгаа) хавсаргаж болно. Файл ЭХЛЭЭД хадгалалт
// (MinIO) руу бичигдэж, дараа нь үнэлгээ хадгалагдана; хадгалалт унавал
// орфон файл үлдэхгүйн тулд цэвэрлэгдэнэ (valuation-return-тай ижил зарчим).
func (s *landService) CalculateParcelEstimatedValue(ctx context.Context, acquisitionID, parcelID uuid.UUID, confidencePercent *float64, baseFeePerM2 *float64, attachment *EstimatedValueAttachment, userID string) (float64, error) {
	p, err := s.landRepo.GetParcelDetailByID(ctx, parcelID)
	if err != nil {
		return 0, err
	}
	if p.AcquisitionID != acquisitionID {
		return 0, domain.ErrNotFound
	}
	if p.RightType == domain.RightTypeOwnership {
		confidencePercent = nil
	} else if confidencePercent == nil || !finitePositive(*confidencePercent) {
		return 0, domain.ErrInvalidInput
	}
	fees, err := s.landRepo.ListParcelFeeItems(ctx, parcelID)
	if err != nil {
		return 0, err
	}
	if len(fees) == 0 {
		if baseFeePerM2 == nil {
			baseFeePerM2 = p.EstimatedBaseFeePerM2
		}
		if baseFeePerM2 == nil || !finitePositive(*baseFeePerM2) {
			return 0, domain.ErrInvalidInput
		}
		fees = []domain.ParcelFeeItem{{BaseFeePerM2: *baseFeePerM2, Area: p.AcquisitionAreaM2}}
	} else if baseFeePerM2 != nil {
		// Saved middleware rates remain authoritative when available.
		return 0, domain.ErrInvalidInput
	}
	value, err := calculateEstimatedValue(fees, p.AcquisitionAreaM2, confidencePercent)
	if err != nil {
		return 0, err
	}

	change := domain.ParcelEstimatedValueChange{
		AcquisitionID:     acquisitionID,
		ParcelID:          parcelID,
		Value:             &value,
		ConfidencePercent: confidencePercent,
		BaseFeePerM2:      baseFeePerM2,
		AreaM2:            p.AcquisitionAreaM2,
		UserID:            userID,
	}
	if attachment != nil {
		if s.files == nil {
			return 0, domain.ErrInternalServer
		}
		stored, err := s.files.Save(ctx, "estimated-value", parcelID.String(),
			attachment.Filename, attachment.ContentType, attachment.Data, true)
		if err != nil {
			return 0, err
		}
		change.AttachmentURL, change.AttachmentName = stored.URL, stored.Name
		defer func() {
			// Хадгалалт унасан бол тавьсан файлыг цэвэрлэнэ.
			if !attachment.committed {
				_ = s.files.Delete(ctx, stored.URL)
			}
		}()
	}
	if err := s.landRepo.SetParcelEstimatedValue(ctx, change); err != nil {
		return 0, err
	}
	if attachment != nil {
		attachment.committed = true
	}
	s.auditLog(ctx, "estimated_value_set", domain.AuditResourceParcel, parcelID.String(),
		uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
			"estimated_value": value, "confidence_percent": confidencePercent,
			"attachment_name":        change.AttachmentName,
			"manual_base_fee_per_m2": baseFeePerM2,
			"area_m2":                p.AcquisitionAreaM2, "user_id": userID,
		})
	return value, nil
}

// EstimatedValueAttachment — төсөөллийн үнэлгээг гэрчлэх хавсралт (ЗААВАЛ БИШ).
//
// `committed` нь дотоод тэмдэглэгээ: үнэлгээ амжилттай хадгалагдсаны дараа л
// true болно. Үүнгүй бол хадгалалт унасан үед MinIO-д орфон файл үлдэнэ.
type EstimatedValueAttachment struct {
	Filename    string
	ContentType string
	Data        []byte

	committed bool
}

// ListParcelEstimatedValueHistory — төсөөллийн үнэлгээний өөрчлөлтийн түүх.
func (s *landService) ListParcelEstimatedValueHistory(
	ctx context.Context, acquisitionID, parcelID uuid.UUID,
) ([]*domain.ParcelEstimatedValueHistory, error) {
	return s.landRepo.ListParcelEstimatedValueHistory(ctx, acquisitionID, parcelID)
}

func finitePositive(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

// confidencePercent == nil бол ИТГЭЛЦҮҮРГҮЙ (өмчлөх эрхийн газар): суурь үнэ
// нь зах зээлийн жишиг үнэ тул хувиргалтгүйгээр талбайгаар үржигдэнэ.
func calculateEstimatedValue(fees []domain.ParcelFeeItem, affectedArea float64, confidencePercent *float64) (float64, error) {
	if !finitePositive(affectedArea) || len(fees) == 0 {
		return 0, domain.ErrInvalidInput
	}
	if confidencePercent != nil && !finitePositive(*confidencePercent) {
		return 0, domain.ErrInvalidInput
	}
	totalArea, weightedPrice := 0.0, 0.0
	for _, fee := range fees {
		// Match the areas used by middleware fee calculations, including both
		// land-use and remaining zone portions. Area is the legacy fallback.
		area := fee.LanduseArea + fee.ZoneArea
		if area == 0 {
			area = fee.Area
		}
		if !finitePositive(area) || !finitePositive(fee.BaseFeePerM2) {
			return 0, domain.ErrInvalidInput
		}
		perM2 := fee.BaseFeePerM2
		if confidencePercent != nil {
			perM2 = perM2 * 100 / *confidencePercent
		}
		price := math.Round(perM2)
		totalArea += area
		weightedPrice += price * area
	}
	value := math.Round(weightedPrice / totalArea * affectedArea)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= 1e18 {
		return 0, domain.ErrInvalidInput
	}
	return value, nil
}
