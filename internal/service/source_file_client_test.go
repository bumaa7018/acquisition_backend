package service_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const srcFilePath = "mgis/011/01107/1681600281/01107-10-00109-26/01107-10-00109-26-01-6a7e697d329b7.pdf"

// ftp_id-гаар ЗӨВ сервер сонгогдож, зам нь хэвээр залгагдана.
func TestHTTPSourceFileClient_PicksHostByFtpID(t *testing.T) {
	var gotPath string
	ftp3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer ftp3.Close()

	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("буруу сервер рүү хандлаа")
	}))
	defer other.Close()

	client := service.NewHTTPSourceFileClient(other.URL, map[string]string{"3": ftp3.URL}, 0)
	file, err := client.Open(ctx, "3", srcFilePath)
	require.NoError(t, err)
	defer file.Content.Close()

	assert.Equal(t, "/"+srcFilePath, gotPath)
	assert.Equal(t, "application/pdf", file.ContentType)
	body, err := io.ReadAll(file.Content)
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.4", string(body))
}

// ftp_id тохирохгүй эсвэл хоосон бол анхдагч хаяг руу.
func TestHTTPSourceFileClient_FallsBackToDefaultHost(t *testing.T) {
	hits := 0
	def := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("ok"))
	}))
	defer def.Close()

	client := service.NewHTTPSourceFileClient(def.URL, map[string]string{"3": "http://ftp3.invalid"}, 0)
	for _, ftpID := range []string{"", "  ", "99"} {
		file, err := client.Open(ctx, ftpID, srcFilePath)
		require.NoError(t, err, "ftp_id=%q", ftpID)
		file.Content.Close()
	}
	assert.Equal(t, 3, hits)
}

// Хаяг огт тохируулаагүй бол тодорхой алдаа (500 биш).
func TestHTTPSourceFileClient_NotConfigured(t *testing.T) {
	_, err := service.NewHTTPSourceFileClient("", nil, 0).Open(ctx, "3", srcFilePath)
	assert.Equal(t, domain.ErrSourceFileNotConfigured, err)
}

// Эх системийн зам итгэмжлэгдээгүй — ".." нь тохируулсан хостын хүрээнээс
// гаргах эрсдэлтэй тул татгалзана.
func TestHTTPSourceFileClient_RejectsTraversal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("traversal замаар хүсэлт явуулж болохгүй")
	}))
	defer srv.Close()

	client := service.NewHTTPSourceFileClient(srv.URL, nil, 0)
	for _, bad := range []string{"../../etc/passwd", "mgis/../../secret.pdf", `mgis\..\..\x.pdf`, "  "} {
		_, err := client.Open(ctx, "", bad)
		assert.Equal(t, domain.ErrInvalidInput, err, "зам: %q", bad)
	}
}

// Файл сервер дээр байхгүй бол 404 → ErrNotFound.
func TestHTTPSourceFileClient_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := service.NewHTTPSourceFileClient(srv.URL, nil, 0).Open(ctx, "", srcFilePath)
	assert.Equal(t, domain.ErrNotFound, err)
}
