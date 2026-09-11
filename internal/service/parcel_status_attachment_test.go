package service

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// "Татгалзсан" (4) нь нэгж талбарыг чөлөөлөлтөөс ГАРГАДАГ эцсийн шийдвэр тул
// гэрчлэх баримт ЗААВАЛ. Шаардлагыг service барина — handler дээр л шалгавал
// өөр дуудагч (prof маршрут, шууд API) тойрч гарна.
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

	err := svc.UpdateParcelStatus(context.Background(), ch, nil)

	require.ErrorIs(t, err, domain.ErrStatusAttachmentRequired)
	// Төлөв ОГТ солигдох ёсгүй — хавсралтгүй татгалзал бүртгэгдэхгүй.
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Бусад төлөвт хавсралт ЗААВАЛ БИШ — өмнөх зан төлөв хэвээр.
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

	err := svc.UpdateParcelStatus(context.Background(), ch,
		&ParcelStatusAttachment{Filename: "shiidver.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")})

	// Хадгалалтын давхарга өөрөө "тохируулаагүй" гэж унана — алдааны ТӨРӨЛ
	// биш, ТӨЛӨВ СОЛИГДООГҮЙ гэдэг нь чухал.
	require.Error(t, err)
	repo.AssertNotCalled(t, "UpdateParcelStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
