//go:build e2e

// ҮНЭЛГЭЭ БУЦААХАД ХАВСРАЛТ + НЭГЖ ТАЛБАРЫН БАЙРШЛЫН ДАВХАРДЛЫН E2E шалгалт
// (БОДИТ PostgreSQL/PostGIS дээр).
//
// Ажиллуулах:
//
//	make test-e2e
//	# эсвэл
//	E2E_DATABASE_URL='postgres://postgres:pass@localhost:5432/appdb?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run E2E_Valuation -v
//
// Яагаад E2E шаардлагатай вэ:
//   - Хавсралт нь 000016 миграцийн ШИНЭ баганууд руу бичигддэг ба түүх
//     (history) руу мөн хуулагдана — mock-оор баганын байгаа эсэхийг шалгаж
//     чадахгүй.
//   - Давхардлын тооцоо нь ST_Intersection / calculate_area_utm-ийн БОДИТ
//     зан үйлээс (шүргэлцэх хөрш vs үнэхээр давхцах) хамаардаг.
package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Буцаалтын хавсралт нь submission дээр БОЛОН түүхэн дээр хадгалагдана;
// дараагийн үйлдэл submission-ийхийг дарж бичсэн ч ТҮҮХЭН дээрх нь үлдэнэ.
func TestE2E_ValuationReturnAttachment(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	code := fmt.Sprintf("E2EV%s", uuid.NewString()[:8])
	res, err := f.repo.InsertParcelCodes(ctx, f.acqID,
		[]*domain.SourceParcel{{ParcelID: code, GeometryWKT: e2eBoundary}}, e2eBoundary)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	var parcelUUID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM parcel WHERE acquisition_id=$1 AND parcel_id=$2`,
		f.acqID, code).Scan(&parcelUUID))

	// Илгээхэд "Үнэлгээний тайлан" + "Ажлын зураг" хоёр баримт заавал.
	attachSubmitDocuments(t, f, parcelUUID)
	base := domain.ValuationTransition{
		AcquisitionID: f.acqID,
		ParcelID:      parcelUUID,
		ValuationType: domain.ValuationTypeAsset,
		UserID:        "e2e-user",
	}

	// 1) Илгээх — хавсралт өгсөн ч ХАДГАЛАГДАХГҮЙ (зөвхөн буцаалтад хамаарна).
	submit := base
	submit.Action = "submit"
	submit.Note = "илгээв"
	submit.AttachmentURL = "/api/files/valuation-return/x/ignored.pdf"
	submit.AttachmentName = "ignored.pdf"
	sub, err := f.repo.TransitionValuationSubmission(ctx, submit)
	require.NoError(t, err)
	require.Equal(t, domain.ValuationStatusSubmitted, sub.Status)
	assert.Empty(t, sub.AttachmentURL, "илгээх дээрх хавсралт хадгалагдах ёсгүй")
	assert.Empty(t, sub.AttachmentName)

	// 2) Буцаах — хавсралттай.
	ret := base
	ret.Action = "return"
	ret.Note = "3-р хуудсыг залруулна"
	ret.AttachmentURL = "/api/files/valuation-return/" + parcelUUID.String() + "/zalruulga.pdf"
	ret.AttachmentName = "zalruulga.pdf"
	sub, err = f.repo.TransitionValuationSubmission(ctx, ret)
	require.NoError(t, err)
	require.Equal(t, domain.ValuationStatusReturned, sub.Status)
	assert.Equal(t, ret.AttachmentURL, sub.AttachmentURL)
	assert.Equal(t, "zalruulga.pdf", sub.AttachmentName)

	// Уншилтын зам ч хавсралтыг буцаана.
	got, err := f.repo.GetValuationSubmission(ctx, f.acqID, parcelUUID, domain.ValuationTypeAsset)
	require.NoError(t, err)
	assert.Equal(t, ret.AttachmentURL, got.AttachmentURL)

	// 3) Дахин илгээх — submission дээрх хавсралт АРИЛНА (сүүлийн үйлдлийг
	//    харуулдаг тул), гэхдээ түүхэн дээрх БУЦААЛТЫН хавсралт үлдэнэ.
	resubmit := base
	resubmit.Action = "submit"
	resubmit.Note = "залруулаад дахин илгээв"
	sub, err = f.repo.TransitionValuationSubmission(ctx, resubmit)
	require.NoError(t, err)
	assert.Empty(t, sub.AttachmentURL, "дахин илгээхэд картны хавсралт арилна")

	hist, err := f.repo.ListValuationSubmissionHistory(ctx, f.acqID, parcelUUID, domain.ValuationTypeAsset)
	require.NoError(t, err)
	var returned *domain.ValuationSubmissionHistory
	for _, h := range hist {
		if h.Action == "return" {
			returned = h
			break
		}
	}
	require.NotNil(t, returned, "буцаалтын түүх байх ёстой")
	assert.Equal(t, ret.AttachmentURL, returned.AttachmentURL,
		"түүх нь эх сурвалж — буцаалтын хавсралт үлдэх ёстой")
	assert.Equal(t, "zalruulga.pdf", returned.AttachmentName)

	// Илгээх үйлдлүүдийн түүхэнд хавсралт БАЙХГҮЙ.
	for _, h := range hist {
		if h.Action == "submit" {
			assert.Empty(t, h.AttachmentURL, "илгээхийн түүхэнд хавсралт байх ёсгүй")
		}
	}
}

// Байршлын давхардал: ҮНЭХЭЭР давхцаж байгаа хосыг олж, зөвхөн ХИЛЭЭРЭЭ
// шүргэлцэх хөршийг давхардал гэж бүртгэхгүй (босго нь хэмжилтийн шуугианыг
// шүүнэ).
func TestE2E_ValuationParcelOverlapDetection(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	// A ба B нь ХАГАСААР давхцана (A-ийн баруун-дээд булан = B-ийн зүүн-доод).
	// C нь A-ийн ДООД хилээр л шүргэлцэх ба B-тэй огт хүрэлцэхгүй (B нь
	// y >= 47.6710-аас эхэлдэг) — иймд давхардалтай нь ЗӨВХӨН A ба B.
	const (
		aWKT = "POLYGON((106.7700 47.6700,106.7720 47.6700,106.7720 47.6720,106.7700 47.6720,106.7700 47.6700))"
		bWKT = "POLYGON((106.7710 47.6710,106.7730 47.6710,106.7730 47.6730,106.7710 47.6730,106.7710 47.6710))"
		cWKT = "POLYGON((106.7700 47.6680,106.7720 47.6680,106.7720 47.6700,106.7700 47.6700,106.7700 47.6680))"
	)
	codes := map[string]string{}
	for name, wkt := range map[string]string{"a": aWKT, "b": bWKT, "c": cWKT} {
		code := fmt.Sprintf("E2EO%s", uuid.NewString()[:8])
		codes[name] = code
		_, err := f.repo.InsertParcelCodes(ctx, f.acqID,
			[]*domain.SourceParcel{{ParcelID: code, GeometryWKT: wkt}}, e2eBoundary)
		require.NoError(t, err)
	}

	n, err := f.repo.DetectParcelOverlaps(ctx, f.acqID)
	require.NoError(t, err)
	// A ба B л давхардалтай — C нь шүргэлцсэн (талбай ≈ 0) тул орохгүй.
	assert.Equal(t, 2, n, "зөвхөн үнэхээр давхцах хос давхардалтай гэж тоологдоно")

	var aID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM parcel WHERE acquisition_id=$1 AND parcel_id=$2`,
		f.acqID, codes["a"]).Scan(&aID))

	overlaps, err := f.repo.ListParcelOverlaps(ctx, aID)
	require.NoError(t, err)
	require.Len(t, overlaps, 1, "A нь зөвхөн B-тэй давхцана")
	assert.Equal(t, codes["b"], overlaps[0].OtherParcelCode)
	assert.Greater(t, overlaps[0].OverlapAreaM2, 1.0, "давхцах талбай босгоос дээш")

	// Дахин тооцоолол нь давхардуулахгүй (мөрүүд орлуулагдана).
	n2, err := f.repo.DetectParcelOverlaps(ctx, f.acqID)
	require.NoError(t, err)
	assert.Equal(t, n, n2)
	again, err := f.repo.ListParcelOverlaps(ctx, aID)
	require.NoError(t, err)
	assert.Len(t, again, 1)
}
