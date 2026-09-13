package service_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValuationReportUploadPreservesArchivedFile(t *testing.T) {
	ctx := context.Background()
	parcelID := uuid.New()
	repo := new(mocks.MockLandAcquisitionRepository)
	repo.On("GetDocumentType", mock.Anything, 1).Return(&domain.DocumentType{ID: 1, Type: "valuation_report"}, nil)
	repo.On("GetParcelByID", mock.Anything, parcelID).Return(&domain.Parcel{ID: parcelID, ParcelID: "123456"}, nil)
	// Cancellation removed the live document, but its stored object still exists.
	repo.On("ListParcelDocuments", mock.Anything, parcelID).Return([]*domain.Document{}, nil)
	repo.On("AddParcelDocument", mock.Anything, parcelID, mock.Anything).Return(nil)
	files := filestorage.NewMockStorage("/api/files", t.TempDir())
	svc := service.NewLandService(repo, files, nil)
	first, err := svc.UploadParcelDocument(ctx, parcelID, "report.pdf", "Report", "application/pdf", []byte("original"), "tester", 1)
	require.NoError(t, err)
	second, err := svc.UploadParcelDocument(ctx, parcelID, "report.pdf", "Report", "application/pdf", []byte("replacement"), "tester", 1)
	require.NoError(t, err)
	require.NotEqual(t, first.FileURL, second.FileURL)
	require.NoError(t, files.Delete(ctx, second.FileURL))
	content, err := files.Read(ctx, first.FileURL)
	require.NoError(t, err)
	require.Equal(t, "original", string(content))
	repo.AssertExpectations(t)
}
