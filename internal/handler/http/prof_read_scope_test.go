package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	jwtpkg "acquisition_backend/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Үнэлгээний байгууллага /prof замаар зөвхөн ӨӨРИЙН урсгалыг уншина:
// чөлөөлөлтийн гишүүнчлэл дангаараа бусад урсгалын дүнг нээхгүй.

const profTestOrg = domain.ID(31)

// profRouter — P1 нэгж талбарын ХӨНДЛӨНГИЙН байгууллагын (үндсэн биш) токен.
func profRouter(t *testing.T, acqID uuid.UUID, scope domain.ValuationOrgScope) (*gin.Engine, *mocks.MockLandService, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	org := profTestOrg
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{ID: 90, ValuationOrg: true, OrgID: &org})
	require.NoError(t, err)
	svc := &mocks.MockLandService{}
	svc.On("IsAcquisitionLocked", mock.Anything, acqID).Return(false, nil).Maybe()
	svc.On("IsParcelAcquisitionLocked", mock.Anything, mock.Anything).Return(false, nil).Maybe()
	svc.On("IsProfMemberOfAcquisition", mock.Anything, acqID, profTestOrg).Return(true, nil).Maybe()
	svc.On("IsProfMemberOfParcel", mock.Anything, mock.Anything, profTestOrg).Return(true, nil).Maybe()
	svc.On("ValuationOrgReadScope", mock.Anything, acqID, profTestOrg).Return(scope, nil).Maybe()
	router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)
	return router, svc, pair.AccessToken
}

func profGet(router *gin.Engine, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/prof/land-acquisitions/"+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

var independentOnP1 = domain.ValuationOrgScope{IndependentParcels: map[string]bool{"P1": true}}

func TestProf_ListAssetsFiltersToOwnStreams(t *testing.T) {
	acqID := uuid.New()
	router, svc, token := profRouter(t, acqID, independentOnP1)
	own := uuid.New()
	svc.On("ListAssets", mock.Anything, acqID, mock.Anything).Return([]*domain.Asset{
		{ID: own, AcquisitionID: acqID, ParcelID: "P1", ValuationType: "independent"},
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P1", ValuationType: "asset"},       // үндсэн байгууллагынх
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P1", ValuationType: "mika"},        // МИКА
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P2", ValuationType: "independent"}, // өөр нэгж талбар
	}, int64(4), nil)

	w := profGet(router, token, acqID.String()+"/assets")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, own.String(), body.Data[0].ID)
}

func TestProf_ListCompensationsFiltersToOwnStreams(t *testing.T) {
	acqID := uuid.New()
	router, svc, token := profRouter(t, acqID, independentOnP1)
	own := uuid.New()
	svc.On("ListCompensations", mock.Anything, acqID, "", "").Return([]*domain.Compensation{
		{ID: own, AcquisitionID: acqID, ParcelID: "P1", ValuationType: "independent"},
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P1", ValuationType: "asset"},
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P2", ValuationType: "independent"},
	}, nil)

	w := profGet(router, token, acqID.String()+"/compensations")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, own.String(), body.Data[0].ID)
}

