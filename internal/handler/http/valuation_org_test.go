package http

// Ажилтны JSON оролтын задрал. UI нь бүх ID-г МӨР болгож жигдрүүлдэг тул
// employees[].id нь "3" гэж ирж болно — domain.ID (int32) руу шууд уншвал
// 400 болно. nullableID хоёуланг ч хүлээж авах ёстой.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValuationOrgBody_EmployeeIDAcceptsStringAndNumber(t *testing.T) {
	const payload = `{
		"name": "Тест Үнэлгээ ХХК",
		"employees": [
			{"id": "3",  "last_name": "Бат", "first_name": "Дорж", "register_no": "УБ11111111"},
			{"id": 4,    "last_name": "Сүх", "first_name": "Оюун", "register_no": "УБ22222222"},
			{"id": null, "last_name": "Хас", "first_name": "Туяа", "register_no": "УБ33333333"},
			{"id": "",   "last_name": "Ган", "first_name": "Сараа", "register_no": "УБ44444444"},
			{            "last_name": "Мөнх","first_name": "Ганаа", "register_no": "УБ55555555"}
		]
	}`

	var b valuationOrgBody
	require.NoError(t, json.Unmarshal([]byte(payload), &b))

	req, err := valuationOrgReqFromBody(b)
	require.NoError(t, err)
	require.Len(t, req.Employees, 5)

	require.NotNil(t, req.Employees[0].ID, `"3" мөр нь ID болж уншигдана`)
	assert.EqualValues(t, 3, *req.Employees[0].ID)

	require.NotNil(t, req.Employees[1].ID, "тоо мөн уншигдана")
	assert.EqualValues(t, 4, *req.Employees[1].ID)

	// null / "" / талбар байхгүй — бүгд ШИНЭ ажилтан.
	assert.Nil(t, req.Employees[2].ID)
	assert.Nil(t, req.Employees[3].ID)
	assert.Nil(t, req.Employees[4].ID)
}

func TestValuationOrgBody_InvalidEmployeeIDFails(t *testing.T) {
	var b valuationOrgBody
	err := json.Unmarshal([]byte(`{"name":"Т","employees":[{"id":"abc"}]}`), &b)
	assert.Error(t, err, "тоо биш ID нь 400 болох ёстой, чимээгүй 0 болох ёсгүй")
}

// employees талбарыг ОГТ илгээхгүй бол ажилтнуудад хүрэхгүй (nil), хоосон
// массив илгээвэл "бүх ажилтныг хас" гэсэн ЗОРИУДЫН үйлдэл.
func TestValuationOrgBody_NilVsEmptyEmployees(t *testing.T) {
	var absent valuationOrgBody
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Т"}`), &absent))
	req, err := valuationOrgReqFromBody(absent)
	require.NoError(t, err)
	assert.Nil(t, req.Employees)

	var empty valuationOrgBody
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Т","employees":[]}`), &empty))
	req, err = valuationOrgReqFromBody(empty)
	require.NoError(t, err)
	assert.NotNil(t, req.Employees)
	assert.Empty(t, req.Employees)
}

// Нэвтрэх эрхийн талбарууд дамжина (нууц үгийг service hash хийнэ).
func TestValuationOrgBody_LoginFieldsPassThrough(t *testing.T) {
	var b valuationOrgBody
	require.NoError(t, json.Unmarshal([]byte(`{
		"name":"Т",
		"employees":[{"last_name":"Бат","first_name":"Дорж","register_no":"УБ1",
		              "username":"dorj","password":"Testpass123!","position_name":"Үнэлгээчин"}]
	}`), &b))

	req, err := valuationOrgReqFromBody(b)
	require.NoError(t, err)
	require.Len(t, req.Employees, 1)
	assert.Equal(t, "dorj", req.Employees[0].Username)
	assert.Equal(t, "Testpass123!", req.Employees[0].Password)
	assert.Equal(t, "Үнэлгээчин", req.Employees[0].PositionName)
}
