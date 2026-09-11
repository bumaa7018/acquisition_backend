//go:build e2e

package postgres_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func floatPtr(v float64) *float64 { return &v }

func TestE2E_EstimatedValueConfidenceRoundTrip(t *testing.T) {
	f := newFixture(t, 0, 1)
	ctx := context.Background()
	parcelID := f.parcels[1]
	_, err := f.pool.Exec(ctx, "UPDATE parcel SET acquisition_area_m2 = 200 WHERE id = $1", parcelID)
	require.NoError(t, err)
	fees := []domain.ParcelFeeItem{{Area: 500, BaseFeePerM2: 10, ConfidencePercent: 1}}
	_, err = f.repo.ReplaceParcelFeeItems(ctx, parcelID, f.acqID, fees)
	require.NoError(t, err)
	value, err := f.svc.CalculateParcelEstimatedValue(ctx, f.acqID, parcelID, floatPtr(2), nil, nil, "e2e")
	require.NoError(t, err)
	require.Equal(t, 100000.0, value)
	p, err := f.repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.NotNil(t, p.EstimatedValue)
	require.Equal(t, value, *p.EstimatedValue)
	require.NotNil(t, p.EstimatedConfidencePercent)
	require.Equal(t, 2.0, *p.EstimatedConfidencePercent)
	savedFees, err := f.repo.ListParcelFeeItems(ctx, parcelID)
	require.NoError(t, err)
	require.Equal(t, 1.0, savedFees[0].ConfidencePercent)
	// Replacing middleware data must preserve the user-selected percentage.
	_, err = f.repo.ReplaceParcelFeeItems(ctx, parcelID, f.acqID, fees)
	require.NoError(t, err)
	p, err = f.repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.Equal(t, 2.0, *p.EstimatedConfidencePercent)
	require.NoError(t, f.svc.SetParcelEstimatedValue(ctx, f.acqID, parcelID, nil, "e2e"))
	p, err = f.repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.Nil(t, p.EstimatedValue)
	require.Nil(t, p.EstimatedConfidencePercent)
}

func TestE2E_EstimatedValueManualBaseFee(t *testing.T) {
	f := newFixture(t, 0, 1)
	ctx := context.Background()
	parcelID := f.parcels[1]
	_, err := f.pool.Exec(ctx, "UPDATE parcel SET acquisition_area_m2 = 200 WHERE id = $1", parcelID)
	require.NoError(t, err)
	baseFee := 10.0
	value, err := f.svc.CalculateParcelEstimatedValue(ctx, f.acqID, parcelID, floatPtr(2), &baseFee, nil, "e2e")
	require.NoError(t, err)
	require.Equal(t, 100000.0, value)
	p, err := f.repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.NotNil(t, p.EstimatedBaseFeePerM2)
	require.Equal(t, baseFee, *p.EstimatedBaseFeePerM2)
	require.Equal(t, value, *p.EstimatedValue)
	require.Equal(t, 2.0, *p.EstimatedConfidencePercent)
	fees, err := f.repo.ListParcelFeeItems(ctx, parcelID)
	require.NoError(t, err)
	require.Empty(t, fees)
	value, err = f.svc.CalculateParcelEstimatedValue(ctx, f.acqID, parcelID, floatPtr(4), nil, nil, "e2e")
	require.NoError(t, err)
	require.Equal(t, 50000.0, value)
	require.NoError(t, f.svc.SetParcelEstimatedValue(ctx, f.acqID, parcelID, nil, "e2e"))
	p, err = f.repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.Nil(t, p.EstimatedBaseFeePerM2)
	require.Nil(t, p.EstimatedConfidencePercent)
	require.Nil(t, p.EstimatedValue)
}
