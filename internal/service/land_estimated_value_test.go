package service

import (
	"context"
	"math"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func ptr(v float64) *float64 { return &v }

func TestCalculateEstimatedValue(t *testing.T) {
	fees := []domain.ParcelFeeItem{
		{BaseFeePerM2: 10, LanduseArea: 100, ZoneArea: 200},
		{BaseFeePerM2: 20, Area: 100},
	}
	for _, tt := range []struct {
		name    string
		fees    []domain.ParcelFeeItem
		area    float64
		percent *float64
		want    float64
		invalid bool
	}{
		{name: "multiple zones use one percentage and affected area", fees: fees, area: 200, percent: ptr(2), want: 125000},
		{name: "doubling percentage halves estimate", fees: fees, area: 200, percent: ptr(4), want: 62500},
		{name: "round unit price before total", fees: []domain.ParcelFeeItem{{BaseFeePerM2: 1, Area: 400}}, area: 200, percent: ptr(3), want: 6600},
		// ИТГЭЛЦҮҮРГҮЙ (өмчлөх эрхийн газар): суурь үнэ нь зах зээлийн жишиг үнэ
		// тул хувиргалтгүйгээр талбайгаар үржигдэнэ.
		{name: "no percentage multiplies market price by area", fees: fees, area: 200, percent: nil, want: 2500},
		{name: "no percentage with one zone", fees: []domain.ParcelFeeItem{{BaseFeePerM2: 1500, Area: 400}}, area: 200, percent: nil, want: 300000},
		{name: "no fees", fees: nil, area: 200, percent: ptr(2), invalid: true},
		{name: "missing affected area does not use parcel area", fees: fees, area: 0, percent: ptr(2), invalid: true},
		{name: "zero percentage", fees: fees, area: 200, percent: ptr(0), invalid: true},
		{name: "negative percentage", fees: fees, area: 200, percent: ptr(-1), invalid: true},
		{name: "infinite percentage", fees: fees, area: 200, percent: ptr(math.Inf(1)), invalid: true},
		{name: "NaN percentage", fees: fees, area: 200, percent: ptr(math.NaN()), invalid: true},
		{name: "overflow", fees: fees, area: 200, percent: ptr(math.SmallestNonzeroFloat64), invalid: true},
		{name: "missing base fee", fees: []domain.ParcelFeeItem{{Area: 100}}, area: 200, percent: ptr(2), invalid: true},
		{name: "missing zone area", fees: []domain.ParcelFeeItem{{BaseFeePerM2: 10}}, area: 200, percent: ptr(2), invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			value, err := calculateEstimatedValue(tt.fees, tt.area, tt.percent)
			if tt.invalid {
				require.ErrorIs(t, err, domain.ErrInvalidInput)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, value)
		})
	}
}

func TestCalculateParcelEstimatedValueSavesAmountAndPercentage(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	acqID, parcelID := uuid.New(), uuid.New()
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{
		Parcel: domain.Parcel{AcquisitionID: acqID, AreaM2: 1000, AcquisitionAreaM2: 200, RightType: domain.RightTypeLease},
	}, nil)
	repo.On("ListParcelFeeItems", mock.Anything, parcelID).Return([]domain.ParcelFeeItem{{BaseFeePerM2: 10, Area: 1000, ConfidencePercent: 1}}, nil)
	value, percent := 100000.0, 2.0
	repo.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, &value, &percent, (*float64)(nil), "user").Return(nil)
	got, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, &percent, nil, "user")
	require.NoError(t, err)
	require.Equal(t, value, got)
	repo.AssertExpectations(t)
}

// ӨМЧЛӨХ эрхийн газарт итгэлцүүр ХЭРЭГЛЭХГҮЙ — client илгээсэн ч алгасагдаж,
// хадгалагдахдаа ч NULL болно (дараа нээхэд итгэлцүүр байсан гэж уншигдахгүй).
func TestCalculateParcelEstimatedValueOwnershipIgnoresConfidence(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	acqID, parcelID := uuid.New(), uuid.New()
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{
		Parcel: domain.Parcel{AcquisitionID: acqID, AreaM2: 1000, AcquisitionAreaM2: 200, RightType: domain.RightTypeOwnership},
	}, nil)
	repo.On("ListParcelFeeItems", mock.Anything, parcelID).Return([]domain.ParcelFeeItem(nil), nil)
	// Зах зээлийн жишиг үнэ 1500₮/м² × 200 м² = 300 000₮ (итгэлцүүр 2%-ыг
	// хэрэглэсэн бол 15 000 000₮ болох байсан).
	rate, value := 1500.0, 300000.0
	repo.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, &value, (*float64)(nil), &rate, "user").Return(nil)
	got, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, ptr(2), &rate, "user")
	require.NoError(t, err)
	require.Equal(t, value, got)
	repo.AssertExpectations(t)
}

// Өмчлөхөөс БУСАД эрхийн төрөлд итгэлцүүр ЗААВАЛ — байхгүй бол тооцохгүй
// (эс бөгөөс суурь төлбөрөө шууд зах зээлийн үнэ гэж авч, үнэлгээ дордоно).
func TestCalculateParcelEstimatedValueRequiresConfidenceForNonOwnership(t *testing.T) {
	for _, rightType := range []int{domain.RightTypeUse, domain.RightTypeLease} {
		repo := &mocks.MockLandAcquisitionRepository{}
		svc := NewLandService(repo, nil, nil, nil)
		acqID, parcelID := uuid.New(), uuid.New()
		repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{
			Parcel: domain.Parcel{AcquisitionID: acqID, AcquisitionAreaM2: 200, RightType: rightType},
		}, nil)
		_, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, nil, ptr(10), "user")
		require.ErrorIs(t, err, domain.ErrInvalidInput)
		repo.AssertNotCalled(t, "SetParcelEstimatedValue", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
}

func TestCalculateParcelEstimatedValueRejectsOtherAcquisition(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	parcelID := uuid.New()
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{Parcel: domain.Parcel{AcquisitionID: uuid.New()}}, nil)
	_, err := svc.CalculateParcelEstimatedValue(context.Background(), uuid.New(), parcelID, ptr(2), nil, "user")
	require.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "ListParcelFeeItems", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SetParcelEstimatedValue", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCalculateParcelEstimatedValueManualFee(t *testing.T) {
	for _, tt := range []struct {
		name             string
		rate             float64
		hasFees, invalid bool
	}{
		{"manual fee", 10, false, false},
		{"zero fee", 0, false, true},
		{"negative fee", -10, false, true},
		{"infinite fee", math.Inf(1), false, true},
		{"cannot override saved fees", 10, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockLandAcquisitionRepository{}
			svc := NewLandService(repo, nil, nil, nil)
			acqID, parcelID := uuid.New(), uuid.New()
			repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{Parcel: domain.Parcel{AcquisitionID: acqID, AcquisitionAreaM2: 200, RightType: domain.RightTypeLease}}, nil)
			var fees []domain.ParcelFeeItem
			if tt.hasFees {
				fees = []domain.ParcelFeeItem{{BaseFeePerM2: 20, Area: 200}}
			}
			repo.On("ListParcelFeeItems", mock.Anything, parcelID).Return(fees, nil)
			value, percent := 100000.0, 2.0
			if !tt.invalid {
				repo.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, &value, &percent, &tt.rate, "user").Return(nil)
			}
			got, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, &percent, &tt.rate, "user")
			if tt.invalid {
				require.ErrorIs(t, err, domain.ErrInvalidInput)
			} else {
				require.NoError(t, err)
				require.Equal(t, value, got)
			}
			repo.AssertExpectations(t)
		})
	}
}
