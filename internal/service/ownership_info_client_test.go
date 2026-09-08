package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// УБЕГ-ийн хариултын галигласан талбарууд (ognoo, ubd, registr...) domain
// нэрсэд зөв хөрвөх ба null (bair_number, toot) хоосон текст болох ёстой.
func TestHTTPOwnershipInfoClient_ListParcelOwnerships(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":[
			{"ognoo":"20110414","ubd":"г2201014561","zoriulalt":"Гэр бүлийн хэрэгцээний","hemjee":"589",
			 "prprty_parcel_id":"1800803094","registr":"чв73062073","ner":"мандах","ovog":"нүдэнхүү",
			 "umchlugch":"Өв залгамжлуулсан","umch_code":"000005",
			 "uilchilgee":"Төрөөс хувьчилсан газрын өмчлөх эрхийг анх удаа бүртгэх ","service_code":"000002",
			 "aimag_city_name":"Улаанбаатар","aimag_code":"11","soum_district_name":"Сонгинохайрхан","soum_code":"16",
			 "bag_khoroo_name":"5-р хороо","bag_code":"59","address_street_name":"баянгол 48",
			 "bair_number":null,"toot":"18","full_address":"Улаанбаатар Сонгинохайрхан дүүрэг 5-р хороо"},
			{"ognoo":"20260514","ubd":"г2201014561","zoriulalt":"Гэр бүлийн хэрэгцээний","hemjee":"",
			 "prprty_parcel_id":"1800803094","registr":"чв79091268","ner":"отгонсүрэн","ovog":"нүдэнхүү",
			 "umchlugch":"Өмчилсөн","umch_code":"000001",
			 "uilchilgee":"Өв залгамжлалаар өмчлөгч өөрчлөгдөх","service_code":"000012",
			 "bair_number":null,"toot":null,"full_address":""}
		]}`))
	}))
	defer srv.Close()

	client := service.NewHTTPOwnershipInfoClient(srv.URL)
	items, err := client.ListParcelOwnerships(ctx, "1800803094")
	require.NoError(t, err)

	assert.Equal(t, "/xyp/ownership/citizen/parcel", gotPath)
	assert.Equal(t, "parcel_id=1800803094", gotQuery)
	require.Len(t, items, 2)

	first := items[0]
	assert.Equal(t, "20110414", first.RecordDate)
	assert.Equal(t, "г2201014561", first.CertificateNo)
	assert.Equal(t, "чв73062073", first.RegisterNo)
	assert.Equal(t, "нүдэнхүү", first.LastName)
	assert.Equal(t, "мандах", first.FirstName)
	assert.Equal(t, "Өв залгамжлуулсан", first.OwnershipType)
	assert.Equal(t, "000005", first.OwnershipCode)
	// Хоосон зайтай ирсэн үйлчилгээний нэр цэвэрлэгдэнэ.
	assert.Equal(t, "Төрөөс хувьчилсан газрын өмчлөх эрхийг анх удаа бүртгэх", first.ServiceName)
	assert.Equal(t, float64(589), first.Area)
	assert.Equal(t, "18", first.Toot)
	assert.Empty(t, first.BairNumber)

	// Тоо болгож болохгүй hemjee → 0 (мөр хаягдахгүй).
	assert.Equal(t, float64(0), items[1].Area)
	assert.Empty(t, items[1].Toot)
}

// Бүртгэл байхгүй нэгж талбар — status=false, data байхгүй. Энэ нь АЛДАА биш,
// хоосон жагсаалт (татах алхам "Олдоогүй" гэж дүгнэгдэнэ).
func TestHTTPOwnershipInfoClient_EmptyRegistryIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":false,"data":null}`))
	}))
	defer srv.Close()

	items, err := service.NewHTTPOwnershipInfoClient(srv.URL).ListParcelOwnerships(ctx, "1800803094")
	require.NoError(t, err)
	assert.Empty(t, items)
}

// Тохируулаагүй (хоосон baseURL) эсвэл дугаар хоосон — сүлжээ рүү гарахгүй.
func TestHTTPOwnershipInfoClient_RequiresBaseURLAndParcel(t *testing.T) {
	_, err := service.NewHTTPOwnershipInfoClient("").ListParcelOwnerships(ctx, "1800803094")
	require.Error(t, err)

	_, err = service.NewHTTPOwnershipInfoClient("http://example.invalid").ListParcelOwnerships(ctx, "  ")
	require.Error(t, err)
}
