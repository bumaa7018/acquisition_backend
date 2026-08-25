package postgres

// Газрын ТӨЛБӨРИЙН БОДОЛТ (parcel_fee) ба суурь үнийн ХҮЧИН ЗҮЙЛ
// (parcel_base_price_factor) — хоёулаа ГУС-аас татагдаж бүхэлдээ орлуулагддаг
// бүртгэл.

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// ReplaceParcelFeeItems — төлбөрийн бодолтыг БҮХЭЛДЭЭ орлуулна.
//
// Бүсийн мөрөнд эх системд тогтвортой түлхүүр байхгүй (zone_id хоосон байж
// болно) тул upsert хийх боломжгүй: хуучин мөрийг устгаад шинээр бичнэ. Ингэснээр
// эх системд байхгүй болсон бүс үлдэж нийт төлбөр хөөрөгдөхөөс сэргийлнэ.
func (r *landAcqRepo) ReplaceParcelFeeItems(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	items []domain.ParcelFeeItem,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM parcel_fee WHERE parcel_id = $1", parcelID); err != nil {
		return 0, err
	}

	saved := 0
	for i := range items {
		v := items[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_fee (
				parcel_id, acquisition_id, item_no,
				zone_id, zone_type, zone_no, zone_name,
				area, landuse_area, zone_area,
				base_fee_id, resolution_id, confidence_percent,
				subsidized_area, subsidized_fee_rate,
				base_price_m2, base_fee_per_m2, landuses, value, payment
			) VALUES (
				$1, $2, $3,
				NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''),
				$8, $9, $10,
				NULLIF($11, ''), NULLIF($12, ''), $13,
				$14, $15,
				$16, $17, NULLIF($18, ''), $19, $20
			)
		`, parcelID, acquisitionID, i+1,
			v.ZoneID, v.ZoneType, v.ZoneNo, v.ZoneName,
			v.Area, v.LanduseArea, v.ZoneArea,
			v.BaseFeeID, v.ResolutionID, v.ConfidencePercent,
			v.SubsidizedArea, v.SubsidizedFeeRate,
			v.BasePriceM2, v.BaseFeePerM2, v.Landuses, v.Value, v.Payment,
		); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

func (r *landAcqRepo) ListParcelFeeItems(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelFeeItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(zone_id,''), COALESCE(zone_type,''), COALESCE(zone_no,''),
		       COALESCE(zone_name,''), COALESCE(area,0), COALESCE(landuse_area,0),
		       COALESCE(zone_area,0), COALESCE(base_fee_id,''), COALESCE(resolution_id,''),
		       COALESCE(confidence_percent,0), COALESCE(subsidized_area,0),
		       COALESCE(subsidized_fee_rate,0), COALESCE(base_price_m2,0),
		       COALESCE(base_fee_per_m2,0), COALESCE(landuses,''), COALESCE(value,0),
		       COALESCE(payment,0)
		FROM parcel_fee WHERE parcel_id = $1 ORDER BY item_no
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelFeeItem
	for rows.Next() {
		v := domain.ParcelFeeItem{}
		if err := rows.Scan(&v.ZoneID, &v.ZoneType, &v.ZoneNo, &v.ZoneName,
			&v.Area, &v.LanduseArea, &v.ZoneArea, &v.BaseFeeID, &v.ResolutionID,
			&v.ConfidencePercent, &v.SubsidizedArea, &v.SubsidizedFeeRate,
			&v.BasePriceM2, &v.BaseFeePerM2, &v.Landuses, &v.Value, &v.Payment); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// ReplaceParcelBasePriceFactors — хүчин зүйлийн утгыг factor_id-гаар upsert
// хийж, эх системд байхгүй болсныг устгана.
func (r *landAcqRepo) ReplaceParcelBasePriceFactors(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	items []domain.ParcelBasePriceFactor,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keep := make([]int, 0, len(items))
	for _, v := range items {
		if v.FactorID != 0 {
			keep = append(keep, v.FactorID)
		}
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_base_price_factor WHERE parcel_id = $1 AND NOT (factor_id = ANY($2))",
		parcelID, keep,
	); err != nil {
		return 0, err
	}

	saved := 0
	for _, v := range items {
		if v.FactorID == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_base_price_factor (
				parcel_id, acquisition_id, factor_id, factor_code, factor_name,
				factor_data_type, factor_value, year_value, in_active, closets_parcel_id
			) VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7, $8, $9, NULLIF($10,''))
			ON CONFLICT (parcel_id, factor_id) DO UPDATE SET
				acquisition_id    = EXCLUDED.acquisition_id,
				factor_code       = EXCLUDED.factor_code,
				factor_name       = EXCLUDED.factor_name,
				factor_data_type  = EXCLUDED.factor_data_type,
				factor_value      = EXCLUDED.factor_value,
				year_value        = EXCLUDED.year_value,
				in_active         = EXCLUDED.in_active,
				closets_parcel_id = EXCLUDED.closets_parcel_id,
				updated_at        = NOW()
		`, parcelID, acquisitionID, v.FactorID, v.FactorCode, v.FactorName,
			v.FactorDataType, v.FactorValue, v.YearValue, v.InActive, v.ClosetsParcelID,
		); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

func (r *landAcqRepo) ListParcelBasePriceFactors(
	ctx context.Context, parcelID uuid.UUID,
) ([]domain.ParcelBasePriceFactor, error) {
	rows, err := r.db.Query(ctx, `
		SELECT factor_id, COALESCE(factor_code,''), COALESCE(factor_name,''),
		       COALESCE(factor_data_type,''), factor_value, year_value, in_active,
		       COALESCE(closets_parcel_id,'')
		FROM parcel_base_price_factor WHERE parcel_id = $1 ORDER BY factor_id
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelBasePriceFactor
	for rows.Next() {
		v := domain.ParcelBasePriceFactor{}
		if err := rows.Scan(&v.FactorID, &v.FactorCode, &v.FactorName,
			&v.FactorDataType, &v.FactorValue, &v.YearValue, &v.InActive,
			&v.ClosetsParcelID); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
