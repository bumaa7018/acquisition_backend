package service_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeParcelStatusRepo — сүүлд бичигдсэн утгыг барьж авна.
type fakeParcelStatusRepo struct {
	stored *domain.ParcelStatus
	saved  *domain.ParcelStatus
}

func (f *fakeParcelStatusRepo) List(context.Context) ([]*domain.ParcelStatus, error) {
	return []*domain.ParcelStatus{f.stored}, nil
}

func (f *fakeParcelStatusRepo) GetByID(context.Context, int) (*domain.ParcelStatus, error) {
	if f.stored == nil {
		return nil, domain.ErrNotFound
	}
	cp := *f.stored
	return &cp, nil
}

func (f *fakeParcelStatusRepo) Create(_ context.Context, ps *domain.ParcelStatus) error {
	f.saved = ps
	return nil
}

func (f *fakeParcelStatusRepo) Update(_ context.Context, ps *domain.ParcelStatus) error {
	f.saved = ps
	return nil
}

func (f *fakeParcelStatusRepo) Delete(context.Context, int) error { return nil }

func strptr(s string) *string { return &s }

// Өнгө нь ЖИЖИГ үсэг рүү нэгтгэгдэнэ: DB-ийн CHECK нь `#AABBCC`-г ч
// зөвшөөрдөг тул ижил өнгө хоёр янзаар хадгалагдвал frontend-ийн харьцуулалт
// таарахгүй болно.
func TestParcelStatus_ColorNormalisedToLowercase(t *testing.T) {
	repo := &fakeParcelStatusRepo{}
	svc := service.NewParcelStatusService(repo)

	_, err := svc.Create(context.Background(), service.CreateParcelStatusRequest{
		Code: "pending", Name: "Хүлээгдэж буй", Color: "#F97316",
	})
	require.NoError(t, err)
	assert.Equal(t, "#f97316", repo.saved.Color)
}

// Буруу өнгө нь domain.ErrInvalidInput байх ёстой — response.mapError нь
// алдааг УТГААР нь харьцуулдаг тул боосон алдаа 400-ын оронд 500 буцаана.
func TestParcelStatus_RejectsInvalidColor(t *testing.T) {
	svc := service.NewParcelStatusService(&fakeParcelStatusRepo{})

	for _, bad := range []string{"f97316", "#f9731", "#gggggg", "#f97316ff", "rgb(1,2,3)"} {
		_, err := svc.Create(context.Background(), service.CreateParcelStatusRequest{
			Code: "x", Name: "X", Color: bad,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "буруу өнгө: %q", bad)
	}
}

// Update дээр `color` талбарын ГУРВАН төлөв ялгагдана.
func TestParcelStatus_UpdateColorTristate(t *testing.T) {
	const existing = "#22c55e"

	t.Run("талбар байхгүй бол өнгө хэвээр", func(t *testing.T) {
		repo := &fakeParcelStatusRepo{stored: &domain.ParcelStatus{
			ID: 1, Code: "released", Name: "Чөлөөлсөн", Color: existing,
		}}
		svc := service.NewParcelStatusService(repo)

		_, err := svc.Update(context.Background(), 1, service.UpdateParcelStatusRequest{
			Name: "Чөлөөлсөн",
		})
		require.NoError(t, err)
		assert.Equal(t, existing, repo.saved.Color,
			"өнгийг мэддэггүй клиент түүнийг арилгах ёсгүй")
	})

	t.Run("хоосон мөр ирвэл арилгана", func(t *testing.T) {
		repo := &fakeParcelStatusRepo{stored: &domain.ParcelStatus{
			ID: 1, Code: "released", Name: "Чөлөөлсөн", Color: existing,
		}}
		svc := service.NewParcelStatusService(repo)

		_, err := svc.Update(context.Background(), 1, service.UpdateParcelStatusRequest{
			Color: strptr(""),
		})
		require.NoError(t, err)
		assert.Empty(t, repo.saved.Color, "тодорхой хоосон утга нь арилгах хүсэлт")
	})

	t.Run("шинэ өнгө ирвэл нэгтгээд шинэчилнэ", func(t *testing.T) {
		repo := &fakeParcelStatusRepo{stored: &domain.ParcelStatus{
			ID: 1, Code: "released", Name: "Чөлөөлсөн", Color: existing,
		}}
		svc := service.NewParcelStatusService(repo)

		_, err := svc.Update(context.Background(), 1, service.UpdateParcelStatusRequest{
			Color: strptr("#EF4444"),
		})
		require.NoError(t, err)
		assert.Equal(t, "#ef4444", repo.saved.Color)
	})

	t.Run("буруу өнгө татгалзана", func(t *testing.T) {
		repo := &fakeParcelStatusRepo{stored: &domain.ParcelStatus{
			ID: 1, Code: "released", Name: "Чөлөөлсөн", Color: existing,
		}}
		svc := service.NewParcelStatusService(repo)

		_, err := svc.Update(context.Background(), 1, service.UpdateParcelStatusRequest{
			Color: strptr("улаан"),
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, repo.saved, "алдаатай үед юу ч бичигдэх ёсгүй")
	})
}
