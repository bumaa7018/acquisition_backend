package service

import (
	"context"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

// DecisionServices — router-ийн `optionalSvcs ...any` руу нэг утгаар
// дамжуулах багц. WorkType/Budget хоёр нь ижил интерфейстэй тул тусад нь
// дамжуулбал type switch тэдгээрийг ялгаж чадахгүй.
type DecisionServices struct {
	Draft    DecisionDraftService
	WorkType DecisionOptionService
	Budget   DecisionOptionService
}

type DecisionDraftService interface {
	List(ctx context.Context, f domain.DecisionDraftFilter) ([]*domain.DecisionDraft, int64, error)
	GetByID(ctx context.Context, id string) (*domain.DecisionDraft, error)
	Create(ctx context.Context, req CreateDecisionDraftRequest, actor string) (*domain.DecisionDraft, error)
	Update(ctx context.Context, id string, req UpdateDecisionDraftRequest, actor string) (*domain.DecisionDraft, error)
	Delete(ctx context.Context, id string) error
	Confirm(ctx context.Context, id string, req ConfirmDecisionDraftRequest, actor string) (*domain.DecisionDraft, error)
	AddProgress(ctx context.Context, id string, req AddDecisionDraftProgressRequest, actor string) (*domain.DecisionDraftProgressHistory, error)
	ListProgressHistory(ctx context.Context, id string) ([]*domain.DecisionDraftProgressHistory, error)

	ListParcels(ctx context.Context, draftID string) ([]*domain.DecisionDraftParcel, error)
	ListByParcel(ctx context.Context, parcelUUID string) ([]*domain.DecisionDraftParcel, error)
	LinkParcel(ctx context.Context, draftID, parcelUUID, fundingLinkID, actor string) error
	SetParcelFundingSource(ctx context.Context, draftID, linkID, fundingLinkID string) error
	UnlinkParcel(ctx context.Context, draftID, linkID, actor string) error

	ListFundingSourceOptions(ctx context.Context) ([]*domain.FundingSourceOption, error)
	CreateFundingSourceOption(ctx context.Context, req CreateFundingSourceRequest, actor string) (*domain.FundingSourceOption, error)
	ListDraftFundingSources(ctx context.Context, draftID string) ([]*domain.DecisionDraftFundingLink, error)
	AddFundingSource(ctx context.Context, draftID, fundingSourceID string, amount float64, actor string) error
	RemoveFundingSource(ctx context.Context, draftID, linkID string) error
}

// CreateDecisionDraftRequest — үүсгэх үед захирамжийн дугаар/огноог БӨГЛӨХГҮЙ.
// Тэдгээр нь "Явц нэмэх" (Confirm) үед бөглөгдөнө.
// Санхүүгийн эх үүсвэр энд БАЙХГҮЙ — төслийг үүсгэсний дараа дэлгэрэнгүй
// хэсгээс олноор нь нэмнэ (AddFundingSource).
type CreateDecisionDraftRequest struct {
	ProposalNo   string `json:"proposal_no"       validate:"required"`
	Location     string `json:"location"`
	DurationYear *int   `json:"duration_year"`
	WorkTypeID   *int   `json:"work_type_id"`
	BudgetID     *int   `json:"budget_id"`
}

type UpdateDecisionDraftRequest struct {
	ProposalNo   string `json:"proposal_no"`
	Location     string `json:"location"`
	DurationYear *int   `json:"duration_year"`
	WorkTypeID   *int   `json:"work_type_id"`
	BudgetID     *int   `json:"budget_id"`
}

// CreateFundingSourceRequest — захирамжаас ГАРААР шинэ санхүүгийн эх үүсвэр
// бүртгэх. Чөлөөлөлтгүй (бие даасан) мөр болж funding_sources-д бичигдэнэ.
type CreateFundingSourceRequest struct {
	OrganizationName string   `json:"organization_name" validate:"required"`
	SourceType       string   `json:"source_type"       validate:"required"`
	Amount           *float64 `json:"amount"`
	Currency         string   `json:"currency"`
	Note             string   `json:"note"`
}

// ConfirmDecisionDraftRequest — "Захирамжийн явц нэмэх". Хоёулаа заавал.
type ConfirmDecisionDraftRequest struct {
	DecreeNumber string `json:"decree_number"`
	DecisionDate string `json:"decision_date"` // YYYY-MM-DD
	Note         string `json:"note"`
}

type AddDecisionDraftProgressRequest struct {
	ProgressType string `json:"progress_type"`
	Recipient    string `json:"recipient"`
	ProgressDate string `json:"progress_date"` // YYYY-MM-DD
	Note         string `json:"note"`
}

type decisionDraftService struct {
	repo repository.DecisionDraftRepository
}

func NewDecisionDraftService(repo repository.DecisionDraftRepository) DecisionDraftService {
	return &decisionDraftService{repo: repo}
}

func (s *decisionDraftService) List(ctx context.Context, f domain.DecisionDraftFilter) ([]*domain.DecisionDraft, int64, error) {
	return s.repo.List(ctx, f)
}

// GetByID — дэлгэрэнгүйд санхүүгийн эх үүсвэрүүдийг хамт буцаана (жагсаалтад
// зөвхөн тоо/нэрсийн нийлбэр очно).
func (s *decisionDraftService) GetByID(ctx context.Context, id string) (*domain.DecisionDraft, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	funding, err := s.repo.ListDraftFundingSources(ctx, id)
	if err != nil {
		return nil, err
	}
	d.FundingSources = funding
	return d, nil
}

func (s *decisionDraftService) Create(ctx context.Context, req CreateDecisionDraftRequest, actor string) (*domain.DecisionDraft, error) {
	if strings.TrimSpace(req.ProposalNo) == "" {
		return nil, domain.ErrInvalidInput
	}
	d := &domain.DecisionDraft{
		ProposalNo:   strings.TrimSpace(req.ProposalNo),
		Location:     strings.TrimSpace(req.Location),
		DurationYear: req.DurationYear,
		Status:       domain.DecisionDraftStatusDraft,
		WorkTypeID:   zeroToNil(req.WorkTypeID),
		BudgetID:     zeroToNil(req.BudgetID),
		CreatedBy:    actor,
		UpdatedBy:    actor,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, d.ID)
}

func (s *decisionDraftService) Update(ctx context.Context, id string, req UpdateDecisionDraftRequest, actor string) (*domain.DecisionDraft, error) {
	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == domain.DecisionDraftStatusConfirmed {
		return nil, domain.ErrDecisionConfirmed
	}

	if strings.TrimSpace(req.ProposalNo) != "" {
		cur.ProposalNo = strings.TrimSpace(req.ProposalNo)
	}
	cur.Location = strings.TrimSpace(req.Location)
	cur.DurationYear = req.DurationYear
	cur.WorkTypeID = zeroToNil(req.WorkTypeID)
	cur.BudgetID = zeroToNil(req.BudgetID)
	cur.UpdatedBy = actor

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *decisionDraftService) Delete(ctx context.Context, id string) error {
	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cur.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}
	return s.repo.Delete(ctx, id)
}

