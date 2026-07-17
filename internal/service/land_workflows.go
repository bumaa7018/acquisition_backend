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
	if err := s.landRepo.UpdateParcelStatus(ctx, parcelID, acquisitionID, statusID, createdBy); err != nil {
		return err
	}
	s.notifyParcelStatusChanged(ctx, acquisitionID, parcelID)
	s.auditLog(ctx, "status_changed", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
		"to_status":  statusID,
		"created_by": createdBy,
	})
	return nil
}

func (s *landService) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	return s.landRepo.ListParcelStatusHistory(ctx, parcelID)
}

func (s *landService) GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error) {
	return s.landRepo.GetValuationSubmission(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error) {
	sub, err := s.landRepo.TransitionValuationSubmission(ctx, acquisitionID, parcelID, valuationType, action, note, userID)
	if err != nil {
		return nil, err
	}
	s.notifyValuationTransition(ctx, acquisitionID, parcelID, valuationType, action, sub)
	s.auditLog(ctx, "valuation_"+action, domain.AuditResourceValuation, parcelID.String(), uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
		"valuation_type": valuationType,
		"note":           note,
		"user_id":        userID,
		"status":         sub.Status,
	})
	return sub, nil
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
	s.auditLog(ctx, "created", domain.AuditResourceFundingSource, fs.ID.String(), uuidPtr(fs.AcquisitionID), nil, map[string]any{
		"organization_name": fs.OrganizationName,
		"source_type":       fs.SourceType,
		"amount":            fs.Amount,
	})
	return fs, nil
}

func (s *landService) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	existing, err := s.landRepo.GetFundingSource(ctx, fs.ID)
	if err != nil {
		return nil, err
	}
	fs.AcquisitionID = existing.AcquisitionID
	if err := s.landRepo.UpdateFundingSource(ctx, fs); err != nil {
		return nil, err
	}
	s.auditLog(ctx, "updated", domain.AuditResourceFundingSource, fs.ID.String(), uuidPtr(fs.AcquisitionID), nil, map[string]any{
		"organization_name": fs.OrganizationName,
		"source_type":       fs.SourceType,
		"amount":            fs.Amount,
	})
	return fs, nil
}

func (s *landService) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	existing, err := s.landRepo.GetFundingSource(ctx, id)
	if err != nil {
		return err
	}
	if err := s.landRepo.DeleteFundingSource(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceFundingSource, id.String(), uuidPtr(existing.AcquisitionID), nil, nil)
	return nil
}

func (s *landService) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	return s.landRepo.ListAuthorizedRepresentatives(ctx, acquisitionID, parcelID)
}

func (s *landService) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error) {
	if err := s.landRepo.CreateAuthorizedRepresentative(ctx, rep); err != nil {
		return nil, err
	}
	s.auditLog(ctx, "created", domain.AuditResourceRepresentative, rep.ID.String(), uuidPtr(rep.AcquisitionID), nil, map[string]any{
		"parcel_id":   rep.ParcelID,
		"first_name":  rep.FirstName,
		"last_name":   rep.LastName,
		"register_no": rep.RegisterNo,
	})
	return rep, nil
}

func (s *landService) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	if err := s.landRepo.DeleteAuthorizedRepresentative(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceRepresentative, id.String(), nil, nil, nil)
	return nil
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