func TestProf_SingleStreamReads(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		want        int
	}{
		{"өөрийн хөндлөнгийн урсгал", "parcel_id=P1&valuation_type=independent", http.StatusOK},
		{"үндсэн урсгал хаалттай", "parcel_id=P1&valuation_type=asset", http.StatusForbidden},
		{"урсгал заагаагүй (=asset) хаалттай", "parcel_id=P1", http.StatusForbidden},
		{"МИКА урсгал хаалттай", "parcel_id=P1&valuation_type=mika", http.StatusForbidden},
		{"өөр нэгж талбар хаалттай", "parcel_id=P2&valuation_type=independent", http.StatusForbidden},
	} {
		for _, endpoint := range []string{"land-valuation", "valuation-notes"} {
			t.Run(endpoint+"/"+tc.name, func(t *testing.T) {
				acqID := uuid.New()
				router, svc, token := profRouter(t, acqID, independentOnP1)
				svc.On("GetLandValuation", mock.Anything, acqID, mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Maybe()
				svc.On("ListValuationNotes", mock.Anything, acqID, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

				w := profGet(router, token, acqID.String()+"/"+endpoint+"?"+tc.query)
				assert.Equal(t, tc.want, w.Code, w.Body.String())
				if tc.want != http.StatusOK {
					svc.AssertNotCalled(t, "GetLandValuation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					svc.AssertNotCalled(t, "ListValuationNotes", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				}
			})
		}
	}
}

// Мөрийн ID-аар унших (чанар, тооцоолол, олговрын түүх) ба илгээлтийн төлөв /
// түүх / архив — мөрийн, эсвэл хүссэн урсгалын эзэн биш бол 403.
func TestProf_RowAndSubmissionReadsRequireStreamOwner(t *testing.T) {
	acqID, rowID, parcelID := uuid.New(), uuid.New(), uuid.New()
	for _, path := range []string{
		"/assets/" + rowID.String() + "/specs",
		"/assets/" + rowID.String() + "/calculations",
		"/compensations/" + rowID.String() + "/history",
		"/parcels/" + parcelID.String() + "/valuation-status?valuation_type=asset",
		"/parcels/" + parcelID.String() + "/valuation-status-history?valuation_type=asset",
		"/parcels/" + parcelID.String() + "/valuation-snapshots?valuation_type=mika",
	} {
		t.Run(path, func(t *testing.T) {
			router, svc, token := profRouter(t, acqID, independentOnP1)
			svc.On("AuthorizeAssetWrite", mock.Anything, acqID, rowID, mock.Anything).Return("", domain.ErrForbidden).Maybe()
			svc.On("AuthorizeCompensationWrite", mock.Anything, acqID, rowID, mock.Anything).Return("", domain.ErrForbidden).Maybe()
			svc.On("AuthorizeValuationSubmit", mock.Anything, acqID, parcelID, mock.Anything, mock.Anything).Return("", domain.ErrForbidden).Maybe()

			w := profGet(router, token, acqID.String()+path)
			assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
			for _, m := range []string{"ListAssetSpecs", "ListAssetCalculations", "ListCompensationHistory",
				"GetValuationSubmission", "ListValuationSubmissionHistory", "ListValuationSnapshots"} {
				svc.AssertNotCalled(t, m, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				svc.AssertNotCalled(t, m, mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// Дотоод ажилтны уншилт өөрчлөгдөөгүй — урсгалаар шүүгдэхгүй.
func TestInternal_ListCompensationsUnfiltered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	acqID := uuid.New()
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{ID: 77, Roles: []string{"admin"}})
	require.NoError(t, err)
	svc := &mocks.MockLandService{}
	svc.On("ListCompensations", mock.Anything, acqID, "", "").Return([]*domain.Compensation{
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P1", ValuationType: "asset"},
		{ID: uuid.New(), AcquisitionID: acqID, ParcelID: "P1", ValuationType: "mika"},
	}, nil)
	router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/"+acqID.String()+"/compensations", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Data, 2)
	svc.AssertNotCalled(t, "ValuationOrgReadScope", mock.Anything, mock.Anything, mock.Anything)
}

// Үнэлгээний байгууллага ЗӨВХӨН өөрийн оруулсан хавсралтыг устгана.
func TestProf_DeleteOnlyOwnDocuments(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	own := &domain.Document{ID: uuid.New(), UploadedBy: domain.ValuationOrgUploader(profTestOrg)}
	internal := &domain.Document{ID: uuid.New(), UploadedBy: "31"} // ижил дугаартай ДОТООД хэрэглэгч
	otherOrg := &domain.Document{ID: uuid.New(), UploadedBy: domain.ValuationOrgUploader(99)}

	del := func(router *gin.Engine, token, path string) int {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/prof"+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}
	for _, tc := range []struct {
		name string
		doc  *domain.Document
		want int
	}{
		{"өөрийн", own, http.StatusOK},
		{"дотоод ажилтных", internal, http.StatusForbidden},
		{"өөр байгууллагынх", otherOrg, http.StatusForbidden},
	} {
		t.Run("чөлөөлөлт/"+tc.name, func(t *testing.T) {
			router, svc, token := profRouter(t, acqID, domain.ValuationOrgScope{})
			svc.On("ListAcquisitionDocuments", mock.Anything, acqID).Return([]*domain.Document{own, internal, otherOrg}, nil)
			svc.On("DeleteAcquisitionDocument", mock.Anything, acqID, tc.doc.ID).Return(nil).Maybe()
			assert.Equal(t, tc.want, del(router, token, "/land-acquisitions/"+acqID.String()+"/documents/"+tc.doc.ID.String()))
			if tc.want != http.StatusOK {
				svc.AssertNotCalled(t, "DeleteAcquisitionDocument", mock.Anything, mock.Anything, mock.Anything)
			}
		})
		t.Run("нэгж талбар/"+tc.name, func(t *testing.T) {
			router, svc, token := profRouter(t, acqID, domain.ValuationOrgScope{})
			svc.On("ListParcelDocuments", mock.Anything, parcelID).Return([]*domain.Document{own, internal, otherOrg}, nil)
			svc.On("DeleteParcelDocument", mock.Anything, parcelID, tc.doc.ID).Return(nil).Maybe()
			assert.Equal(t, tc.want, del(router, token, "/parcels/"+parcelID.String()+"/documents/"+tc.doc.ID.String()))
			if tc.want != http.StatusOK {
				svc.AssertNotCalled(t, "DeleteParcelDocument", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// Тэмдэггүй (хуучин) хавсралтаас ЗӨВХӨН үнэлгээний төрлийнхийг устгана
// (өмнө оруулсан тайлангаа сольж чадахгүй бол илгээлт унана).
func TestProf_DeleteLegacyValuationDocumentOnly(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	legacyReport := &domain.Document{ID: uuid.New(), UploadedBy: "12", DocumentTypeID: docTypeReport}
	legacyIDCard := &domain.Document{ID: uuid.New(), UploadedBy: "12", DocumentTypeID: docTypeIDCard}
	for _, tc := range []struct {
		name string
		doc  *domain.Document
		want int
	}{
		{"хуучин үнэлгээний тайлан", legacyReport, http.StatusOK},
		{"хуучин иргэний үнэмлэх", legacyIDCard, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, svc, token := profRouter(t, acqID, domain.ValuationOrgScope{})
			svc.On("ListParcelDocuments", mock.Anything, parcelID).Return([]*domain.Document{legacyReport, legacyIDCard}, nil)
			svc.On("GetDocumentType", mock.Anything, docTypeReport).Return(&domain.DocumentType{ID: docTypeReport, Type: "valuation_report"}, nil).Maybe()
			svc.On("GetDocumentType", mock.Anything, docTypeIDCard).Return(&domain.DocumentType{ID: docTypeIDCard, Type: "id_card"}, nil).Maybe()
			svc.On("DeleteParcelDocument", mock.Anything, parcelID, tc.doc.ID).Return(nil).Maybe()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/prof/parcels/"+parcelID.String()+"/documents/"+tc.doc.ID.String(), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}
