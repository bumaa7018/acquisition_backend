package domain

import "strconv"

// Үнэлгээний урсгал бүр (asset / independent / mika) ТУСДАА хадгалагддаг ба
// урсгал бүрт ЗӨВХӨН нэг эзэн бичнэ:
//
//	asset       — чөлөөлөлтийн үндсэн үнэлгээний байгууллага (land_acquisition.professional_org_id)
//	independent — тухайн нэгж талбарын хөндлөнгийн байгууллага (parcel.independent_org_id)
//	mika        — МИКА роль-той хэрэглэгч
//
// Дотоод ажилтан (админ, ахлах, хариуцсан мэргэжилтэн) үнэлгээний ДҮНГ
// өөрчлөхгүй — тэд зөвхөн харж, санхүү баталгаажуулна/буцаана.

// ValuationCaller — үнэлгээ бичих гэж буй дуудагч.
type ValuationCaller struct {
	// ValuationOrg — токены `valuation_org` тэмдэгтэй (үнэлгээний байгууллагын ажилтан).
	ValuationOrg bool
	// OrgID — дуудагчийн байгууллага (ValuationOrg үед).
	OrgID *ID
	// Mika — МИКА роль-той.
	Mika bool
}

// ValuationStreamOwners — нэгж талбарын үнэлгээний урсгалын эзэд.
type ValuationStreamOwners struct {
	ProfessionalOrgID *ID
	IndependentOrgID  *ID
}

// NormalizeValuationType — хоосон бол "asset" (түүхэн өгөгдөлтэй нийцнэ);
// мэдэгдэхгүй утга бол ok=false.
func NormalizeValuationType(t string) (string, bool) {
	switch t {
	case "", ValuationTypeAsset:
		return ValuationTypeAsset, true
	case ValuationTypeIndependent, ValuationTypeMika:
		return t, true
	}
	return "", false
}

// CanWriteValuationStream — дуудагч тухайн нэгж талбарын урсгалд бичиж болох эсэх.
// vType нь NormalizeValuationType-аар хэвийн болсон байх ёстой.
func CanWriteValuationStream(caller ValuationCaller, vType string, owners ValuationStreamOwners) bool {
	switch {
	case caller.ValuationOrg:
		if caller.OrgID == nil {
			return false
		}
		switch vType {
		case ValuationTypeAsset:
			return owners.ProfessionalOrgID != nil && *owners.ProfessionalOrgID == *caller.OrgID
		case ValuationTypeIndependent:
			return owners.IndependentOrgID != nil && *owners.IndependentOrgID == *caller.OrgID
		}
		return false
	case caller.Mika:
		return vType == ValuationTypeMika
	}
	// Дотоод ажилтан (админ, ахлах ч гэсэн) үнэлгээний дүн өөрчлөхгүй.
	return false
}

// ValuationOrgScope — үнэлгээний байгууллага нэг чөлөөлөлт дотор УНШИЖ болох
// үнэлгээний урсгалууд. Уншилтын дүрэм нь бичилтийнхтэй ИЖИЛ: байгууллага
// зөвхөн ӨӨРИЙН урсгалыг харна.
//
//	asset       — чөлөөлөлтийн үндсэн байгууллага бол БҮХ нэгж талбарынх
//	independent — зөвхөн хөндлөнгийнх нь болсон нэгж талбарынх
//	mika        — хэзээ ч биш
//
// Чөлөөлөлтийн гишүүнчлэл (RequireProfOrgMember) дангаараа хүрэлцэхгүй: нэг
// нэгж талбарын хөндлөнгийн байгууллага бусад нэгж талбарын болон үндсэн /
// МИКА-гийн урсгалын дүнг уншиж чаддаг байв.
type ValuationOrgScope struct {
	// MainOrg — дуудагчийн байгууллага чөлөөлөлтийн үндсэн байгууллага мөн.
	MainOrg bool
	// IndependentParcels — байгууллага хөндлөнгийнх нь болсон нэгж талбарын дугаарууд.
	IndependentParcels map[string]bool
}

// CanRead — тухайн нэгж талбарын (дугаараар) урсгалыг уншиж болох эсэх.
// Хоосон урсгал нь "asset" (NormalizeValuationType); танигдаагүй урсгал хаалттай.
func (s ValuationOrgScope) CanRead(parcelCode, valuationType string) bool {
	vType, ok := NormalizeValuationType(valuationType)
	if !ok {
		return false
	}
	switch vType {
	case ValuationTypeAsset:
		return s.MainOrg
	case ValuationTypeIndependent:
		return s.IndependentParcels[parcelCode]
	}
	return false
}

// ValuationOrgUploader — үнэлгээний байгууллагын оруулсан хавсралтын
// uploaded_by тэмдэг. Дотоод хэрэглэгчийн дугаартай андуурагдахгүй (тэр нь
// цэвэр тоо) бөгөөд байгууллагын аль ч ажилтан эзэмшигч гэж танигдана.
func ValuationOrgUploader(orgID ID) string {
	return "valuation_org:" + strconv.FormatInt(int64(orgID), 10)
}
