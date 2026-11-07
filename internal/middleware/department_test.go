package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeDeptChecker struct{ dept *int }

func (f fakeDeptChecker) AcquisitionDepartmentID(context.Context, uuid.UUID) (*int, error) {
	return f.dept, nil
}
func (f fakeDeptChecker) ParcelDepartmentID(context.Context, uuid.UUID) (*int, error) {
	return f.dept, nil
}

func TestRequireAcquisitionDepartment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	five, seven := 5, 7
	cases := []struct {
		name      string
		roles     []string
		callerDep *domain.ID
		acqDep    *int
		want      int
	}{
		{"системийн админ — хэлтэс хамаарахгүй", []string{"admin"}, nil, &seven, http.StatusOK},
		{"өөрийн хэлтэс", []string{"specialist"}, ptrID(5), &five, http.StatusOK},
		{"өөр хэлтэс", []string{"department_head"}, ptrID(5), &seven, http.StatusForbidden},
		{"хэлтэсгүй чөлөөлөлт", []string{"specialist"}, ptrID(5), nil, http.StatusForbidden},
		{"хэлтэсгүй дуудагч", []string{"specialist"}, nil, &five, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/x/:id", func(c *gin.Context) {
				c.Set(CtxRoles, tc.roles)
				if tc.callerDep != nil {
					c.Set(CtxDepartmentID, *tc.callerDep)
				}
				c.Next()
			}, RequireAcquisitionDepartment(fakeDeptChecker{tc.acqDep}), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			c.Request = httptest.NewRequest(http.MethodGet, "/x/"+uuid.New().String(), nil)
			r.ServeHTTP(w, c.Request)
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func ptrID(v domain.ID) *domain.ID { return &v }
