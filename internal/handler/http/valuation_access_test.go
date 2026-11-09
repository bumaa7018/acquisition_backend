package http_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	"acquisition_backend/internal/middleware"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testDeptID / testDeptInt — дуудагч болон чөлөөлөлтийн НИЙТЛЭГ хэлтэс
// (системийн админаас бусад нь зөвхөн өөрийн хэлтсийн чөлөөлөлтөд хандана).
var (
	testDeptID  = domain.ID(5)
	testDeptInt = 5
)

func TestRouter_ValuationCancelAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name                   string
		roles                  []string
		permission             string
		assigned, locked, prof bool
		assignmentErr          error
		want                   int
	}{
		{name: "admin without permission claims", roles: []string{"admin"}, want: 200},
		{name: "admin name", roles: []string{"Админ"}, want: 200},
		{name: "senior", roles: []string{"senior_specialist"}, permission: "land:update", want: 200},
		{name: "finance", roles: []string{"finance_specialist"}, permission: "compensation:update", want: 200},
		{name: "assigned employee", roles: []string{"specialist"}, permission: "land:update", assigned: true, want: 200},
		{name: "unassigned employee", roles: []string{"specialist"}, permission: "land:update", want: 403},
		{name: "assignment lookup fails closed", roles: []string{"specialist"}, permission: "land:update", assignmentErr: errors.New("lookup failed"), want: 403},
		{name: "missing update permission", roles: []string{"specialist"}, assigned: true, want: 403},
		{name: "mika even when assigned", roles: []string{"mika"}, permission: "compensation:update", assigned: true, want: 403},
		{name: "legacy professional even when assigned", roles: []string{"professional_org"}, permission: "land:update", assigned: true, want: 403},
		{name: "appdb professional", prof: true, want: 403},
		{name: "confirmed acquisition admin", roles: []string{"admin"}, locked: true, want: 423},
		{name: "confirmed acquisition finance", roles: []string{"finance_specialist"}, permission: "compensation:update", locked: true, want: 423},
		{name: "confirmed acquisition assigned", roles: []string{"specialist"}, permission: "land:update", assigned: true, locked: true, want: 423},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acqID, parcelID := uuid.New(), uuid.New()
			manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{
				ID: domain.ID(77), FullName: "Cancellation Employee", Roles: tc.roles,
				Permissions: []string{tc.permission}, ValuationOrg: tc.prof,
				DepartmentID: &testDeptID,
			})
			require.NoError(t, err)
			svc := &mocks.MockLandService{}
			svc.On("IsAcquisitionLocked", mock.Anything, acqID).Return(tc.locked, nil).Maybe()
			svc.On("AcquisitionStatus", mock.Anything, acqID).Return(domain.LandAcquisitionStatusField, nil).Maybe()
			svc.On("IsAssigned", mock.Anything, acqID, domain.ID(77)).Return(tc.assigned, tc.assignmentErr).Maybe()
			svc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&testDeptInt, nil).Maybe()
			if tc.want == http.StatusOK {
				svc.On("TransitionValuationSubmission", mock.Anything,
					mock.MatchedBy(func(in domain.ValuationTransition) bool {
						return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
							in.Action == "cancel" && in.UserID == "Cancellation Employee"
					}), mock.MatchedBy(func(file *service.ValuationAttachment) bool {
						return file != nil && file.Filename == "cancel.pdf"
					})).Return(sampleSubmission(acqID, domain.ValuationStatusDraft, "asset"), nil).Once()
			}
			router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("action", "cancel"))
			require.NoError(t, writer.WriteField("note", "Revalue"))
			require.NoError(t, writer.WriteField("valuation_type", "asset"))
			file, err := writer.CreateFormFile("file", "cancel.pdf")
			require.NoError(t, err)
			_, err = file.Write(samplePDF)
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/valuation-status", &body)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.want != http.StatusOK {
				svc.AssertNotCalled(t, "TransitionValuationSubmission", mock.Anything, mock.Anything, mock.Anything)
			}
			svc.AssertExpectations(t)
		})
	}
}

func TestRouter_AdminReadsValuationStatusWithoutPermissionClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	acqID, parcelID := uuid.New(), uuid.New()
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{ID: 77, Roles: []string{"admin"}})
	require.NoError(t, err)
	svc := &mocks.MockLandService{}
	svc.On("GetValuationSubmission", mock.Anything, acqID, parcelID, "asset").
		Return(sampleSubmission(acqID, domain.ValuationStatusApproved, "asset"), nil).Once()
	router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/valuation-status?valuation_type=asset", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertExpectations(t)
}

func TestRouter_AcquisitionDetailIncludesAssignedUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{ID: 77, Permissions: []string{"land:read"}, DepartmentID: &testDeptID})
	require.NoError(t, err)
	acqID := uuid.New()
	svc := &mocks.MockLandService{}
	svc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&testDeptInt, nil).Once()
	svc.On("IsAssigned", mock.Anything, acqID, domain.ID(77)).Return(true, nil).Once()
	svc.On("GetByID", mock.Anything, acqID).Return(&domain.LandAcquisition{
		ID: acqID, AssignedUsers: []domain.AcquisitionAssignee{{UserID: 77, UserName: "Assigned Employee", AssignedBy: 1}},
	}, nil).Once()
	svc.On("ListFundingSources", mock.Anything, acqID).Return([]*domain.FundingSource{}, nil).Once()
	router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/"+acqID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		Data struct {
			AssignedUsers []struct {
				UserID   string `json:"user_id"`
				UserName string `json:"user_name"`
			} `json:"assigned_users"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Data.AssignedUsers, 1)
	assert.Equal(t, "77", response.Data.AssignedUsers[0].UserID)
	assert.Equal(t, "Assigned Employee", response.Data.AssignedUsers[0].UserName)
	svc.AssertExpectations(t)
}

func TestAdvanceStatusStoresCallerName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	acqID := uuid.New()
	svc := &mocks.MockLandService{}
	svc.On("AdvanceStatus", mock.Anything, acqID, domain.LandAcquisitionStatusField,
		"Start field work", "Progress Employee", "", (*time.Time)(nil)).
		Return(&domain.LandAcquisition{ID: acqID}, nil).Once()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}
	c.Set(middleware.CtxUserID, domain.ID(77))
	c.Set(middleware.CtxUserName, "Progress Employee")
	c.Request = httptest.NewRequest(http.MethodPost, "/advance", bytes.NewBufferString(`{"to_status":2,"note":"Start field work"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.NewLandHandler(svc).AdvanceStatus(c)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertExpectations(t)
}
