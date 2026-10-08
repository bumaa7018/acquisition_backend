package service

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// rejectionRequirement — «Татгалзсан»-ы өгөгдмөл заавал хавсралт (seed 000047).
func rejectionRequirement(present bool) []domain.ParcelStatusRequirement {
	return []domain.ParcelStatusRequirement{{
		DocumentTypeID: 34, DocumentTypeCode: "rejection_document",
		DocumentTypeName: "Татгалзсан тухай баримт", Present: present,
	}}
}

// memFiles — Save/Delete-ийг санах ойд тэмдэглэх хуурамч хадгалалт.
type memFiles struct {
	filestorage.DisabledStorage
	saved, deleted []string
}

func (m *memFiles) Save(_ context.Context, _, _, filename, contentType string, data []byte, _ bool) (*filestorage.StoredFile, error) {
	url := "/api/files/parcel/" + filename
	m.saved = append(m.saved, url)
	return &filestorage.StoredFile{URL: url, Name: filename, Type: contentType, SizeBytes: int64(len(data))}, nil
}

func (m *memFiles) Delete(_ context.Context, fileURL string) error {
	m.deleted = append(m.deleted, fileURL)
	return nil
}

// "Татгалзсан" (4) нь нэгж талбарыг чөлөөлөлтөөс ГАРГАДАГ эцсийн шийдвэр тул
// гэрчлэх баримт ЗААВАЛ — одоо «Нэгж ажлын урсгал»-ын тохиргоогоор (өгөгдмөл
// нь «Татгалзсан тухай баримт»). Шаардлагыг service барина — handler дээр л
// шалгавал өөр дуудагч (prof маршрут, шууд API) тойрч гарна.
func TestUpdateParcelStatusRequiresAttachmentWhenRejecting(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID:      uuid.New(),
		AcquisitionID: uuid.New(),
		StatusID:      domain.ParcelStatusRejected,
		CreatedBy:     "user",
		Reason:        "Иргэн татгалзсан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, ch.StatusID).Return(rejectionRequirement(false), nil)

	err := svc.UpdateParcelStatus(context.Background(), ch, nil)

	var missing *domain.RequiredDocumentsMissingError
	require.ErrorAs(t, err, &missing)
	require.Equal(t, []string{"Татгалзсан тухай баримт"}, missing.Names)
	// Төлөв ОГТ солигдох ёсгүй — хавсралтгүй татгалзал бүртгэгдэхгүй.
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Шаардлагатай хавсралт АЛЬ ХЭДИЙН орсон бол файлгүйгээр шилжинэ.
func TestUpdateParcelStatusRequirementAlreadyPresent(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID: uuid.New(), AcquisitionID: uuid.New(),
		StatusID: domain.ParcelStatusRejected, CreatedBy: "user", Reason: "шалтгаан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, ch.StatusID).Return(rejectionRequirement(true), nil)
	repo.On("UpdateParcelStatus", mock.Anything, ch.ParcelID, ch.AcquisitionID, ch.StatusID, "user", "шалтгаан").Return(nil)

	require.NoError(t, svc.UpdateParcelStatus(context.Background(), ch, nil))
	repo.AssertExpectations(t)
}

// Тохируулсан шаардлагагүй төлөвт хавсралт ЗААВАЛ БИШ — өмнөх зан төлөв хэвээр.
func TestUpdateParcelStatusWithoutAttachmentForOtherStatuses(t *testing.T) {
	for _, statusID := range []int{domain.ParcelStatusAffectedOut, 2, 5} {
		repo := &mocks.MockLandAcquisitionRepository{}
		svc := NewLandService(repo, nil, nil, nil)
		ch := domain.ParcelStatusChange{
			ParcelID:      uuid.New(),
			AcquisitionID: uuid.New(),
			StatusID:      statusID,
			CreatedBy:     "user",
			Reason:        "шалтгаан",
		}
		repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
		repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, statusID).Return([]domain.ParcelStatusRequirement{}, nil)
		repo.On("UpdateParcelStatus", mock.Anything, ch.ParcelID, ch.AcquisitionID, statusID, "user", "шалтгаан").Return(nil)

		require.NoError(t, svc.UpdateParcelStatus(context.Background(), ch, nil))
		repo.AssertExpectations(t)
	}
}

