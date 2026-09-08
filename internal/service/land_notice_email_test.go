package service_test

// Мэдэгдэх хуудсыг имэйлээр илгээх урсгалын тест.
//
// SMTP-г мокоор орлуулна (хөгжүүлэлтийн орчинд имэйл тохируулаагүй) — эндээс
// шалгах гол зүйлс: хавсралт нь PDF-ээр явж байгаа, түүх бүртгэгдэж байгаа,
// амжилтгүй үед ч мөр үлдэж байгаа.

import (
	"context"
	"errors"
	"io"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// noticeSender — email.Sender-ийн мок: илгээсэн аргументуудыг хадгална.
type noticeSender struct {
	to, subject, body, attachmentName string
	attachment                        []byte
	err                               error
	calls                             int
}

func (s *noticeSender) SendPasswordReset(string, string) error { return nil }

func (s *noticeSender) SendWithAttachment(to, subject, body, attachmentName string, attachment []byte) error {
	s.calls++
	s.to, s.subject, s.body, s.attachmentName, s.attachment = to, subject, body, attachmentName, attachment
	return s.err
}

// memStorage — файлын системийн мок (санах ойд).
type memStorage struct {
	saved   map[string][]byte
	deleted []string
}

func newMemStorage() *memStorage { return &memStorage{saved: map[string][]byte{}} }

func (m *memStorage) Save(_ context.Context, namespace, ownerID, filename, _ string, data []byte, _ bool) (*filestorage.StoredFile, error) {
	url := "/api/files/" + namespace + "/" + ownerID + "/" + filename
	m.saved[url] = data
	return &filestorage.StoredFile{Name: filename, URL: url, SizeBytes: int64(len(data))}, nil
}

func (m *memStorage) SaveStream(context.Context, string, string, string, string, io.Reader, int64, bool) (*filestorage.StoredFile, error) {
	return nil, errors.New("not implemented")
}
func (m *memStorage) Read(context.Context, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (m *memStorage) OpenStream(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}
func (m *memStorage) Delete(_ context.Context, fileURL string) error {
	m.deleted = append(m.deleted, fileURL)
	return nil
}

func noticeTestParcel() (uuid.UUID, *domain.ParcelFull) {
	parcelID := uuid.New()
	return parcelID, &domain.ParcelFull{
		Parcel: domain.Parcel{
			ID:            parcelID,
			ParcelID:      "1800803094",
			AcquisitionID: uuid.New(),
		},
	}
}

func TestSendParcelNoticeEmail_SendsPDFAndRecordsHistory(t *testing.T) {
	parcelID, parcel := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(parcel, nil)
	repo.On("AddParcelNoticeEmail", mock.Anything, mock.Anything).Return(nil)

	sender := &noticeSender{}
	files := newMemStorage()
	svc := service.NewLandService(repo, files, nil, sender)

	pdf := []byte("%PDF-1.4 notice")
	res, err := svc.SendParcelNoticeEmail(ctx, parcelID, service.NoticeEmailInput{
		ToEmail:  "iргэн@example.mn",
		FileName: "medegdeh_huudas_1800803094.pdf",
		FileData: pdf,
		SentBy:   "7",
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, domain.NoticeEmailStatusSent, res.Status)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, "iргэн@example.mn", sender.to)
	assert.Equal(t, "Урьдчилан мэдэгдэх хуудас", sender.subject)
	// Хавсралт нь дамжуулсан PDF ЯГ ТЭР байх ёстой
	assert.Equal(t, pdf, sender.attachment)
	assert.Equal(t, "medegdeh_huudas_1800803094.pdf", sender.attachmentName)
	// Биед нэгж талбарын дугаар орсон байна
	assert.Contains(t, sender.body, "1800803094")
	// Илгээсэн файл хадгалагдсан (дараа нь дахин харах боломжтой)
	assert.Contains(t, res.FileURL, "notice/"+parcelID.String())
	assert.Len(t, files.saved, 1)
	repo.AssertExpectations(t)
}

func TestSendParcelNoticeEmail_FailedSendIsRecorded(t *testing.T) {
	parcelID, parcel := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(parcel, nil)

	var stored *domain.ParcelNoticeEmail
	repo.On("AddParcelNoticeEmail", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			stored = args.Get(1).(*domain.ParcelNoticeEmail)
		}).Return(nil)

	sender := &noticeSender{err: errors.New("smtp: 550 mailbox unavailable")}
	svc := service.NewLandService(repo, newMemStorage(), nil, sender)

	_, err := svc.SendParcelNoticeEmail(ctx, parcelID, service.NoticeEmailInput{
		ToEmail:  "citizen@example.mn",
		FileName: "notice.pdf",
		FileData: []byte("%PDF-1.4"),
	})
	// Илгээгдээгүйг дуудагчид ХЭЛНЭ (чимээгүй амжилт болгохгүй)
	require.ErrorIs(t, err, domain.ErrEmailSendFailed)
	// Гэхдээ оролдлого түүхэд үлдэнэ — шалтгаантайгаа
	require.NotNil(t, stored)
	assert.Equal(t, domain.NoticeEmailStatusFailed, stored.Status)
	assert.Contains(t, stored.ErrorMessage, "550")
}

func TestSendParcelNoticeEmail_RejectsInvalidInput(t *testing.T) {
	parcelID, _ := noticeTestParcel()
	repo := new(mocks.MockLandAcquisitionRepository)
	sender := &noticeSender{}
	svc := service.NewLandService(repo, newMemStorage(), nil, sender)

	// Буруу имэйл — файл хадгалахгүй, SMTP руу хандахгүй
	_, err := svc.SendParcelNoticeEmail(ctx, parcelID, service.NoticeEmailInput{
		ToEmail: "buruu", FileData: []byte("%PDF"),
	})
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)

	// Файлгүй
	_, err = svc.SendParcelNoticeEmail(ctx, parcelID, service.NoticeEmailInput{
		ToEmail: "citizen@example.mn",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	assert.Equal(t, 0, sender.calls)
	repo.AssertNotCalled(t, "AddParcelNoticeEmail", mock.Anything, mock.Anything)
}
