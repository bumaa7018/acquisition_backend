package service

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

func (s *landService) ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error) {
	return s.landRepo.ListParcelWorkflows(ctx)
}

func (s *landService) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error) {
	return s.landRepo.CreateParcelWorkflow(ctx, fromStatusID, toStatusID, sortOrder)
}

func (s *landService) DeleteParcelWorkflow(ctx context.Context, id int) error {
	return s.landRepo.DeleteParcelWorkflow(ctx, id)
}

func (s *landService) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	return s.landRepo.GetAvailableParcelStatuses(ctx, currentStatusID)
}

func (s *landService) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	return s.landRepo.UpdateParcelStatus(ctx, parcelID, acquisitionID, statusID, createdBy)
}

func (s *landService) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	return s.landRepo.ListParcelStatusHistory(ctx, parcelID)
}

func (s *landService) GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error) {
	return s.landRepo.GetValuationSubmission(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error) {
	return s.landRepo.TransitionValuationSubmission(ctx, acquisitionID, parcelID, valuationType, action, note, userID)
}

func (s *landService) ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error) {
	return s.landRepo.ListValuationSubmissionHistory(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	return s.landRepo.ListFundingSources(ctx, acquisitionID)
}

func (s *landService) CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	if err := s.landRepo.CreateFundingSource(ctx, fs); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *landService) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	if err := s.landRepo.UpdateFundingSource(ctx, fs); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *landService) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteFundingSource(ctx, id)
}

func (s *landService) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	return s.landRepo.ListAuthorizedRepresentatives(ctx, acquisitionID, parcelID)
}

func (s *landService) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error) {
	if err := s.landRepo.CreateAuthorizedRepresentative(ctx, rep); err != nil {
		return nil, err
	}
	return rep, nil
}

func (s *landService) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteAuthorizedRepresentative(ctx, id)
}

func (s *landService) ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error) {
	return s.landRepo.ListAcquisitionWorkflows(ctx)
}

func (s *landService) CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error) {
	return s.landRepo.CreateAcquisitionWorkflow(ctx, fromStatusID, toStatusID, sortOrder)
}

func (s *landService) DeleteAcquisitionWorkflow(ctx context.Context, id int) error {
	return s.landRepo.DeleteAcquisitionWorkflow(ctx, id)
}

func (s *landService) ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error) {
	return s.landRepo.ListAllCompensations(ctx, filter)
}

func (s *landService) GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error) {
	return s.landRepo.GetGlobalCompensation(ctx, id)
}
