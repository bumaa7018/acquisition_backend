package service_test

import (
	"context"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/actorctx"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// notifySpy — NotificationService-ийн хамгийн бага хэрэгжилт. Мэдэгдэл нь
// goroutine дээр илгээгддэг тул дуудалтууд суваг руу бичигдэнэ.
type notifySpy struct {
	calls chan notifyCall
}

type notifyCall struct {
	source     string
	recipients []domain.ID
	n          domain.Notification
}

func newNotifySpy() *notifySpy { return &notifySpy{calls: make(chan notifyCall, 8)} }

func (s *notifySpy) List(context.Context, string, domain.ID, bool, int, int) ([]*domain.Notification, int64, error) {
	return nil, 0, nil
}
func (s *notifySpy) UnreadCount(context.Context, string, domain.ID) (int64, error) { return 0, nil }
func (s *notifySpy) MarkRead(context.Context, string, domain.ID, uuid.UUID) error  { return nil }
func (s *notifySpy) MarkAllRead(context.Context, string, domain.ID) error          { return nil }
func (s *notifySpy) Subscribe(string, domain.ID) (<-chan *domain.Notification, func()) {
	return nil, func() {}
}

func (s *notifySpy) Notify(ctx context.Context, recipients []domain.ID, n domain.Notification) {
	s.NotifyTo(ctx, domain.UserSourceAuth, recipients, n)
}

func (s *notifySpy) NotifyTo(_ context.Context, source string, recipients []domain.ID, n domain.Notification) {
	s.calls <- notifyCall{source: source, recipients: recipients, n: n}
}

// collect — хүлээгдэж буй тоо хэмжээний дуудалтыг эх сурвалжаар нь цуглуулна.
func (s *notifySpy) collect(t *testing.T, want int) map[string]notifyCall {
	t.Helper()
	out := make(map[string]notifyCall, want)
	for i := 0; i < want; i++ {
		select {
		case call := <-s.calls:
			out[call.source] = call
		case <-time.After(3 * time.Second):
			t.Fatalf("мэдэгдэл ирсэнгүй: %d/%d", i, want)
		}
	}
	return out
}

// Нэгж талбар «Үнэлгээ хийх» төлөвт орох үед мэдэгдэл ХОЁР эх сурвалж руу
// ТУСАД нь явна: хуваарилагдсан ажилтнууд auth руу, оноогдсон байгууллагууд
// нь өөрсдийн АЖИЛТНЫ дугаараар valuation_org руу.
//
// РЕГРЕСС: өмнө нь байгууллагын ID нь хэрэглэгчийн ID мэт auth жагсаалтад
// нэмэгддэг байсан тул байгууллагын ажилтан мэдэгдэл авахгүй, харин ижил
// дугаартай ДОТООД хэрэглэгч нь хүлээн авах эрсдэлтэй байв.
func TestLandService_ParcelStatusNotification_SplitsRecipientSources(t *testing.T) {
	acqID, parcelPK := uuid.New(), uuid.New()
	profOrgID, independentOrgID := domain.ID(7), domain.ID(9)

	landRepo := &mocks.MockLandAcquisitionRepository{}
	spy := newNotifySpy()
	svc := service.NewLandService(landRepo, nil, nil, service.NotificationService(spy))

	landRepo.On("ParcelStatusRequirements", mock.Anything, acqID, parcelPK, domain.ParcelStatusEvaluating).
		Return([]domain.ParcelStatusRequirement{}, nil)
	landRepo.On("UpdateParcelStatus", mock.Anything, parcelPK, acqID,
		domain.ParcelStatusEvaluating, "41", "").Return(nil)
	landRepo.On("GetParcelByID", mock.Anything, parcelPK).Return(&domain.Parcel{
		ID:               parcelPK,
		ParcelID:         "010101000001",
		StatusID:         domain.ParcelStatusEvaluating,
		StatusName:       "Үнэлгээ хийх",
		IndependentOrgID: &independentOrgID,
	}, nil)
	landRepo.On("GetAssignees", mock.Anything, acqID).
		Return([]domain.AcquisitionAssignee{{UserID: 41}, {UserID: 42}}, nil)
	landRepo.On("GetByID", mock.Anything, acqID).Return(&domain.LandAcquisition{
		ID: acqID, ProfessionalOrgID: &profOrgID,
	}, nil)
	landRepo.On("OrgUserIDs", mock.Anything, profOrgID).Return([]domain.ID{41, 101}, nil)
	landRepo.On("OrgUserIDs", mock.Anything, independentOrgID).Return([]domain.ID{102}, nil)

	// Actor нь ДОТООД ажилтан — зөвхөн auth жагсаалтаас хасагдана.
	reqCtx := actorctx.With(context.Background(), actorctx.Actor{
		ID: 41, Source: domain.UserSourceAuth, Name: "Тест Ажилтан",
	})
	require.NoError(t, svc.UpdateParcelStatus(reqCtx, domain.ParcelStatusChange{
		AcquisitionID: acqID,
		ParcelID:      parcelPK,
		StatusID:      domain.ParcelStatusEvaluating,
		CreatedBy:     "41",
	}, nil))

	// 1) статус шилжилт (ажилтнууд), 2) үнэлгээ хүсэх (байгууллагууд).
	calls := spy.collect(t, 2)

	authCall, ok := calls[domain.UserSourceAuth]
	require.True(t, ok, "дотоод ажилтнуудад мэдэгдэл очих ёстой")
	assert.Equal(t, []domain.ID{41, 42}, authCall.recipients)
	assert.Equal(t, domain.NotifParcelStatus, authCall.n.Type)
	require.NotNil(t, authCall.n.ActorID)
	assert.Equal(t, domain.ID(41), *authCall.n.ActorID)

	orgCall, ok := calls[domain.UserSourceValuationOrg]
	require.True(t, ok, "байгууллагын ажилтнуудад мэдэгдэл очих ёстой")
	// Байгууллагын ID (7, 9) БИШ, ажилтны ID-ууд явна.
	assert.Equal(t, []domain.ID{41, 101, 102}, orgCall.recipients)
	assert.Equal(t, domain.NotifValuationRequested, orgCall.n.Type)
	// Actor нь өөр эх сурвалжийнх тул хасагдахгүй: valuation_org-ийн 41
	// дугаартай ажилтан нь дотоод 41-тэй ӨӨР хүн.
	assert.Nil(t, orgCall.n.ActorID)
	assert.Equal(t, "Тест Ажилтан", orgCall.n.ActorName)
}