// Confirm — захирамжийн дугаар/огноог бөглөж баталгаажуулна. Үүнээс хойш
// төсөлд ямар нэг өөрчлөлт (засвар, нэгж талбар холбох/хасах) орохгүй.
func (s *decisionDraftService) Confirm(ctx context.Context, id string, req ConfirmDecisionDraftRequest, actor string) (*domain.DecisionDraft, error) {
	decree := strings.TrimSpace(req.DecreeNumber)
	date := strings.TrimSpace(req.DecisionDate)
	if decree == "" || date == "" {
		return nil, domain.ErrDecreeRequired
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, domain.ErrDecreeRequired
	}

	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == domain.DecisionDraftStatusConfirmed {
		return nil, domain.ErrDecisionConfirmed
	}
	if err := s.repo.Confirm(ctx, id, decree, date, strings.TrimSpace(req.Note), actor); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *decisionDraftService) AddProgress(ctx context.Context, id string, req AddDecisionDraftProgressRequest, actor string) (*domain.DecisionDraftProgressHistory, error) {
	if strings.TrimSpace(req.ProgressType) != domain.DecisionDraftProgressReviewing {
		return nil, domain.ErrInvalidInput
	}
	recipient := strings.TrimSpace(req.Recipient)
	date := strings.TrimSpace(req.ProgressDate)
	if recipient == "" || date == "" {
		return nil, domain.ErrInvalidInput
	}
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == domain.DecisionDraftStatusConfirmed {
		return nil, domain.ErrDecisionConfirmed
	}
	h := &domain.DecisionDraftProgressHistory{
		DecisionDraftID: id,
		ProgressType:    domain.DecisionDraftProgressReviewing,
		Recipient:       recipient,
		ProgressDate:    parsed,
		Note:            strings.TrimSpace(req.Note),
		CreatedBy:       actor,
	}
	if err := s.repo.AddProgressHistory(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *decisionDraftService) ListProgressHistory(ctx context.Context, id string) ([]*domain.DecisionDraftProgressHistory, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListProgressHistory(ctx, id)
}

func (s *decisionDraftService) ListParcels(ctx context.Context, draftID string) ([]*domain.DecisionDraftParcel, error) {
	if _, err := s.repo.GetByID(ctx, draftID); err != nil {
		return nil, err
	}
	return s.repo.ListParcels(ctx, draftID)
}

func (s *decisionDraftService) ListByParcel(ctx context.Context, parcelUUID string) ([]*domain.DecisionDraftParcel, error) {
	return s.repo.ListByParcel(ctx, parcelUUID)
}

// LinkParcel — нэгж талбарыг захирамжид холбоно.
//
// Дүрмүүд:
//   - Баталгаажсан захирамжид холбохгүй
//   - Зөвхөн "Чөлөөлсөн" (status = 5) төлөвтэй нэгж талбар холбогдоно
//   - Нэг нэгж талбар зөвхөн НЭГ идэвхтэй захирамжтай (DB-д partial unique
//     index-ээр мөн баталгаажна)
//   - Нэг захирамжид олон өөр чөлөөлөлтийн нэгж талбар холбогдож болно.
func (s *decisionDraftService) LinkParcel(ctx context.Context, draftID, parcelUUID, fundingLinkID, actor string) error {
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	if draft.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}

	// Санхүүгийн эх үүсвэр заавал: захирамжид эх үүсвэр нэмээгүй бол, эсвэл
	// нэгж талбарт эх үүсвэр сонгоогүй бол холбохгүй.
	if strings.TrimSpace(fundingLinkID) == "" {
		return domain.ErrFundingSourceRequired
	}
	belongs, err := s.repo.FundingLinkBelongsTo(ctx, draftID, fundingLinkID)
	if err != nil {
		return err
	}
	if !belongs {
		return domain.ErrFundingSourceRequired
	}

	parcel, err := s.repo.GetParcelInfo(ctx, parcelUUID)
	if err != nil {
		return err
	}
	if parcel.Status != domain.ParcelStatusReleased {
		return domain.ErrParcelNotReleased
	}

	// Урьдчилсан шалгалт — DB index-ийн ерөнхий алдааны оронд тодорхой
	// мессеж буцаахын тулд.
	existing, err := s.repo.ActiveLinkOf(ctx, parcelUUID)
	if err != nil {
		return err
	}
	if existing != "" {
		return domain.ErrParcelAlreadyLinked
	}

	if _, err := s.repo.LinkParcel(ctx, draftID, parcelUUID, fundingLinkID, actor); err != nil {
		return err
	}
	if draft.AcquisitionID == nil {
		return s.repo.SetAcquisition(ctx, draftID, parcel.AcquisitionID)
	}
	return nil
}

