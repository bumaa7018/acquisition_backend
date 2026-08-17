package service_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// syncDocsSvc — хавсралт татах урсгалын бэлтгэл. appNo нь нэгж талбарын
// дэлгэрэнгүйгээс (parcel_detail) уншигдана.
func syncDocsSvc(t *testing.T, acqID, parcelID uuid.UUID, parcelCode, appNo string) (
	service.LandService,
	*mocks.MockLandAcquisitionRepository,
	*mocks.MockParcelInfoClient,
) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo)

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{ID: acqID}, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).
		Return(&domain.Parcel{ID: parcelID, ParcelID: parcelCode}, nil)
	landRepo.On("GetParcelDetailByID", ctx, parcelID).Return(&domain.ParcelFull{
		Detail: &domain.ParcelDetailInfo{AppNo: appNo},
	}, nil)
	return svc, landRepo, parcelInfo
}

// Эх системийн хавсралтууд татагдаж, ЗӨВХӨН ШИНЭ нь бүртгэгдэнэ.
func TestLandService_SyncParcelDocuments_SavesOnlyNew(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "01107-10-00109-26"
	svc, landRepo, parcelInfo := syncDocsSvc(t, acqID, parcelID, parcelCode, appNo)

	parcelInfo.On("ListApplicationDocuments", ctx, appNo, []string(nil)).
		Return([]*domain.SourceDocument{
			{AppNo: appNo, DocName: "Өргөдөл", DocCode: "10", DocID: "5001", FileURL: "https://ftp/a.pdf", FtpID: "77"},
			{AppNo: appNo, DocName: "Кадастрын зураг", DocCode: "20", DocID: "5002", FileURL: "https://ftp/b.png"},
		}, nil)

	// Repo нь давхардсаныг өөрөө шүүнэ — 2 ирснээс 1 нь шинэ
	var sent []*domain.Document
	landRepo.On("AddSourceParcelDocuments", ctx, parcelID, mock.Anything).
		Run(func(args mock.Arguments) {
			sent = args.Get(2).([]*domain.Document)
		}).Return(1, nil)

	stored := []*domain.Document{{ID: uuid.New(), Name: "Өргөдөл", SourceDocID: "5001"}}
	landRepo.On("ListParcelDocuments", ctx, parcelID).Return(stored, nil)

	res, err := svc.SyncParcelDocuments(ctx, acqID, parcelCode, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, res.Found)
	assert.Equal(t, 1, res.Saved)
	assert.Equal(t, 1, res.Skipped, "өмнө татагдсан нь алгасагдана")
	assert.Equal(t, stored, res.Documents)

	// Эх дугаар болон файлын төрөл зөв бэлтгэгдсэн эсэх
	require.Len(t, sent, 2)
	assert.Equal(t, "5001", sent[0].SourceDocID)
	assert.Equal(t, "77", sent[0].SourceFtpID)
	assert.Equal(t, "pdf", sent[0].FileType)
	assert.Equal(t, "Өргөдөл", sent[0].Name)
	assert.Equal(t, "png", sent[1].FileType)
	assert.Empty(t, sent[1].SourceFtpID)
	parcelInfo.AssertExpectations(t)
	landRepo.AssertExpectations(t)
}

// role шүүлтүүр дундын сервис рүү дамжина.
func TestLandService_SyncParcelDocuments_PassesRoleFilter(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "01107-10-00109-26"
	svc, landRepo, parcelInfo := syncDocsSvc(t, acqID, parcelID, parcelCode, appNo)

	roles := []string{"10", "20"}
	parcelInfo.On("ListApplicationDocuments", ctx, appNo, roles).
		Return([]*domain.SourceDocument{
			{DocName: "Өргөдөл", DocCode: "10", DocID: "5001", FileURL: "https://ftp/a.pdf"},
		}, nil)
	landRepo.On("AddSourceParcelDocuments", ctx, parcelID, mock.Anything).Return(1, nil)
	landRepo.On("ListParcelDocuments", ctx, parcelID).Return([]*domain.Document{}, nil)

	res, err := svc.SyncParcelDocuments(ctx, acqID, parcelCode, roles)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Saved)
	parcelInfo.AssertExpectations(t)
}

// app_no байхгүй бол татах боломжгүй — эхлээд нэгж талбараа sync хийнэ.
func TestLandService_SyncParcelDocuments_RequiresAppNo(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode := "010101000001"
	svc, landRepo, parcelInfo := syncDocsSvc(t, acqID, parcelID, parcelCode, "  ")

	_, err := svc.SyncParcelDocuments(ctx, acqID, parcelCode, nil)
	assert.Equal(t, domain.ErrParcelAppNoMissing, err)

	parcelInfo.AssertNotCalled(t, "ListApplicationDocuments", mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "AddSourceParcelDocuments", mock.Anything, mock.Anything, mock.Anything)
}

