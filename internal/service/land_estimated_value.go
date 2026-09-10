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
func (s *landService) CalculateParcelEstimatedValue(ctx context.Context, acquisitionID, parcelID uuid.UUID, confidencePercent float64, baseFeePerM2 *float64, userID string) (float64, error) {
	if !finitePositive(confidencePercent) {
		return 0, domain.ErrInvalidInput
	}
	p, err := s.landRepo.GetParcelDetailByID(ctx, parcelID)
	if err != nil {
		return 0, err
	}
	if p.AcquisitionID != acquisitionID {
		return 0, domain.ErrNotFound
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
	if err := s.landRepo.SetParcelEstimatedValue(ctx, acquisitionID, parcelID, &value, &confidencePercent, baseFeePerM2, userID); err != nil {
		return 0, err
	}
	s.auditLog(ctx, "estimated_value_set", domain.AuditResourceParcel, parcelID.String(),
		uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
			"estimated_value": value, "confidence_percent": confidencePercent,
			"manual_base_fee_per_m2": baseFeePerM2,
			"area_m2":                p.AcquisitionAreaM2, "user_id": userID,
		})
	return value, nil
}

func finitePositive(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func calculateEstimatedValue(fees []domain.ParcelFeeItem, affectedArea, confidencePercent float64) (float64, error) {
	if !finitePositive(affectedArea) || !finitePositive(confidencePercent) || len(fees) == 0 {
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
		price := math.Round(fee.BaseFeePerM2 * 100 / confidencePercent)
		totalArea += area
		weightedPrice += price * area
	}
	value := math.Round(weightedPrice / totalArea * affectedArea)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= 1e18 {
		return 0, domain.ErrInvalidInput
	}
	return value, nil
}
