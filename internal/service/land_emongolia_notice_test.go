package service_test

// E-Mongolia-гаар мэдэгдэл хүргүүлэх урсгал.
//
// Гол шалгах зүйл: МОК хэрэгжүүлэлт үед түүхэд `mocked` гэж бүртгэгдэнэ
// (`sent` БИШ) — иргэнд хүрсэн гэж эндүүрэхгүй байхын тулд.

import (
	"context"
	"errors"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubEMongolia — гадаад гарцын мок (алдаа шиднэ эсвэл лавлагаа буцаана).
type stubEMongolia struct {
	req    service.EMongoliaNoticeRequest
	result *service.EMongoliaNoticeResult
	err    error
	calls  int
}

func (s *stubEMongolia) SendNotice(
	_ context.Context, req service.EMongoliaNoticeRequest,
) (*service.EMongoliaNoticeResult, error) {
	s.calls++
	s.req = req
	return s.result, s.err
}

func TestSendParcelEMongoliaNotice_MockedIsRecordedAsMocked(t *testing.T) {
	parcelID, parcel := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(parcel, nil)

	var stored *domain.ParcelEMongoliaNotice
	repo.On("AddParcelEMongoliaNotice", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			stored = args.Get(1).(*domain.ParcelEMongoliaNotice)
		}).Return(nil)

	gateway := &stubEMongolia{result: &service.EMongoliaNoticeResult{ExternalRef: "MOCK-ABC123", Mocked: true}}
	files := newMemStorage()
	svc := service.NewLandService(repo, files, nil, gateway)

	pdf := []byte("%PDF-1.4 notice")
	res, err := svc.SendParcelEMongoliaNotice(ctx, parcelID, service.EMongoliaNoticeInput{
		RegisterNo:    " УА00112233 ",
		RecipientName: "Дорж Бат",
		FileName:      "medegdeh_huudas.pdf",
		FileData:      pdf,
		SentBy:        "7",
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// МОК тул "sent" БИШ, "mocked"
	assert.Equal(t, domain.EMongoliaStatusMocked, res.Status)
	assert.Equal(t, "MOCK-ABC123", res.ExternalRef)
	assert.Equal(t, "УА00112233", res.RegisterNo, "регистр цэвэрлэгдэнэ")

	// Гадаад гарц рүү ЯГ ТЭР PDF, нэгж талбарын дугаартай хамт явсан
	require.Equal(t, 1, gateway.calls)
	assert.Equal(t, pdf, gateway.req.FileData)
	assert.Equal(t, "1800803094", gateway.req.ParcelCode)
	assert.Contains(t, gateway.req.Message, "1800803094")

	// Илгээсэн файл хадгалагдсан (дараа нь дахин харах боломжтой)
	assert.Contains(t, res.FileURL, "emongolia/"+parcelID.String())
	assert.Len(t, files.saved, 1)

	require.NotNil(t, stored)
	assert.Equal(t, domain.EMongoliaStatusMocked, stored.Status)
}

func TestSendParcelEMongoliaNotice_FailureIsRecorded(t *testing.T) {
	parcelID, parcel := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(parcel, nil)

	var stored *domain.ParcelEMongoliaNotice
	repo.On("AddParcelEMongoliaNotice", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			stored = args.Get(1).(*domain.ParcelEMongoliaNotice)
		}).Return(nil)

	gateway := &stubEMongolia{err: errors.New("gateway 503")}
	svc := service.NewLandService(repo, newMemStorage(), nil, gateway)

	_, err := svc.SendParcelEMongoliaNotice(ctx, parcelID, service.EMongoliaNoticeInput{
		RegisterNo: "УА00112233",
		FileData:   []byte("%PDF-1.4"),
	})
	require.ErrorIs(t, err, domain.ErrEMongoliaSendFailed)
	require.NotNil(t, stored)
	assert.Equal(t, domain.EMongoliaStatusFailed, stored.Status)
	assert.Contains(t, stored.ErrorMessage, "503")
}

func TestSendParcelEMongoliaNotice_RejectsInvalidInput(t *testing.T) {
	parcelID, _ := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	gateway := &stubEMongolia{}
	svc := service.NewLandService(repo, newMemStorage(), nil, gateway)

	// Регистрийн дугааргүй — файл хадгалахгүй, гарц рүү хандахгүй
	_, err := svc.SendParcelEMongoliaNotice(ctx, parcelID, service.EMongoliaNoticeInput{
		FileData: []byte("%PDF"),
	})
	assert.ErrorIs(t, err, domain.ErrRegisterNoRequired)

	// Файлгүй
	_, err = svc.SendParcelEMongoliaNotice(ctx, parcelID, service.EMongoliaNoticeInput{
		RegisterNo: "УА00112233",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	assert.Equal(t, 0, gateway.calls)
	repo.AssertNotCalled(t, "AddParcelEMongoliaNotice", mock.Anything, mock.Anything)
}

// Гарц огт тохируулаагүй (мок ч байхгүй) — чимээгүй "илгээгдлээ" гэж хэлэхгүй.
func TestSendParcelEMongoliaNotice_NotConfigured(t *testing.T) {
	parcelID, _ := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	svc := service.NewLandService(repo, newMemStorage(), nil)

	_, err := svc.SendParcelEMongoliaNotice(ctx, parcelID, service.EMongoliaNoticeInput{
		RegisterNo: "УА00112233",
		FileData:   []byte("%PDF"),
	})
	assert.ErrorIs(t, err, domain.ErrEMongoliaNotConfigured)
}
