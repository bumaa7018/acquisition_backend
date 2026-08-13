package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Дундын сервисийн /application/persons хариултыг эзэмшигч болгон хөрвүүлэх:
// bs_person.name → овог, first_name → өөрийн нэр.
func TestHTTPParcelInfoClient_ListApplicationPersons(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[
			{"main_applicant":true,"name":"Дорж","first_name":"Бат","person_register":"УА00112233","phone":"11223344 99887766","email_address":"bat@example.mn","person_type":"2: Монгол улсын иргэн"},
			{"main_applicant":false,"name":"Дорж","first_name":"Болд","person_register":"УА00445566","phone":"99001122","email_address":"","person_type":"2: Монгол улсын иргэн"},
			{"main_applicant":false,"name":"","first_name":"","person_register":"","phone":"","email_address":"","person_type":""}
		],"message":"OK"}`))
	}))
	defer srv.Close()

	client := service.NewHTTPParcelInfoClient(srv.URL)
	holders, err := client.ListApplicationPersons(ctx, "2024-0001", "010101000001")
	require.NoError(t, err)

	assert.Equal(t, "/application/persons", gotPath)
	assert.Equal(t, "app_no=2024-0001", gotQuery, "app_no байвал зөвхөн түүгээр хайна")

	require.Len(t, holders, 2, "хоосон мөр алгасагдана")
	assert.True(t, holders[0].MainApplicant)
	assert.Equal(t, "Дорж", holders[0].LastName)
	assert.Equal(t, "Бат", holders[0].Name)
	assert.Equal(t, "УА00112233", holders[0].RegisterNo)
	assert.Equal(t, "11223344 99887766", holders[0].Phone)
	assert.Equal(t, "bat@example.mn", holders[0].Email)
	assert.Equal(t, "2024-0001", holders[0].AppNo)
	assert.Equal(t, "2: Монгол улсын иргэн", holders[0].PersonType)
	assert.False(t, holders[1].MainApplicant)
	assert.Equal(t, "Болд", holders[1].Name)
}

// ХУУЛИЙН ЭТГЭЭД: first_name хоосон ирдэг тул байгууллагын бүтэн нэр name-д
// бичигдэж, last_name хоосон үлдэнэ (нэрийг хоёр талбарт хуулахгүй).
func TestHTTPParcelInfoClient_ListApplicationPersons_LegalEntity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[
			{"main_applicant":true,"name":"Хишигбат ХХК","first_name":"","person_register":"2080311","phone":"","email_address":"","person_type":"3: Монгол улсын хуулийн этгээд"}
		],"message":"OK"}`))
	}))
	defer srv.Close()

	holders, err := service.NewHTTPParcelInfoClient(srv.URL).
		ListApplicationPersons(ctx, "2024-0001", "010101000001")
	require.NoError(t, err)
	require.Len(t, holders, 1)

	assert.True(t, holders[0].MainApplicant)
	assert.Equal(t, "Хишигбат ХХК", holders[0].Name)
	assert.Empty(t, holders[0].LastName)
	assert.Equal(t, "2080311", holders[0].RegisterNo)
	assert.Equal(t, "3: Монгол улсын хуулийн этгээд", holders[0].PersonType)
}

// app_no хоосон бол нэгж талбарын дугаараар хайна.
func TestHTTPParcelInfoClient_ListApplicationPersons_ByParcelID(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":200,"data":[],"message":"OK"}`))
	}))
	defer srv.Close()

	holders, err := service.NewHTTPParcelInfoClient(srv.URL).
		ListApplicationPersons(ctx, "  ", "1781111606")
	require.NoError(t, err)
	assert.Empty(t, holders)
	assert.Equal(t, "parcel_id=1781111606", gotQuery)
}

