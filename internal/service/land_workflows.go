package service

import (
	"context"
	"errors"

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

func (s *landService) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy, reason string) error {
	if err := s.landRepo.UpdateParcelStatus(ctx, parcelID, acquisitionID, statusID, createdBy, reason); err != nil {
		return err
	}
	s.notifyParcelStatusChanged(ctx, acquisitionID, parcelID)
	s.auditLog(ctx, "status_changed", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
		"to_status":  statusID,
		"created_by": createdBy,
		"reason":     reason,
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

// resolveParcelRow — URL дахь нэгж талбарын ДУГААРААС parcel мөрийг олно.
// Тухайн нэгж талбар заасан чөлөөлөлтөд харьяалагдаж байгааг ШАЛГАНА — эс
// бөгөөс өөр чөлөөлөлтийн эзэмшигчийг өөрчлөх боломж үүснэ.
func (s *landService) resolveParcelRow(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.Parcel, error) {
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if p.AcquisitionID != acquisitionID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *landService) ListParcelRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) ([]domain.ParcelHolder, error) {
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return nil, err
	}
	holders, err := s.landRepo.ListParcelHolders(ctx, p.ID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	reps := make([]domain.ParcelHolder, 0, len(holders))
	for _, h := range holders {
		if h.HolderRole == domain.ParcelHolderRoleRepresentative {
			reps = append(reps, h)
		}
	}
	return reps, nil
}

// CreateParcelRepresentative — итгэмжлэгдсэн төлөөлөгчийг эзэмшигчийн төрөл
// болгож бүртгэнэ. Бүртгэмэгц ШУУД нөхөн төлбөр авагч болно.
func (s *landService) CreateParcelRepresentative(
	ctx context.Context,
	acquisitionID uuid.UUID,
	parcelCode string,
	holder *domain.ParcelHolder,
) (*domain.ParcelHolder, error) {
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return nil, err
	}

	rowID, err := s.landRepo.AddParcelRepresentative(ctx, p.ID, acquisitionID, holder)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return nil, domain.ErrInvalidInput
		}
		return nil, domain.ErrInternalServer
	}

	s.auditLog(ctx, "created", domain.AuditResourceRepresentative, rowID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_id":         parcelCode,
			"name":              holder.Name,
			"last_name":         holder.LastName,
			"register_no":       holder.RegisterNo,
			"payment_recipient": true,
		})

	holders, err := s.landRepo.ListParcelHolders(ctx, p.ID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	for i := range holders {
		if holders[i].ID == rowID {
			return &holders[i], nil
		}
	}
	return nil, domain.ErrInternalServer
}

func (s *landService) DeleteParcelRepresentative(
	ctx context.Context,
	acquisitionID uuid.UUID,
	parcelCode string,
	parcelHolderID uuid.UUID,
) error {
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return err
	}
	if err := s.landRepo.DeleteParcelHolder(ctx, p.ID, parcelHolderID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return domain.ErrInternalServer
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceRepresentative, parcelHolderID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{"parcel_id": parcelCode})
	return nil
}

// SetParcelPaymentRecipient — үндсэн эзэмшигч олон байх үед нөхөн төлбөрийг
// хэнд олгохыг сонгоно.
func (s *landService) SetParcelPaymentRecipient(
	ctx context.Context,
	acquisitionID uuid.UUID,
	parcelCode string,
	parcelHolderID uuid.UUID,
) error {
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return err
	}
	if err := s.landRepo.SetParcelPaymentRecipient(ctx, p.ID, parcelHolderID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return domain.ErrInternalServer
	}
	s.auditLog(ctx, "updated", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_id":         parcelCode,
			"payment_recipient": parcelHolderID.String(),
		})
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
