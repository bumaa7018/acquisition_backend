package postgres

// УБЕГ-ийн газар өмчлөлийн бүртгэл (parcel_ownership) — ХУР/XYP гарцаас
// татагдаж БҮХЭЛДЭЭ орлуулагддаг бүртгэл.

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// ReplaceParcelOwnerships — өмчлөлийн мөрүүдийг БҮХЭЛДЭЭ орлуулна.
//
// УБЕГ мөр тус бүрд тогтвортой түлхүүр буцаадаггүй (нэг register_no өмчлөлийн
// түүхээр давтагдана) тул upsert хийх боломжгүй: нэгж талбарын мөрүүдийг
// устгаад шинээр бичнэ. Ингэснээр эх системд хүчингүй болсон өмчлөл үлдэхгүй.
func (r *landAcqRepo) ReplaceParcelOwnerships(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	items []domain.ParcelOwnership,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM parcel_ownership WHERE parcel_id = $1", parcelID); err != nil {
		return 0, err
	}

	saved := 0
	for i := range items {
		v := items[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_ownership (
				parcel_id, acquisition_id, item_no,
				source_parcel_id, record_date, certificate_no, landuse_name, area,
				register_no, first_name, last_name,
				ownership_type, ownership_code, service_name, service_code,
				aimag_city_name, aimag_code, soum_district_name, soum_code,
				bag_khoroo_name, bag_code, address_street_name,
				bair_number, toot, full_address
			) VALUES (
				$1, $2, $3,
				NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8::numeric, 0),
				NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''),
				NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''),
				NULLIF($16, ''), NULLIF($17, ''), NULLIF($18, ''), NULLIF($19, ''),
				NULLIF($20, ''), NULLIF($21, ''), NULLIF($22, ''),
				NULLIF($23, ''), NULLIF($24, ''), NULLIF($25, '')
			)
		`, parcelID, acquisitionID, i+1,
			v.SourceParcelID, v.RecordDate, v.CertificateNo, v.LanduseName, v.Area,
			v.RegisterNo, v.FirstName, v.LastName,
			v.OwnershipType, v.OwnershipCode, v.ServiceName, v.ServiceCode,
			v.AimagCityName, v.AimagCode, v.SoumDistrictName, v.SoumCode,
			v.BagKhorooName, v.BagCode, v.AddressStreetName,
			v.BairNumber, v.Toot, v.FullAddress,
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

// ListParcelOwnerships — өмчлөлийн бүртгэл, ХАДГАЛАГДСАН дараалалаар
// (item_no нь УБЕГ-ийн эрэмбэ: хамгийн эртний бүртгэлээс).
func (r *landAcqRepo) ListParcelOwnerships(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelOwnership, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(source_parcel_id,''), COALESCE(record_date,''),
		       COALESCE(certificate_no,''), COALESCE(landuse_name,''), COALESCE(area,0),
		       COALESCE(register_no,''), COALESCE(first_name,''), COALESCE(last_name,''),
		       COALESCE(ownership_type,''), COALESCE(ownership_code,''),
		       COALESCE(service_name,''), COALESCE(service_code,''),
		       COALESCE(aimag_city_name,''), COALESCE(aimag_code,''),
		       COALESCE(soum_district_name,''), COALESCE(soum_code,''),
		       COALESCE(bag_khoroo_name,''), COALESCE(bag_code,''),
		       COALESCE(address_street_name,''), COALESCE(bair_number,''),
		       COALESCE(toot,''), COALESCE(full_address,'')
		FROM parcel_ownership WHERE parcel_id = $1 ORDER BY item_no
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelOwnership
	for rows.Next() {
		v := domain.ParcelOwnership{}
		if err := rows.Scan(&v.ID, &v.SourceParcelID, &v.RecordDate,
			&v.CertificateNo, &v.LanduseName, &v.Area,
			&v.RegisterNo, &v.FirstName, &v.LastName,
			&v.OwnershipType, &v.OwnershipCode,
			&v.ServiceName, &v.ServiceCode,
			&v.AimagCityName, &v.AimagCode,
			&v.SoumDistrictName, &v.SoumCode,
			&v.BagKhorooName, &v.BagCode,
			&v.AddressStreetName, &v.BairNumber,
			&v.Toot, &v.FullAddress); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