// Эх системийн SQL-ийн баганууд ЯГ ТЭР хэлбэрээрээ ирэхэд зөв уншигдах эсэх:
//   - зөвхөн SQL-д байгаа багана ирнэ (id, plan_code, au1/au3, holder_* байхгүй)
//   - геометр нь "geometry" нэрээр, хэсэг нь "au2" нэрээр
//   - кодууд тоогоор (landuse=2201, old_parcel_id=0, address_khashaa=0)
//   - valid_till нь infinity (292278994-08-17) → хугацаа хязгааргүй = nil
//   - app_timestamp нь "2026-03-30 17:00:00.000"
func TestHTTPParcelInfoClient_GetParcelInfo_RawSQLShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/parcel/info/1800805431", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"data":{
			"parcel_id":1800805431,
			"old_parcel_id":0,
			"area_m2":49235,
			"landuse":2201,
			"landuse_name":"9 ба түүнээс дээш давхар нийтийн орон сууц",
			"address_khashaa":0,
			"address_streetname":"",
			"valid_from":"2026-05-13",
			"valid_till":"292278994-08-17",
			"geometry":"POLYGON ((106.81534877803381 47.935465536299766, 106.81632471290263 47.93542447328761, 106.81633366254741 47.93523336220047, 106.81534877803381 47.935465536299766))",
			"au2":"01116",
			"property_no":"Э-2201077008",
			"app_no":"01116-54-00006-26",
			"app_timestamp":"2026-03-30 17:00:00.000",
			"app_type":"54. Төсөвт байгууллагад зайлшгүй хэрэгцээтэй газрыг эзэмшүүлэх",
			"right_type":2,
			"app_status":"11: Улсын Бүртгэл зөвшөөрч дугаар олгосон",
			"decision_no":"01116-А/685/2026",
			"decision_date":"2026-05-13",
			"contract_no":"01116-2026/04452",
			"contract_date":"2026-05-13",
			"contract_begin":"2026-05-13",
			"contract_end":"2041-05-13",
			"certificate_no":"",
			"contract_property_no":"",
			"contract_status":"Хүчинтэй",
			"record_no":"",
			"record_date":null,
			"record_certificate_no":"",
			"record_status":""
		},"message":"OK"}`))
	}))
	defer srv.Close()

	src, err := service.NewHTTPParcelInfoClient(srv.URL).GetParcelInfo(ctx, "1800805431")
	require.NoError(t, err)

	// Тоогоор ирсэн кодууд текст болно
	assert.Equal(t, "1800805431", src.ParcelID)
	assert.Equal(t, "2201", src.Landuse)
	assert.Equal(t, 49235.0, src.AreaM2)
	assert.Equal(t, 2, src.RightType)
	// "0" нь ХООСОН гэсэн утга
	assert.Empty(t, src.OldParcelID, "old_parcel_id=0 нь хоосон")
	assert.Empty(t, src.AddressKhashaa, "address_khashaa=0 нь хоосон")
	// Нэр солигдсон талбарууд
	assert.Contains(t, src.GeometryWKT, "POLYGON", "geometry нэрээр ирсэн ч уншигдана")
	assert.Equal(t, "01116", src.AU2Code, "au2 нэрээр ирсэн ч уншигдана")
	// Огноо
	require.NotNil(t, src.ValidFrom)
	assert.Equal(t, "2026-05-13", src.ValidFrom.Format("2006-01-02"))
	assert.Nil(t, src.ValidTill, "infinity (292278994-08-17) → хугацаа хязгааргүй")
	require.NotNil(t, src.AppTimestamp)
	assert.Equal(t, "2026-03-30 17:00:00", src.AppTimestamp.Format("2006-01-02 15:04:05"))
	require.NotNil(t, src.ContractEnd)
	assert.Equal(t, "2041-05-13", src.ContractEnd.Format("2006-01-02"))
	assert.Nil(t, src.RecordDate)
	// SQL-д байхгүй талбарууд — хоосон, гэхдээ алдаа гаргахгүй
	assert.Equal(t, uuid.Nil, src.ID)
	assert.Empty(t, src.PlanCode)
	assert.Empty(t, src.AU1Code)
	assert.Empty(t, src.AU3Code)
	assert.Empty(t, src.HolderName)
	// Бусад
	assert.Equal(t, "01116-54-00006-26", src.AppNo)
	assert.Equal(t, "Хүчинтэй", src.ContractStatus)
	assert.Equal(t, "Э-2201077008", src.PropertyNo)
}

// Хугацаа хязгааргүйг тэмдэглэх бусад хэлбэрүүд болон RFC3339 огноо
func TestHTTPParcelInfoClient_GetParcelInfo_DateEdgeCases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":{
			"parcel_id":"010101000001",
			"valid_from":"2026-05-13T00:00:00Z",
			"valid_till":"infinity",
			"contract_end":"9999-12-31",
			"record_date":"мэдэгдэхгүй"
		},"message":"OK"}`))
	}))
	defer srv.Close()

	src, err := service.NewHTTPParcelInfoClient(srv.URL).GetParcelInfo(ctx, "010101000001")
	require.NoError(t, err)
	require.NotNil(t, src.ValidFrom)
	assert.Equal(t, "2026-05-13", src.ValidFrom.Format("2006-01-02"))
	assert.Nil(t, src.ValidTill, "infinity → nil")
	require.NotNil(t, src.ContractEnd, "9999 он нь main DB-д хадгалагдана")
	assert.Nil(t, src.RecordDate, "танигдаагүй хэлбэр → nil (алдаа биш)")
}
