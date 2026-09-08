package service_test

// Суурь зургийн тохиргооны ШАЛГАЛТ.
//
// Эдгээр дүрэм нь газрын зургийг ХООСОН болгохоос сэргийлдэг тул тестээр
// хамгаалагдана: орлуулагчгүй tile хаяг бүх тайлыг нэг зураг болгоно,
// http(s) биш схем нь browser дээр гүйцэтгэгдэх хаяг оруулах эрсдэлтэй.

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memSettingRepo — app_setting-ийн санах ойн мок.
type memSettingRepo struct {
	rec *domain.SettingRecord
}

func (m *memSettingRepo) Get(_ context.Context, key string) (*domain.SettingRecord, error) {
	if m.rec == nil || m.rec.Key != key {
		return nil, domain.ErrNotFound
	}
	return m.rec, nil
}

func (m *memSettingRepo) Upsert(_ context.Context, key, valueJSON, updatedBy string) error {
	m.rec = &domain.SettingRecord{Key: key, Value: valueJSON, UpdatedBy: updatedBy}
	return nil
}

func (m *memSettingRepo) Delete(_ context.Context, key string) error {
	if m.rec != nil && m.rec.Key == key {
		m.rec = nil
	}
	return nil
}

func TestGetBasemap_NotConfiguredIsNotAnError(t *testing.T) {
	svc := service.NewSettingService(&memSettingRepo{})
	basemap, err := svc.GetBasemap(ctx)
	require.NoError(t, err)
	assert.Nil(t, basemap, "тохируулаагүй бол nil — дэлгэц өгөгдмөл зургаа хэрэглэнэ")
}

func TestSaveBasemap_RejectsBadInput(t *testing.T) {
	svc := service.NewSettingService(&memSettingRepo{})

	cases := []struct {
		name string
		in   domain.Basemap
		want error
	}{
		{
			name: "tile хаягт орлуулагч байхгүй",
			in:   domain.Basemap{Type: "xyz", URL: "https://tile.example.mn/imagery.png", Enabled: true},
			want: domain.ErrInvalidBasemapTileURL,
		},
		{
			name: "http(s) биш схем",
			in:   domain.Basemap{Type: "xyz", URL: "javascript:alert(1)", Enabled: true},
			want: domain.ErrInvalidBasemapURL,
		},
		{
			name: "танихгүй төрөл",
			in:   domain.Basemap{Type: "wmts", URL: "https://example.mn/x", Enabled: true},
			want: domain.ErrInvalidBasemapType,
		},
		{
			name: "wms давхаргын нэргүй",
			in:   domain.Basemap{Type: "wms", URL: "https://geo.example.mn/wms", Enabled: true},
			want: domain.ErrBasemapLayerRequired,
		},
		{
			name: "image хүрээгүй",
			in:   domain.Basemap{Type: "image", URL: "https://example.mn/o.png", Enabled: true},
			want: domain.ErrBasemapExtentRequired,
		},
		{
			name: "image хүрээ буруу дараалалтай",
			in: domain.Basemap{
				Type: "image", URL: "https://example.mn/o.png", Enabled: true,
				Extent: []float64{107, 48, 106, 47},
			},
			want: domain.ErrBasemapExtentRequired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SaveBasemap(ctx, tc.in, "1")
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestSaveBasemap_StoresAndNormalises(t *testing.T) {
	repo := &memSettingRepo{}
	svc := service.NewSettingService(repo)

	saved, err := svc.SaveBasemap(ctx, domain.Basemap{
		Type: "xyz", URL: " https://tile.example.mn/{z}/{x}/{y}.png ", Enabled: true,
		MaxZoom: 19,
		// xyz-д хамааралгүй талбарууд — хадгалахдаа цэвэрлэгдэх ёстой
		Layer:  "should:drop",
		Extent: []float64{1, 2, 3, 4},
		Name:   " Агаарын зураг 2026 ",
	}, "7")
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, "https://tile.example.mn/{z}/{x}/{y}.png", saved.URL)
	assert.Equal(t, "Агаарын зураг 2026", saved.Name)
	assert.Equal(t, 19, saved.MaxZoom)
	assert.True(t, saved.Enabled)
	assert.Empty(t, saved.Layer, "xyz төрөлд wms-ийн давхаргын нэр хадгалагдах ёсгүй")
	assert.Empty(t, saved.Extent, "xyz төрөлд зургийн хүрээ хадгалагдах ёсгүй")
	assert.Equal(t, "7", saved.UpdatedBy)
}

func TestSaveBasemap_EmptyURLMeansDefault(t *testing.T) {
	svc := service.NewSettingService(&memSettingRepo{})
	saved, err := svc.SaveBasemap(ctx, domain.Basemap{Type: "xyz", URL: "  ", Enabled: true}, "1")
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.False(t, saved.Enabled, "хоосон хаяг = өгөгдмөл суурь зураг (унтраалттай)")
}

func TestResetBasemap_ReturnsToDefault(t *testing.T) {
	repo := &memSettingRepo{}
	svc := service.NewSettingService(repo)
	_, err := svc.SaveBasemap(ctx, domain.Basemap{
		Type: "xyz", URL: "https://tile.example.mn/{z}/{x}/{y}.png", Enabled: true,
	}, "1")
	require.NoError(t, err)

	require.NoError(t, svc.ResetBasemap(ctx))
	basemap, err := svc.GetBasemap(ctx)
	require.NoError(t, err)
	assert.Nil(t, basemap)
}