// Эх системд хавсралт байхгүй бол хадгалагдсан хавсралт УСТГАГДАХГҮЙ.
func TestLandService_SyncParcelDocuments_EmptySourceKeepsStored(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "01107-10-00109-26"
	svc, landRepo, parcelInfo := syncDocsSvc(t, acqID, parcelID, parcelCode, appNo)

	parcelInfo.On("ListApplicationDocuments", ctx, appNo, []string(nil)).Return(nil, nil)
	landRepo.On("AddSourceParcelDocuments", ctx, parcelID, []*domain.Document{}).Return(0, nil)

	stored := []*domain.Document{{ID: uuid.New(), Name: "Гараар оруулсан"}}
	landRepo.On("ListParcelDocuments", ctx, parcelID).Return(stored, nil)

	res, err := svc.SyncParcelDocuments(ctx, acqID, parcelCode, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Found)
	assert.Equal(t, 0, res.Saved)
	assert.Equal(t, stored, res.Documents, "хадгалагдсан хавсралт хэвээр")
}

// Дундын сервис тохируулагдаагүй бол алдаа.
func TestLandService_SyncParcelDocuments_NoClient(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	_, err := svc.SyncParcelDocuments(ctx, uuid.New(), "010101000001", nil)
	assert.Equal(t, domain.ErrInternalServer, err)
}

// ── Хавсралтын файл нээх ───────────────────────────────────────

// stubSourceFiles — эх системийн файл серверийн орлуулагч.
type stubSourceFiles struct {
	gotFtpID, gotPath string
	file              *service.SourceFile
	err               error
}

func (s *stubSourceFiles) Open(_ context.Context, ftpID, filePath string) (*service.SourceFile, error) {
	s.gotFtpID, s.gotPath = ftpID, filePath
	return s.file, s.err
}

func docFileSvc(t *testing.T, files service.SourceFileClient) (
	service.LandService, *mocks.MockLandAcquisitionRepository,
) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	return service.NewLandService(landRepo, nil, nil, parcelInfo, files), landRepo
}

// Эх системийн файл нээгдэж, ftp_id болон зам зөв дамжина.
func TestLandService_OpenParcelDocumentFile_StreamsFromSource(t *testing.T) {
	parcelID, docID := uuid.New(), uuid.New()
	files := &stubSourceFiles{file: &service.SourceFile{
		ContentType: "application/pdf",
		SizeBytes:   1234,
		Content:     io.NopCloser(strings.NewReader("%PDF")),
	}}
	svc, landRepo := docFileSvc(t, files)

	landRepo.On("GetParcelDocument", ctx, parcelID, docID).Return(&domain.Document{
		ID: docID, Name: "Өргөдөл", FileType: "pdf",
		FileURL:     "mgis/011/01107/1681600281/01107-10-00109-26/a-6a7e697d329b7.pdf",
		SourceDocID: "5001", SourceFtpID: "3",
	}, nil)

	file, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	require.NoError(t, err)
	defer file.Content.Close()

	assert.Equal(t, "3", files.gotFtpID)
	assert.Equal(t, "mgis/011/01107/1681600281/01107-10-00109-26/a-6a7e697d329b7.pdf", files.gotPath)
	assert.Equal(t, "application/pdf", file.ContentType)
	assert.Equal(t, int64(1234), file.SizeBytes)
	assert.Equal(t, "Өргөдөл.pdf", file.Name, "hash биш, ойлгомжтой нэрээр татагдана")
}

// Гараар оруулсан (эх системийнх биш) хавсралт энэ замаар нээгдэхгүй —
// тэдгээр нь MinIO дээр байдаг.
func TestLandService_OpenParcelDocumentFile_RejectsNonSourceDoc(t *testing.T) {
	parcelID, docID := uuid.New(), uuid.New()
	files := &stubSourceFiles{}
	svc, landRepo := docFileSvc(t, files)

	landRepo.On("GetParcelDocument", ctx, parcelID, docID).
		Return(&domain.Document{ID: docID, Name: "Гараар оруулсан"}, nil)

	_, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.Empty(t, files.gotPath, "файл сервер рүү хандахгүй")
}

// Өөр нэгж талбарын баримт руу хандах боломжгүй (repo нь холбоог шалгана).
func TestLandService_OpenParcelDocumentFile_NotLinkedToParcel(t *testing.T) {
	parcelID, docID := uuid.New(), uuid.New()
	svc, landRepo := docFileSvc(t, &stubSourceFiles{})

	landRepo.On("GetParcelDocument", ctx, parcelID, docID).Return(nil, domain.ErrNotFound)

	_, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	assert.Equal(t, domain.ErrNotFound, err)
}

// .env тохируулаагүй бол тодорхой алдаа.
func TestLandService_OpenParcelDocumentFile_NotConfigured(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	_, err := svc.OpenParcelDocumentFile(ctx, uuid.New(), uuid.New())
	assert.Equal(t, domain.ErrSourceFileNotConfigured, err)
}
