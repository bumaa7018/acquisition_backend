package service_test

// Санхүүгийн эх үүсвэрийн шинэ урсгал: чөлөөлөлт бүртгэхэд эх үүсвэр асуухаа
// больсон тул захирамжийн төсөлд НЭГЖ ТАЛБАР ХОЛБОХОД сонгосон эх үүсвэр нь
// тухайн чөлөөлөлтийн санхүүжилт болж хадгалагдана.

import (
	"context"
	"errors"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDraftRepo — DecisionDraftRepository-г embed хийж, зөвхөн энэ тестэд
// хэрэгтэй методуудыг дарж бичнэ (бусад метод дуудагдвал nil interface-ээс
// panic болно — тэр нь "дуудагдах ёсгүй" гэсэн шалгалт болж ажиллана).
type stubDraftRepo struct {
	repository.DecisionDraftRepository

	draft   *domain.DecisionDraft
	parcel  *domain.ParcelLinkInfo
	belongs bool
	bindErr error

	bindCalls    [][2]string // {fundingLinkID, acquisitionID}
	setAcqCalls  [][2]string // {draftID, acquisitionID}
	parcelFundOK bool
}

func (s *stubDraftRepo) GetByID(context.Context, string) (*domain.DecisionDraft, error) {
	return s.draft, nil
}
func (s *stubDraftRepo) FundingLinkBelongsTo(context.Context, string, string) (bool, error) {
	return s.belongs, nil
}
func (s *stubDraftRepo) GetParcelInfo(context.Context, string) (*domain.ParcelLinkInfo, error) {
	return s.parcel, nil
}
func (s *stubDraftRepo) ActiveLinkOf(context.Context, string) (string, error) { return "", nil }
func (s *stubDraftRepo) LinkParcel(context.Context, string, string, string, string) (string, error) {
	return "link-1", nil
}
func (s *stubDraftRepo) SetAcquisition(_ context.Context, draftID, acquisitionID string) error {
	s.setAcqCalls = append(s.setAcqCalls, [2]string{draftID, acquisitionID})
	return nil
}
func (s *stubDraftRepo) SetParcelFundingLink(context.Context, string, string, string) error {
	s.parcelFundOK = true
	return nil
}
func (s *stubDraftRepo) BindFundingSourceToAcquisition(_ context.Context, fundingLinkID, acquisitionID string) error {
	s.bindCalls = append(s.bindCalls, [2]string{fundingLinkID, acquisitionID})
	return s.bindErr
}

func newDraft(acquisitionID *string) *domain.DecisionDraft {
	return &domain.DecisionDraft{
		ID:            "draft-1",
		Status:        domain.DecisionDraftStatusDraft,
		AcquisitionID: acquisitionID,
	}
}

func releasedParcel(acquisitionID string) *domain.ParcelLinkInfo {
	return &domain.ParcelLinkInfo{
		ID:            "parcel-uuid",
		ParcelID:      "010101000001",
		Status:        domain.ParcelStatusReleased,
		AcquisitionID: acquisitionID,
	}
}

func TestDecisionDraft_LinkParcel_SavesFundingSourceToAcquisition(t *testing.T) {
	repo := &stubDraftRepo{
		draft:   newDraft(nil), // эхний холболт — чөлөөлөлт хараахан тогтоогдоогүй
		parcel:  releasedParcel("acq-9"),
		belongs: true,
	}
	svc := service.NewDecisionDraftService(repo)

	err := svc.LinkParcel(context.Background(), "draft-1", "parcel-uuid", "fund-link-1", "user-1")

	require.NoError(t, err)
	require.Len(t, repo.bindCalls, 1, "эх үүсвэр чөлөөлөлтөд хадгалагдах ёстой")
	assert.Equal(t, [2]string{"fund-link-1", "acq-9"}, repo.bindCalls[0],
		"нэгж талбарын чөлөөлөлтөд, сонгосон эх үүсвэрээр")
	assert.Equal(t, [][2]string{{"draft-1", "acq-9"}}, repo.setAcqCalls)
}

func TestDecisionDraft_SetParcelFundingSource_UpdatesAcquisitionFunding(t *testing.T) {
	acq := "acq-9"
	repo := &stubDraftRepo{draft: newDraft(&acq), belongs: true}
	svc := service.NewDecisionDraftService(repo)

	err := svc.SetParcelFundingSource(context.Background(), "draft-1", "link-1", "fund-link-2")

	require.NoError(t, err)
	assert.True(t, repo.parcelFundOK)
	require.Len(t, repo.bindCalls, 1, "эх үүсвэр солиход ч чөлөөлөлтөд тусах ёстой")
	assert.Equal(t, [2]string{"fund-link-2", "acq-9"}, repo.bindCalls[0])
}

// Санхүүжилтийг хадгалж чадаагүй нь холбоосыг бүтэлгүйтүүлэхгүй (холбоос аль
// хэдийн үүссэн — буцаах нь хэрэглэгчийг зөв бус мэдээллээр гөлрүүлнэ).
func TestDecisionDraft_LinkParcel_BindErrorDoesNotFailLink(t *testing.T) {
	repo := &stubDraftRepo{
		draft:   newDraft(nil),
		parcel:  releasedParcel("acq-9"),
		belongs: true,
		bindErr: errors.New("db унав"),
	}
	svc := service.NewDecisionDraftService(repo)

	err := svc.LinkParcel(context.Background(), "draft-1", "parcel-uuid", "fund-link-1", "user-1")

	require.NoError(t, err)
	assert.Len(t, repo.bindCalls, 1)
}

// Эх үүсвэр сонгоогүй бол холбохгүй — хадгалах ч гэж оролдохгүй
func TestDecisionDraft_LinkParcel_FundingSourceRequired(t *testing.T) {
	repo := &stubDraftRepo{draft: newDraft(nil), parcel: releasedParcel("acq-9")}
	svc := service.NewDecisionDraftService(repo)

	err := svc.LinkParcel(context.Background(), "draft-1", "parcel-uuid", "   ", "user-1")

	assert.ErrorIs(t, err, domain.ErrFundingSourceRequired)
	assert.Empty(t, repo.bindCalls)
}
