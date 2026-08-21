//go:build e2e

// SyncParcel-ийн НӨХӨХ урсгалын E2E шалгалт (БОДИТ PostgreSQL/PostGIS дээр).
//
// Ажиллуулах:
//
//	make test-e2e
//	# эсвэл
//	E2E_DATABASE_URL='postgres://postgres:pass@localhost:5432/appdb?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run E2E_SyncParcel -v
//
// Яагаад E2E шаардлагатай вэ: /parcel/info нь бүртгэлийн баганыг БАС буцаадаг
// болсон тул ReplaceParcelDetail нь дутуу бүртгэлийг нөхөх ба ApplyParcelGeometry
// нь MULTIPOLYGON-ыг polygon болгож хөрвүүлэх ёстой. Хоёул PostGIS-ийн бодит
// зан үйлээс (cast унах, ST_Intersection) хамаардаг тул mock-оор шалгах
// боломжгүй.
package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Гол шалгалт: зөвхөн ДУГААРААР бүртгэгдсэн нэгж талбарын дутуу мэдээлэл
// /parcel/info-оос ирсэн утгаар нөхөгдөж, дараагийн дуудалт хоосон утгаар
// дарж бичихгүй.
func TestE2E_SyncParcel_BackfillsRegistryColumns(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	code := fmt.Sprintf("E2ES%s", uuid.NewString()[:8])
	// ── Хуучин хувилбарын бүртгэл: зөвхөн дугаар ────────────────────────────
	res, err := f.repo.InsertParcelCodes(ctx, f.acqID,
		[]*domain.SourceParcel{{ParcelID: code}}, e2eBoundary)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	var parcelUUID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM parcel WHERE parcel_id = $1`, code).Scan(&parcelUUID))

	validFrom := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	// ── /parcel/info-ийн бүрэн хариулт: бүртгэл + эрх зүйн дэлгэрэнгүй ──────
	require.NoError(t, f.repo.ReplaceParcelDetail(ctx, parcelUUID, f.acqID, uuid.Nil,
		&domain.SourceParcel{
			ParcelID:       code,
			OldParcelID:    "OLD-9",
			AU2Code:        "0101",
			RightType:      2,
			Landuse:        "2201",
			LanduseName:    "Гэр бүлийн хэрэгцээ",
			AreaM2:         3_000_000,
			AddressKhashaa: "12",
			AddressStreet:  "Их тойруу",
			PropertyNo:     "Э-1",
			ValidFrom:      &validFrom,
			AppNo:          "E2E-APP-1",
		}))

	row := readParcelSourceRow(t, f, code)
	assert.Equal(t, "OLD-9", row.oldParcelID, "бүртгэлийн багана нөхөгдөх ёстой")
	assert.Equal(t, "0101", row.au2)
	assert.Equal(t, "2201", row.landuse)
	assert.Equal(t, "Гэр бүлийн хэрэгцээ", row.landuseNm)
	assert.Equal(t, "12", row.khashaa)
	assert.Equal(t, "Их тойруу", row.street)
	assert.Equal(t, "Э-1", row.propertyNo)
	assert.Equal(t, 3_000_000.0, row.areaM2)
	assert.Equal(t, 2, row.rightType)
	assert.Equal(t, "E2E-APP-1", row.appNo, "эрх зүйн дэлгэрэнгүй үүсэх ёстой")

	// ── ДУТУУ хариулт бүртгэлийг ХООСЛОХГҮЙ ────────────────────────────────
	require.NoError(t, f.repo.ReplaceParcelDetail(ctx, parcelUUID, f.acqID, uuid.Nil,
		&domain.SourceParcel{ParcelID: code, AppNo: "E2E-APP-2"}))

	kept := readParcelSourceRow(t, f, code)
	assert.Equal(t, "2201", kept.landuse, "хоосон ирэхэд хадгалагдсан утга үлдэнэ")
	assert.Equal(t, "Их тойруу", kept.street)
	assert.Equal(t, 3_000_000.0, kept.areaM2)
	assert.Equal(t, 2, kept.rightType, "right_type 0 ирэхэд хадгалагдсанаар үлдэнэ")
	assert.Equal(t, "E2E-APP-2", kept.appNo, "дэлгэрэнгүй нь ОРЛУУЛАГДАНА")
}

// ApplyParcelGeometry нь нэг хэсэгтэй MULTIPOLYGON-ыг polygon болгож хадгалж,
// давхцлыг тооцох ёстой (шууд cast нь query-г унагаадаг).
func TestE2E_SyncParcel_AppliesMultipolygonGeometry(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	code := fmt.Sprintf("E2EM%s", uuid.NewString()[:8])
	res, err := f.repo.InsertParcelCodes(ctx, f.acqID,
		[]*domain.SourceParcel{{ParcelID: code, AreaM2: 3_000_000}}, e2eBoundary)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	var parcelUUID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM parcel WHERE parcel_id = $1`, code).Scan(&parcelUUID))
	require.Empty(t, readParcelSourceRow(t, f, code).geomType, "эхэндээ геометргүй")

	// Хилээс (106.76..106.82) хагас гарсан, НЭГ хэсэгтэй MULTIPOLYGON.
	multi := "MULTIPOLYGON(((106.80 47.70,106.84 47.70,106.84 47.71,106.80 47.71,106.80 47.70)))"

	geom, err := f.repo.ApplyParcelGeometry(ctx, parcelUUID, multi, e2eBoundary)
	require.NoError(t, err, "MULTIPOLYGON дээр query УНАХ ЁСГҮЙ")
	assert.True(t, geom.HasOverlap)
	assert.Greater(t, geom.AcquisitionAreaM2, 0.0)
	assert.Greater(t, geom.RemainingAreaM2, 0.0, "хилээс гарсан хэсэг бий")

	row := readParcelSourceRow(t, f, code)
	assert.Equal(t, "POLYGON", row.geomType, "geometry(Polygon,4326) болж хөрвөнө")
	assert.True(t, row.hasAcquisitionGeom)

	// Олон хэсэгтэй MULTIPOLYGON — хөрвүүлэлт NULL, хадгалагдсан хэвээр үлдэнэ.
	twoParts := "MULTIPOLYGON(((106.80 47.70,106.81 47.70,106.81 47.71,106.80 47.71,106.80 47.70))," +
		"((106.78 47.68,106.79 47.68,106.79 47.69,106.78 47.69,106.78 47.68)))"
	_, err = f.repo.ApplyParcelGeometry(ctx, parcelUUID, twoParts, e2eBoundary)
	require.NoError(t, err)

	after := readParcelSourceRow(t, f, code)
	assert.Equal(t, "POLYGON", after.geomType, "хөрвөхгүй геометр хадгалагдсаныг ХООСЛОХГҮЙ")
	assert.True(t, after.hasAcquisitionGeom)
}
