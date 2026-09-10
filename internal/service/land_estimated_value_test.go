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

func TestCalculateEstimatedValue(t *testing.T) {
	fees := []domain.ParcelFeeItem{
		{BaseFeePerM2: 10, LanduseArea: 100, ZoneArea: 200},
		{BaseFeePerM2: 20, Area: 100},
	}
	for _, tt := range []struct {
		name                string
		fees                []domain.ParcelFeeItem
		area, percent, want float64
		invalid             bool
	}{
		{"multiple zones use one percentage and affected area", fees, 200, 2, 125000, false},
		{"doubling percentage halves estimate", fees, 200, 4, 62500, false},
		{"round unit price before total", []domain.ParcelFeeItem{{BaseFeePerM2: 1, Area: 400}}, 200, 3, 6600, false},
		{"no fees", nil, 200, 2, 0, true},
		{"missing affected area does not use parcel area", fees, 0, 2, 0, true},
		{"zero percentage", fees, 200, 0, 0, true},
		{"negative percentage", fees, 200, -1, 0, true},
		{"infinite percentage", fees, 200, math.Inf(1), 0, true},
		{"NaN percentage", fees, 200, math.NaN(), 0, true},
		{"overflow", fees, 200, math.SmallestNonzeroFloat64, 0, true},
		{"missing base fee", []domain.ParcelFeeItem{{Area: 100}}, 200, 2, 0, true},
		{"missing zone area", []domain.ParcelFeeItem{{BaseFeePerM2: 10}}, 200, 2, 0, true},
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
		Parcel: domain.Parcel{AcquisitionID: acqID, AreaM2: 1000, AcquisitionAreaM2: 200},
	}, nil)
	repo.On("ListParcelFeeItems", mock.Anything, parcelID).Return([]domain.ParcelFeeItem{{BaseFeePerM2: 10, Area: 1000, ConfidencePercent: 1}}, nil)
	value, percent := 100000.0, 2.0
	repo.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, &value, &percent, (*float64)(nil), "user").Return(nil)
	got, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, percent, nil, "user")
	require.NoError(t, err)
	require.Equal(t, value, got)
	repo.AssertExpectations(t)
}

func TestCalculateParcelEstimatedValueRejectsOtherAcquisition(t *testing.T) {
	repo := &mocks.MockLandAcquisitionRepository{}
	svc := NewLandService(repo, nil, nil, nil)
	parcelID := uuid.New()
	repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{Parcel: domain.Parcel{AcquisitionID: uuid.New()}}, nil)
	_, err := svc.CalculateParcelEstimatedValue(context.Background(), uuid.New(), parcelID, 2, nil, "user")
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
			repo.On("GetParcelDetailByID", mock.Anything, parcelID).Return(&domain.ParcelFull{Parcel: domain.Parcel{AcquisitionID: acqID, AcquisitionAreaM2: 200}}, nil)
			var fees []domain.ParcelFeeItem
			if tt.hasFees {
				fees = []domain.ParcelFeeItem{{BaseFeePerM2: 20, Area: 200}}
			}
			repo.On("ListParcelFeeItems", mock.Anything, parcelID).Return(fees, nil)
			value, percent := 100000.0, 2.0
			if !tt.invalid {
				repo.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, &value, &percent, &tt.rate, "user").Return(nil)
			}
			got, err := svc.CalculateParcelEstimatedValue(context.Background(), acqID, parcelID, percent, &tt.rate, "user")
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