// UnlinkParcel — холбоосыг түүх үлдээж хасна. Хасагдсан нэгж талбар өөр
// захирамжид дахин холбогдох боломжтой болно.
func (s *decisionDraftService) UnlinkParcel(ctx context.Context, draftID, linkID, actor string) error {
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	if draft.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}
	return s.repo.UnlinkParcel(ctx, draftID, linkID, actor)
}

func (s *decisionDraftService) ListFundingSourceOptions(ctx context.Context) ([]*domain.FundingSourceOption, error) {
	return s.repo.ListFundingSourceOptions(ctx)
}

func (s *decisionDraftService) CreateFundingSourceOption(ctx context.Context, req CreateFundingSourceRequest, actor string) (*domain.FundingSourceOption, error) {
	org := strings.TrimSpace(req.OrganizationName)
	srcType := strings.TrimSpace(req.SourceType)
	if org == "" || srcType == "" {
		return nil, domain.ErrInvalidInput
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "MNT"
	}
	o := &domain.FundingSourceOption{
		OrganizationName: org,
		SourceType:       srcType,
		Amount:           req.Amount,
		Currency:         currency,
	}
	if err := s.repo.CreateFundingSourceOption(ctx, o, strings.TrimSpace(req.Note), actor); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *decisionDraftService) ListDraftFundingSources(ctx context.Context, draftID string) ([]*domain.DecisionDraftFundingLink, error) {
	if _, err := s.repo.GetByID(ctx, draftID); err != nil {
		return nil, err
	}
	return s.repo.ListDraftFundingSources(ctx, draftID)
}

