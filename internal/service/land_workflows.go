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

func (s *landService) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int, documentTypeIDs []int) (*domain.ParcelWorkflow, error) {
	w, err := s.landRepo.CreateParcelWorkflow(ctx, fromStatusID, toStatusID, sortOrder)
	if err != nil || len(documentTypeIDs) == 0 {
		return w, err
	}
	// Хавсралт буруу бол (төрөл байхгүй г.м.) шилжилтийг ч үлдээхгүй.
	if err := s.landRepo.SetParcelWorkflowRequiredDocuments(ctx, w.ID, documentTypeIDs); err != nil {
		_ = s.landRepo.DeleteParcelWorkflow(ctx, w.ID)
		return nil, err
	}
	all, err := s.landRepo.ListParcelWorkflows(ctx)
	if err != nil {
		return w, nil
	}
	for _, x := range all {
		if x.ID == w.ID {
			return x, nil
		}
	}
	return w, nil
}

func (s *landService) DeleteParcelWorkflow(ctx context.Context, id int) error {
	return s.landRepo.DeleteParcelWorkflow(ctx, id)
}

func (s *landService) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	return s.landRepo.GetAvailableParcelStatuses(ctx, currentStatusID)
}

func (s *landService) SetParcelWorkflowRequiredDocuments(ctx context.Context, workflowID int, documentTypeIDs []int) error {
	return s.landRepo.SetParcelWorkflowRequiredDocuments(ctx, workflowID, documentTypeIDs)
}

func (s *landService) ParcelStatusRequirements(ctx context.Context, acquisitionID, parcelID uuid.UUID, statusID int) ([]domain.ParcelStatusRequirement, error) {
	return s.landRepo.ParcelStatusRequirements(ctx, acquisitionID, parcelID, statusID)
}

