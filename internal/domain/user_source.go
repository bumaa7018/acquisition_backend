package domain

// Хэрэглэгчийн ЭХ СУРВАЛЖ — нэвтрэх бүртгэл нь АЛЬ САНД амьдарч байгааг заана.
//
// ЯАГААД ЗААВАЛ ХЭРЭГТЭЙ ВЭ: domain.ID нь int32 бөгөөд хоёр сан тус тусын
// serial дараалалтай тул user_id МӨРГӨЛДӨНӨ — authdb-ийн admin(3) ба appdb-ийн
// үнэлгээний байгууллагын ажилтан(3) нь ижил тоо. Эх сурвалжийн тэмдэггүйгээр
// токен дахь ID нь хоёр өөр хүнийг заах тул нэг нь нөгөөгийн эрхээр орно.
// Иймд хэрэглэгч хайх БҮХ зам (нэвтрэлт, refresh, сесс хүчингүйлэлт) энэ
// тэмдгээр чиглүүлэгдэнэ.
//
// Эдгээр утга нь user_token_epoch / revoked_tokens / password_reset_tokens дэх
// user_source баганын утгатай ИЖИЛ байх ёстой.
const (
	// UserSourceAuth — sdplatform.sd_user (authdb). Дотоод ажилтан, админ.
	UserSourceAuth = "auth"
	// UserSourceValuationOrg — appdb.valuation_org_employee. Үнэлгээний
	// байгууллагын ажилтан: sdplatform-д бүртгэгддэггүй, роль/эрх хэрэглэдэггүй,
	// хандалт нь зөвхөн токены `valuation_org` тэмдгээр тогтоогдоно.
	UserSourceValuationOrg = "valuation_org"
)
