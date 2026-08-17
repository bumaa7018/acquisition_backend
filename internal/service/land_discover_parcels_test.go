package service_test

// DiscoverParcels — чөлөөлөх ХИЛЭЭР нэгж талбарын ДУГААРЫГ татаж бүртгэх алхам.
//
// Энэ алхам дэлгэрэнгүй мэдээлэл ТАТАХГҮЙ (GetParcelInfo дуудагдахгүй) —
// дэлгэрэнгүйг дараа нь дугаар тус бүрээр SyncParcel татна. Тиймээс энд
// шалгах гол зүйл: хил байхгүй бол дуудалт хийхгүй, дугаарууд repo руу
// дамжина, дундын сервисийн алдаа дээр процесс зогсоно.

import (
	"errors"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discoverSvc(t *testing.T) (
	service.LandService,
	*mocks.MockLandAcquisitionRepository,
	*mocks.MockParcelInfoClient,
) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	return service.NewLandService(landRepo, nil, nil, parcelInfo), landRepo, parcelInfo
}

func TestLandService_DiscoverParcels_RegistersCodesWithoutFetchingDetails(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, wkt).
		Return([]string{"010101000001", "010101000002"}, nil)
	landRepo.On("InsertParcelCodes", ctx, id, []string{"010101000001", "010101000002"}, wkt).
		Return(&domain.ParcelDiscoveryResult{
			Total: 2, Created: 1, Existing: 1,
			ParcelIDs: []string{"010101000001", "010101000002"},
		}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Existing)
	// Дэлгэрэнгүй ЭНД татагдахгүй — 2-р алхмын ажил.
	parcelInfo.AssertNotCalled(t, "GetParcelInfo", mock.Anything, mock.Anything)
	// Байгаа нэгж талбарыг УСТГАХГҮЙ: зөвхөн дугаараар шалгаад дутууг нэмнэ.
	// (Устгаад дахин үүсгэвэл хэрэглэгчийн оруулсан үнэлгээ/олговор/төлөв алдагдана.)
	landRepo.AssertNotCalled(t, "DeleteParcelsAndAUs", mock.Anything, mock.Anything)
	landRepo.AssertExpectations(t)
	parcelInfo.AssertExpectations(t)
}

func TestLandService_DiscoverParcels_NoBoundary(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: "   "}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	parcelInfo.AssertNotCalled(t, "ListParcelIDsByAcquisition", mock.Anything, mock.Anything)
}

func TestLandService_DiscoverParcels_AcquisitionNotFound(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	parcelInfo.AssertNotCalled(t, "ListParcelIDsByAcquisition", mock.Anything, mock.Anything)
}

// Дундын сервис унасан бол бүртгэх алхам руу ОРОХГҮЙ — процесс зогсоно.
func TestLandService_DiscoverParcels_MiddlewareError(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, wkt).Return(nil, errors.New("dial tcp: refused"))

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrInternalServer)
	landRepo.AssertNotCalled(t, "InsertParcelCodes", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Хэсэг дугаар бүртгэгдээд алдаа гарсан бол БҮХ багцыг хаяхгүй — бүртгэгдсэнийг
// нь буцаана (дараагийн алхам тэдгээрийн дэлгэрэнгүйг татна).
func TestLandService_DiscoverParcels_PartialInsertKeepsResult(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, wkt).
		Return([]string{"010101000001", "010101000002"}, nil)
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Return(&domain.ParcelDiscoveryResult{
			Total: 2, Created: 1,
			ParcelIDs: []string{"010101000001"},
		}, errors.New("insert failed"))

	res, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, []string{"010101000001"}, res.ParcelIDs)
}

// Нэг ч дугаар бүртгэгдээгүй дээр алдаа гарвал алдаа буцна.
func TestLandService_DiscoverParcels_InsertErrorWithNoRows(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, wkt).Return([]string{"010101000001"}, nil)
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Return(&domain.ParcelDiscoveryResult{Total: 1}, errors.New("insert failed"))

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrInternalServer)
}