// UpdateParcelStatus — нэгж талбарын төлөв солих.
//
// ЗААВАЛ ХАВСРАЛТ: «Нэгж ажлын урсгал»-д тухайн ШИЛЖИЛТЭД (одоогийн төлөв →
// очих төлөв) тохируулсан хавсралтын төрөл бүр нэгж талбарт (parcel_document)
// байх ёстой — өгөгдмөл нь «… → Татгалзсан» → «Татгалзсан тухай баримт»,
// «… → Чөлөөлсөн» → «Үнэлгээний тайлан» (seed 000047). Шаардлагыг ЭНД (service) шалгана: handler дээр л шалгавал өөр
// дуудагч (prof маршрут) тойрч гарна.
//
// attachment нь дутуу хавсралтыг НӨХӨЖ болно: төрөл заагаагүй бөгөөд яг НЭГ
// төрөл дутуу бол тэр төрлөөр бүртгэгдэнэ (өмнөх клиент «Татгалзсан» үед
// төрөлгүй файл илгээдэг байсантай нийцнэ).
//
// Файл нь НЭГЖ ТАЛБАРЫН БАРИМТ болж бүртгэгдэнэ (parcel_document) — ингэснээр
// "Хавсралт" хэсэгт бусад баримтын хамт харагдана; түүхэн дээр зөвхөн
// холбоос үлдэнэ. Төлөв солих алхам унавал баримт нь орфон болж үлдэхгүйн
// тулд цэвэрлэгдэнэ.
func (s *landService) UpdateParcelStatus(
	ctx context.Context, ch domain.ParcelStatusChange, attachment *ParcelStatusAttachment,
) error {
	reqs, err := s.landRepo.ParcelStatusRequirements(ctx, ch.AcquisitionID, ch.ParcelID, ch.StatusID)
	if err != nil {
		return err
	}
	var missing []domain.ParcelStatusRequirement
	for _, r := range reqs {
		if !r.Present {
			missing = append(missing, r)
		}
	}
	// «Нөхөх олговор олгосон баримт»-д олголтын шат (60/40/бүрэн) заавал тул
	// төрөлгүй файлыг түүгээр автоматаар бүртгэхгүй — хавсралтын хэсгээс оруулна.
	if attachment != nil && attachment.DocumentTypeID == 0 && len(missing) == 1 &&
		missing[0].DocumentTypeCode != domain.CompensationReceiptDocType {
		attachment.DocumentTypeID = missing[0].DocumentTypeID
	}
	var stillMissing []string
	for _, r := range missing {
		if attachment == nil || attachment.DocumentTypeID != r.DocumentTypeID {
			stillMissing = append(stillMissing, r.DocumentTypeName)
		}
	}
	if len(stillMissing) > 0 {
		return &domain.RequiredDocumentsMissingError{Names: stillMissing}
	}
	// Тусдаа файл хавсаргаагүй бол түүхэнд ЗААВАЛ хавсралтын холбоосыг
	// үлдээнэ — «Татгалзсан» шилжилт өмнө нь түүхэн дээрээ баримттай байдаг
	// байсан (одоо баримтыг цонхноос урьдчилан оруулдаг).
	if attachment == nil {
		for _, r := range reqs {
			if r.Present && r.DocumentURL != "" {
				ch.AttachmentURL, ch.AttachmentName = r.DocumentURL, r.DocumentName
				break
			}
		}
	}
	if attachment != nil {
		if s.files == nil {
			return domain.ErrInternalServer
		}
		// ХАРАГДАХ нэр нь хэрэглэгчийн засварласан нэр; хоосон бол файлын нэр.
		// Физик файл нь ҮРГЭЛЖ анхны нэрээрээ (өргөтгөлтэй) хадгалагдана —
		// өргөтгөлгүй нэр тавьбал браузер файлыг зөв нээхээ болино.
		doc, err := s.UploadParcelDocument(ctx, ch.ParcelID,
			attachment.Filename, attachment.DisplayName, attachment.ContentType, attachment.Data, ch.CreatedBy, attachment.DocumentTypeID)
		if err != nil {
			return err
		}
		ch.AttachmentURL, ch.AttachmentName = doc.FileURL, doc.Name
		defer func() {
			if !attachment.committed {
				_ = s.DeleteParcelDocument(ctx, ch.ParcelID, doc.ID)
			}
		}()
	}
	if err := s.landRepo.UpdateParcelStatus(ctx, ch); err != nil {
		return err
	}
	if attachment != nil {
		attachment.committed = true
	}
	s.notifyParcelStatusChanged(ctx, ch.AcquisitionID, ch.ParcelID)
	s.auditLog(ctx, "status_changed", domain.AuditResourceParcel, ch.ParcelID.String(), uuidPtr(ch.AcquisitionID), uuidPtr(ch.ParcelID), map[string]any{
		"to_status":       ch.StatusID,
		"created_by":      ch.CreatedBy,
		"reason":          ch.Reason,
		"attachment_name": ch.AttachmentName,
	})
	return nil
}

// ParcelStatusAttachment — төлөв солиход хавсаргах файл (тохируулсан заавал
// хавсралтыг нөхөж болно).
//
// `committed` нь дотоод тэмдэглэгээ: төлөв амжилттай солигдсоны дараа л true
// болно. Үүнгүй бол шилжилт унасан үед бүртгэгдсэн баримт үлдэнэ.
type ParcelStatusAttachment struct {
	Filename string
	// DisplayName — жагсаалт/түүхэнд ХАРАГДАХ нэр (хэрэглэгч гараас засна).
	// Хоосон бол файлын нэр хэрэглэгдэнэ (documentDisplayName).
	DisplayName string
	ContentType string
	Data        []byte
	// DocumentTypeID — баримтын төрөл (0 = заагаагүй; яг нэг заавал хавсралт
	// дутуу бол тэр төрлийг авна).
	DocumentTypeID int

	committed bool
}

func (s *landService) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	return s.landRepo.ListParcelStatusHistory(ctx, parcelID)
}

