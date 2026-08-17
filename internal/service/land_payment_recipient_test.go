package service_test

// Итгэмжлэгдсэн төлөөлөгч = эзэмшигчийн нэг төрөл, мөн нөхөн төлбөр авагчийн
// сонголт.
//
// Гол шалгах зүйл: URL дахь нэгж талбарын дугаар нь ЗААСАН чөлөөлөлтөд
// харьяалагдаж байгааг шалгах — эс бөгөөс өөр чөлөөлөлтийн эзэмшигчийг
// өөрчлөх боломж үүснэ.

import (
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func payeeSvc(t *testing.T) (service.LandService, *mocks.MockLandAcquisitionRepository) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	return service.NewLandService(landRepo, nil, nil, nil), landRepo
}

const payeeParcelCode = "010101000001"

func TestLandService_SetParcelPaymentRecipient_Success(t *testing.T) {
	acqID, parcelID, rowID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("SetParcelPaymentRecipient", ctx, parcelID, rowID).Return(nil)

	err := svc.SetParcelPaymentRecipient(ctx, acqID, payeeParcelCode, rowID)

	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

// Нэгж талбар ӨӨР чөлөөлөлтийнх бол өөрчлөхийг зөвшөөрөхгүй.
func TestLandService_SetParcelPaymentRecipient_WrongAcquisition(t *testing.T) {
	acqID, otherAcqID, parcelID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: otherAcqID}, nil)

	err := svc.SetParcelPaymentRecipient(ctx, acqID, payeeParcelCode, uuid.New())

	assert.ErrorIs(t, err, domain.ErrNotFound)
	landRepo.AssertNotCalled(t, "SetParcelPaymentRecipient", mock.Anything, mock.Anything, mock.Anything)
}

func TestLandService_SetParcelPaymentRecipient_UnknownHolderRow(t *testing.T) {
	acqID, parcelID, rowID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("SetParcelPaymentRecipient", ctx, parcelID, rowID).Return(domain.ErrNotFound)

	err := svc.SetParcelPaymentRecipient(ctx, acqID, payeeParcelCode, rowID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Төлөөлөгч бүртгэхэд ШУУД төлбөр авагч болно — repo давхаргад тэмдэглэгдэх ба
// service нь буцаахдаа тухайн мөрийг олж өгнө.
func TestLandService_CreateParcelRepresentative_BecomesPayee(t *testing.T) {
	acqID, parcelID, rowID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	input := &domain.ParcelHolder{LastName: "Бат", Name: "Дорж", RegisterNo: "УБ12345678"}
	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("AddParcelRepresentative", ctx, parcelID, acqID, input).Return(rowID, nil)
	landRepo.On("ListParcelHolders", ctx, parcelID).Return([]domain.ParcelHolder{
		{ID: uuid.New(), Name: "Өөр хүн", HolderRole: domain.ParcelHolderRoleHolder, MainApplicant: true},
		{
			ID: rowID, LastName: "Бат", Name: "Дорж",
			HolderRole: domain.ParcelHolderRoleRepresentative, PaymentRecipient: true,
		},
	}, nil)

	created, err := svc.CreateParcelRepresentative(ctx, acqID, payeeParcelCode, input)

	require.NoError(t, err)
	assert.Equal(t, rowID, created.ID)
	assert.True(t, created.PaymentRecipient)
	assert.Equal(t, domain.ParcelHolderRoleRepresentative, created.HolderRole)
	landRepo.AssertExpectations(t)
}

// Жагсаалт нь ЗӨВХӨН төлөөлөгчийг буцаана — ГУС-аас татагдсан эзэмшигч орохгүй.
func TestLandService_ListParcelRepresentatives_FiltersHolders(t *testing.T) {
	acqID, parcelID, repID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("ListParcelHolders", ctx, parcelID).Return([]domain.ParcelHolder{
		{ID: uuid.New(), Name: "Үндсэн", HolderRole: domain.ParcelHolderRoleHolder, MainApplicant: true},
		{ID: uuid.New(), Name: "Хамтран", HolderRole: domain.ParcelHolderRoleHolder},
		{ID: repID, Name: "Төлөөлөгч", HolderRole: domain.ParcelHolderRoleRepresentative},
	}, nil)

	reps, err := svc.ListParcelRepresentatives(ctx, acqID, payeeParcelCode)

	require.NoError(t, err)
	require.Len(t, reps, 1)
	assert.Equal(t, repID, reps[0].ID)
}

// ГУС-аас татагдсан эзэмшигчийг устгах оролдлогыг repo давхарга буцаана.
func TestLandService_DeleteParcelRepresentative_NotFound(t *testing.T) {
	acqID, parcelID, rowID := uuid.New(), uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("DeleteParcelHolder", ctx, parcelID, rowID).Return(domain.ErrNotFound)

	err := svc.DeleteParcelRepresentative(ctx, acqID, payeeParcelCode, rowID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLandService_CreateParcelRepresentative_InvalidInput(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	svc, landRepo := payeeSvc(t)

	input := &domain.ParcelHolder{} // нэр/регистргүй
	landRepo.On("GetParcelByCode", ctx, payeeParcelCode).
		Return(&domain.Parcel{ID: parcelID, AcquisitionID: acqID}, nil)
	landRepo.On("AddParcelRepresentative", ctx, parcelID, acqID, input).
		Return(uuid.Nil, domain.ErrInvalidInput)

	created, err := svc.CreateParcelRepresentative(ctx, acqID, payeeParcelCode, input)

	assert.Nil(t, created)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	landRepo.AssertNotCalled(t, "ListParcelHolders", mock.Anything, mock.Anything)
}