// Хадгалалт (MinIO) тохируулаагүй үед хавсралттай шилжилт ЧИМЭЭГҮЙ өнгөрч
// БОЛОХГҮЙ: файлгүй татгалзал бүртгэгдвэл баримт нь мөнхөд алга болно.
func TestUpdateParcelStatusAttachmentNeedsFileStorage(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID:      uuid.New(),
		AcquisitionID: uuid.New(),
		StatusID:      domain.ParcelStatusRejected,
		CreatedBy:     "user",
		Reason:        "Иргэн татгалзсан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, ch.StatusID).Return(rejectionRequirement(false), nil)
	repo.On("GetDocumentType", mock.Anything, 34).Return(&domain.DocumentType{ID: 34, Type: "rejection_document", Name: "Татгалзсан тухай баримт"}, nil).Maybe()
	repo.On("GetParcelByID", mock.Anything, ch.ParcelID).Return(&domain.Parcel{ID: ch.ParcelID, ParcelID: "010101000001"}, nil).Maybe()
	repo.On("ListParcelDocuments", mock.Anything, ch.ParcelID).Return([]*domain.Document{}, nil).Maybe()

	err := svc.UpdateParcelStatus(context.Background(), ch,
		&ParcelStatusAttachment{Filename: "shiidver.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")})

	// Хадгалалтын давхарга өөрөө "тохируулаагүй" гэж унана — алдааны ТӨРӨЛ
	// биш, ТӨЛӨВ СОЛИГДООГҮЙ гэдэг нь чухал.
	require.Error(t, err)
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Өмнөх клиент «Татгалзсан» үед ТӨРӨЛГҮЙ файл илгээдэг — яг нэг заавал
// хавсралт дутуу бол файл тэр төрлөөр бүртгэгдэж, түүхэнд холбоос үлдэнэ.
func TestUpdateParcelStatusUntypedAttachmentFillsSingleMissingType(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	files := &memFiles{}
	svc := NewLandService(repo, files, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID: uuid.New(), AcquisitionID: uuid.New(),
		StatusID: domain.ParcelStatusRejected, CreatedBy: "user", Reason: "Иргэн татгалзсан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, ch.StatusID).Return(rejectionRequirement(false), nil)
	repo.On("GetDocumentType", mock.Anything, 34).Return(&domain.DocumentType{ID: 34, Type: "rejection_document", Name: "Татгалзсан тухай баримт"}, nil)
	repo.On("GetParcelByID", mock.Anything, ch.ParcelID).Return(&domain.Parcel{ID: ch.ParcelID, ParcelID: "010101000001"}, nil)
	repo.On("ListParcelDocuments", mock.Anything, ch.ParcelID).Return([]*domain.Document{}, nil)
	var stored *domain.Document
	repo.On("AddParcelDocument", mock.Anything, ch.ParcelID, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(2).(*domain.Document)
	}).Return(nil)
	repo.On("UpdateParcelStatus", mock.Anything, ch.ParcelID, ch.AcquisitionID, ch.StatusID, "user", "Иргэн татгалзсан").Return(nil)

	require.NoError(t, svc.UpdateParcelStatus(context.Background(), ch,
		&ParcelStatusAttachment{Filename: "shiidver.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")}))

	require.NotNil(t, stored)
	require.Equal(t, 34, stored.DocumentTypeID, "дутуу ганц төрлөөр бүртгэгдэнэ")
	require.Empty(t, files.deleted, "амжилттай шилжилтийн баримт устахгүй")
	repo.AssertExpectations(t)
}

// «Нөхөх олговор олгосон баримт»-д шат заавал тул төрөлгүй файлыг түүгээр
// автоматаар бүртгэхгүй — дутуу хавсралтын ТОДОРХОЙ алдаа буцна (файл хадгалагдахгүй).
func TestUpdateParcelStatusDoesNotAutoTypePaymentReceipt(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	files := &memFiles{}
	svc := NewLandService(repo, files, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID: uuid.New(), AcquisitionID: uuid.New(),
		StatusID: 5, CreatedBy: "user", Reason: "шалтгаан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, mock.Anything).Return(false, nil)
	repo.On("ParcelStatusRequirements", mock.Anything, ch.AcquisitionID, ch.ParcelID, ch.StatusID).Return([]domain.ParcelStatusRequirement{{
		DocumentTypeID: 162, DocumentTypeCode: domain.CompensationReceiptDocType, DocumentTypeName: "Нөхөх олговор олгосон баримт",
	}}, nil)

	err := svc.UpdateParcelStatus(context.Background(), ch,
		&ParcelStatusAttachment{Filename: "barimt.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")})

	var missing *domain.RequiredDocumentsMissingError
	require.ErrorAs(t, err, &missing)
	require.Equal(t, []string{"Нөхөх олговор олгосон баримт"}, missing.Names)
	require.Empty(t, files.saved, "файл хадгалагдахгүй")
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Одоогийн төлөв «Явц түгжинэ» бол хавсралт шалгахаас ӨМНӨ татгалзана —
// төлөв, хавсралт ОГТ хөндөгдөхгүй.
func TestUpdateParcelStatusRejectedWhenProgressLocked(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	files := &memFiles{}
	svc := NewLandService(repo, files, nil, nil)
	ch := domain.ParcelStatusChange{
		ParcelID: uuid.New(), AcquisitionID: uuid.New(),
		StatusID: domain.ParcelStatusRejected, CreatedBy: "user", Reason: "шалтгаан",
	}
	repo.On("ParcelProgressLocked", mock.Anything, ch.ParcelID).Return(true, nil)

	err := svc.UpdateParcelStatus(context.Background(), ch,
		&ParcelStatusAttachment{Filename: "barimt.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")})

	require.ErrorIs(t, err, domain.ErrParcelProgressLocked)
	require.Empty(t, files.saved, "файл хадгалагдахгүй")
	repo.AssertNotCalled(t, "ParcelStatusRequirements", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