// AddFundingSource — захирамжид санхүүгийн эх үүсвэр нэмнэ. Олон эх үүсвэр
// нэмж болно; баталгаажсан захирамжид өөрчлөлт орохгүй.
func (s *decisionDraftService) AddFundingSource(ctx context.Context, draftID, fundingSourceID string, amount float64, actor string) error {
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	if draft.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}
	if amount < 0 {
		return domain.ErrInvalidInput
	}
	return s.repo.AddFundingSource(ctx, draftID, fundingSourceID, amount, actor)
}

func (s *decisionDraftService) RemoveFundingSource(ctx context.Context, draftID, linkID string) error {
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	if draft.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}
	// Нэгж талбар ашиглаж байгаа эх үүсвэрийг хасвал тэдгээр талбарууд
	// санхүүжилтгүй үлдэнэ — эхлээд талбарын эх үүсвэрийг сольж өгнө.
	inUse, err := s.repo.FundingLinkInUse(ctx, linkID)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrFundingSourceInUse
	}
	return s.repo.RemoveFundingSource(ctx, draftID, linkID)
}

// SetParcelFundingSource — холбогдсон нэгж талбарын санхүүгийн эх үүсвэрийг
// солино (холбосны дараа өөрчлөх боломж).
func (s *decisionDraftService) SetParcelFundingSource(ctx context.Context, draftID, linkID, fundingLinkID string) error {
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	if draft.Status == domain.DecisionDraftStatusConfirmed {
		return domain.ErrDecisionConfirmed
	}
	if strings.TrimSpace(fundingLinkID) == "" {
		return domain.ErrFundingSourceRequired
	}
	belongs, err := s.repo.FundingLinkBelongsTo(ctx, draftID, fundingLinkID)
	if err != nil {
		return err
	}
	if !belongs {
		return domain.ErrFundingSourceRequired
	}
	return s.repo.SetParcelFundingLink(ctx, draftID, linkID, fundingLinkID)
}

// zeroToNil — 0 сонголтыг NULL болгоно (frontend "сонгоогүй"-г 0-ээр илэрхийлнэ).
func zeroToNil(v *int) *int {
	if v == nil || *v == 0 {
		return nil
	}
	return v
}
