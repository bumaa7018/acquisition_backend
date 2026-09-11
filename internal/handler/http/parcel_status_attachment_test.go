package http_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Төлөв солиход хавсаргасан файлын ХАРАГДАХ нэрийг хэрэглэгч гараас засдаг:
// `file_name` талбар ирвэл түүнийг, ирээгүй бол файлын нэрийг (service-ийн
// documentDisplayName) хэрэглэнэ. Физик файлын нэр нь өөрчлөгдөхгүй.
func TestUpdateParcelStatusAttachmentName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pdf := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("a"), 600)...)

	for _, tt := range []struct {
		name            string
		formName        string
		wantDisplayName string
	}{
		{name: "хэрэглэгчийн засварласан нэр", formName: "Иргэний өргөдөл 2026-05-12", wantDisplayName: "Иргэний өргөдөл 2026-05-12"},
		{name: "хоосон бол файлын нэр рүү шилжинэ", formName: "", wantDisplayName: ""},
		{name: "зөвхөн хоосон зайг нэр гэж авахгүй", formName: "   ", wantDisplayName: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			_ = w.WriteField("status_id", "4")
			_ = w.WriteField("reason", "Иргэн татгалзсан")
			if tt.formName != "" {
				_ = w.WriteField("file_name", tt.formName)
			}
			fw, err := w.CreateFormFile("file", "shiidver.pdf")
			require.NoError(t, err)
			_, err = fw.Write(pdf)
			require.NoError(t, err)
			require.NoError(t, w.Close())

			svc := &mocks.MockLandService{}
			acqID, parcelID := uuid.New(), uuid.New()
			svc.On("UpdateParcelStatus", mock.Anything, parcelID, acqID, domain.ParcelStatusRejected, mock.Anything, "Иргэн татгалзсан",
				mock.MatchedBy(func(a *service.ParcelStatusAttachment) bool {
					return a != nil && a.Filename == "shiidver.pdf" && a.DisplayName == tt.wantDisplayName
				})).Return(nil)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Params = gin.Params{{Key: "id", Value: acqID.String()}, {Key: "parcel_id", Value: parcelID.String()}}
			c.Request = httptest.NewRequest(http.MethodPatch, "/status", &buf)
			c.Request.Header.Set("Content-Type", w.FormDataContentType())
			handler.NewParcelWorkflowHandler(svc).UpdateParcelStatus(c)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			svc.AssertExpectations(t)
		})
	}
}
