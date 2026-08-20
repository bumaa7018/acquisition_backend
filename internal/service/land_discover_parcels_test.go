package service_test

// DiscoverParcels — чөлөөлөх ХИЛЭЭР нэгж талбарыг БАГЦААР татаж бүртгэх алхам.
//
// Багц бүр дугаар + бүртгэлийн баганыг (геометр, талбай, зориулалт) агуулна;
// ЭРХ ЗҮЙН дэлгэрэнгүй ТАТАГДАХГҮЙ (GetParcelInfo дуудагдахгүй) — түүнийг
// дараа нь дугаар тус бүрээр SyncParcel татна.
//
// Шалгах гол зүйлс: хил байхгүй бол дуудалт хийхгүй; 100-гийн багцаар
// cursor-оор гүйнэ; мөрүүд repo руу дамжина; дундын сервисийн алдаа дээр
// процесс зогсоно.

import (
	"errors"
	"fmt"
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

// parcelBatch — n мөр бүхий багц үүсгэнэ. Дугаар нь эрэмбэлэгдсэн байх ба
// cursor нь багцын СҮҮЛИЙН дугаар байхыг шалгахад хэрэглэгдэнэ.
func parcelBatch(start, n int) []*domain.SourceParcel {
	batch := make([]*domain.SourceParcel, 0, n)
	for i := 0; i < n; i++ {
		batch = append(batch, &domain.SourceParcel{
			ParcelID: fmt.Sprintf("0101010%05d", start+i),
			AreaM2:   float64(500 + i),
		})
	}
	return batch
}

func TestLandService_DiscoverParcels_RegistersRowsWithoutFetchingDetails(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	rows := []*domain.SourceParcel{
		{ParcelID: "010101000001", GeometryWKT: "POLYGON((1 1,2 1,2 2,1 2,1 1))", Landuse: "2201", AreaM2: 500},
		{ParcelID: "010101000002", AreaM2: 700},
	}
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).Return(rows, nil)
	landRepo.On("InsertParcelCodes", ctx, id, rows, wkt).
		Return(&domain.ParcelDiscoveryResult{
			Total: 2, Created: 1, Existing: 1,
			ParcelIDs: []string{"010101000001", "010101000002"},
		}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Existing)
	// ЭРХ ЗҮЙН дэлгэрэнгүй ЭНД татагдахгүй — 2-р алхмын ажил.
	parcelInfo.AssertNotCalled(t, "GetParcelInfo", mock.Anything, mock.Anything)
	// Байгаа нэгж талбарыг УСТГАХГҮЙ: зөвхөн дугаараар шалгаад дутууг нэмнэ.
	// (Устгаад дахин үүсгэвэл хэрэглэгчийн оруулсан үнэлгээ/олговор/төлөв алдагдана.)
	landRepo.AssertNotCalled(t, "DeleteParcelsAndAUs", mock.Anything, mock.Anything)
	landRepo.AssertExpectations(t)
	parcelInfo.AssertExpectations(t)
}

// 250 нэгж талбар = 100 + 100 + 50 гэсэн ГУРВАН дуудалт. Cursor нь өмнөх
// багцын СҮҮЛИЙН дугаар байх ёстой.
func TestLandService_DiscoverParcels_FetchesInBatchesOfHundred(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	first, second, third := parcelBatch(1, 100), parcelBatch(101, 100), parcelBatch(201, 50)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).Return(first, nil).Once()
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, first[99].ParcelID, 100).Return(second, nil).Once()
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, second[99].ParcelID, 100).Return(third, nil).Once()

	var saved []*domain.SourceParcel
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Run(func(args mock.Arguments) {
			saved = args.Get(2).([]*domain.SourceParcel)
		}).Return(&domain.ParcelDiscoveryResult{Total: 250, Created: 250}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, 250, res.Total)
	require.Len(t, saved, 250, "гурван багц нэгтгэгдэнэ")
	assert.Equal(t, first[0].ParcelID, saved[0].ParcelID)
	assert.Equal(t, third[49].ParcelID, saved[249].ParcelID)
	parcelInfo.AssertExpectations(t)
}

// Cursor-ыг үл хэрэгсэх (хуучин) дундый сервис ижил багцыг дахин буцаавал
// гогцоо ЗОГСОНО — мөнхийн давталт гарахгүй.
func TestLandService_DiscoverParcels_StopsWhenCursorDoesNotAdvance(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	same := parcelBatch(1, 100)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	// Cursor-аас үл хамааран ИЖИЛ багц буцаана.
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, mock.AnythingOfType("string"), 100).Return(same, nil)

	var saved []*domain.SourceParcel
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Run(func(args mock.Arguments) {
			saved = args.Get(2).([]*domain.SourceParcel)
		}).Return(&domain.ParcelDiscoveryResult{Total: 100}, nil)

	_, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	assert.Len(t, saved, 100, "давхардсан дугаар нэг л удаа бүртгэгдэнэ")
	// 1-рт багц авна, 2-рт шинэ дугаар нэмэгдээгүйг хараад зогсоно.
	parcelInfo.AssertNumberOfCalls(t, "ListParcelsByAcquisition", 2)
}

// Дараагийн багц унавал хуримтлагдсаныг бүртгэнэ (бүх багцыг хаяхгүй) —
// дахин дуудахад дутуу нь нөхөгдөнө.
func TestLandService_DiscoverParcels_KeepsEarlierBatchesWhenLaterBatchFails(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	first := parcelBatch(1, 100)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).Return(first, nil).Once()
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, first[99].ParcelID, 100).
		Return(nil, errors.New("dial tcp: refused")).Once()

	var saved []*domain.SourceParcel
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Run(func(args mock.Arguments) {
			saved = args.Get(2).([]*domain.SourceParcel)
		}).Return(&domain.ParcelDiscoveryResult{Total: 100, Created: 100}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, 100, res.Created)
	assert.Len(t, saved, 100)
	parcelInfo.AssertExpectations(t)
}

func TestLandService_DiscoverParcels_NoBoundary(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: "   "}, nil)

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	parcelInfo.AssertNotCalled(t, "ListParcelsByAcquisition", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLandService_DiscoverParcels_AcquisitionNotFound(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	parcelInfo.AssertNotCalled(t, "ListParcelsByAcquisition", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Дундын сервис унасан бол бүртгэх алхам руу ОРОХГҮЙ — процесс зогсоно.
func TestLandService_DiscoverParcels_MiddlewareError(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := discoverSvc(t)

	wkt := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, GeometryWKT: wkt}, nil)
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).Return(nil, errors.New("dial tcp: refused"))

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
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).
		Return([]*domain.SourceParcel{{ParcelID: "010101000001"}, {ParcelID: "010101000002"}}, nil)
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
	parcelInfo.On("ListParcelsByAcquisition", ctx, wkt, "", 100).Return([]*domain.SourceParcel{{ParcelID: "010101000001"}}, nil)
	landRepo.On("InsertParcelCodes", ctx, id, mock.Anything, wkt).
		Return(&domain.ParcelDiscoveryResult{Total: 1}, errors.New("insert failed"))

	res, err := svc.DiscoverParcels(ctx, id)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrInternalServer)
}