func (s *landService) GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error) {
	return s.landRepo.GetValuationSubmission(ctx, acquisitionID, parcelID, valuationType)
}

// TransitionValuationSubmission — үнэлгээний илгээх/зөвшөөрөх/буцаах шилжилт.
//
// ХАВСРАЛТ (attachment) нь БУЦААХ үед сонголттой, ЦУЦЛАХ үед заавал.
// Файл нь ЭХЛЭЭД хадгалалт (MinIO) руу бичигдэж, дараа нь шилжилт хийгдэнэ;
// шилжилт унавал орфон файл үлдэхгүйн тулд файлыг цэвэрлэнэ.
func (s *landService) TransitionValuationSubmission(
	ctx context.Context, in domain.ValuationTransition, attachment *ValuationAttachment,
) (*domain.ValuationSubmission, error) {
	if in.Action == "cancel" && attachment == nil {
		return nil, domain.ErrValuationCancelAttachmentRequired
	}
	// ИЛГЭЭХ үед хавсралт АВАХГҮЙ: баталгаажсан тайлан ба ажлын зураг хоёр нь
	// нэгж талбарын БАРИМТ (parcel_document) болж тусдаа хадгалагдана. Илгээх
	// үед тэдгээр байгаа эсэхийг репозитори шалгана
	// (ErrSubmitReportRequired / ErrSubmitWorkPhotoRequired).
	if in.Action == "submit" {
		attachment = nil
	}
	if attachment != nil && (in.Action == "return" || in.Action == "cancel") {
		if s.files == nil {
			return nil, domain.ErrInternalServer
		}
		bucket := "valuation-return"
		if in.Action == "cancel" {
			bucket = "valuation-cancel"
		}
		stored, err := s.files.Save(ctx, bucket, in.ParcelID.String(),
			attachment.Filename, attachment.ContentType, attachment.Data, true)
		if err != nil {
			return nil, err
		}
		in.AttachmentURL, in.AttachmentName = stored.URL, stored.Name
		defer func() {
			// Шилжилт унасан бол хадгалсан файлыг цэвэрлэнэ.
			if in.AttachmentURL != "" && !attachment.committed {
				_ = s.files.Delete(ctx, in.AttachmentURL)
			}
		}()
	}

	sub, err := s.landRepo.TransitionValuationSubmission(ctx, in)
	if err != nil {
		return nil, err
	}
	if attachment != nil {
		attachment.committed = true
	}
	s.notifyValuationTransition(ctx, in.AcquisitionID, in.ParcelID, in.ValuationType, in.Action, sub)
	s.auditLog(ctx, "valuation_"+in.Action, domain.AuditResourceValuation, in.ParcelID.String(),
		uuidPtr(in.AcquisitionID), uuidPtr(in.ParcelID), map[string]any{
			"valuation_type":  in.ValuationType,
			"note":            in.Note,
			"user_id":         in.UserID,
			"status":          sub.Status,
			"attachment_name": sub.AttachmentName,
		})
	return sub, nil
}

func (s *landService) ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error) {
	return s.landRepo.ListValuationSubmissionHistory(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) ListValuationSnapshots(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSnapshot, error) {
	return s.landRepo.ListValuationSnapshots(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	return s.landRepo.ListFundingSources(ctx, acquisitionID)
}

func (s *landService) GetFinanceDashboard(
	ctx context.Context, filter domain.FinanceDashboardFilter,
) (*domain.FinanceDashboard, error) {
	return s.landRepo.GetFinanceDashboard(ctx, filter)
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

// ValuationAttachment — үнэлгээг БУЦААХАД хавсаргах файл (ЗААВАЛ БИШ).
//
// `committed` нь дотоод тэмдэглэгээ: шилжилт амжилттай болсны дараа л true
// болно. Үүнгүй бол шилжилт унасан тохиолдолд хадгалалтад орфон файл үлдэнэ.
type ValuationAttachment struct {
	Filename    string
	ContentType string
	Data        []byte

	committed bool
}
