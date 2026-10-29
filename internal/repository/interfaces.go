package repository

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// LoginUserRepository — НЭВТРЭЛТЭД шаардлагатай ХАМГИЙН БАГА багц.
//
// Хэрэглэгчийн бүртгэл хоёр санд амьдарна: sdplatform.sd_user (authdb) ба
// appdb.valuation_org_employee. AuthService тэдгээрийг ЗӨВХӨН энэ интерфейсээр
// хардаг тул эх сурвалж нэмэхэд нэвтрэлтийн логик өөрчлөгдөхгүй.
//
// Source() нь токенд бичигдэх тэмдэг (domain.UserSource*) — хоёр сангийн
// user_id мөргөлддөг тул хэрэглэгчийг ID-гаар хайхад ЗААВАЛ хамт хэрэглэнэ.
type LoginUserRepository interface {
	Source() string
	GetActiveByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.User, error)
	GetRoles(ctx context.Context, userID domain.ID) ([]string, error)
	GetPermissions(ctx context.Context, userID domain.ID) ([]string, error)
	OrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error)
	ValuationOrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error)
	UpdatePassword(ctx context.Context, id domain.ID, hashed string) error
	InvalidateSessions(ctx context.Context, userID domain.ID) error
}

type UserRepository interface {
	// Source — domain.UserSource* (LoginUserRepository-г үз).
	Source() string
	Create(ctx context.Context, user *domain.User) error
	CreateWithEmployee(ctx context.Context, user *domain.User, employee *domain.Employee, person *domain.Person) error
	GetByID(ctx context.Context, id domain.ID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	// GetActiveByLogin — нэвтрэх нэр ЭСВЭЛ имэйлээр хайх, ЗӨВХӨН идэвхтэй
	// хэрэглэгч. Нэвтрэлтэд ҮҮНИЙГ хэрэглэнэ — GetByUsername/GetByEmail нь
	// идэвхгүй хэрэглэгчийг ч олдог тул (давхардлын шалгалт, дахин
	// идэвхжүүлэлтэд шаардлагатай) нэвтрэлтэд тохирохгүй.
	GetActiveByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByEmployeeID(ctx context.Context, employeeID domain.ID) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	// UpdatePassword — Update() нь password баганыг хөндөхгүй тул нууц үг
	// шинэчлэхэд тусад нь хэрэглэнэ.
	UpdatePassword(ctx context.Context, id domain.ID, hashed string) error
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error)
	GetPermissions(ctx context.Context, userID domain.ID) ([]string, error)
	GetRoles(ctx context.Context, userID domain.ID) ([]string, error)
	// OrgIDForUser — хэрэглэгчийн харьяа байгууллага (JWT-ийн org_id).
	// Эх сурвалж бүр (auth/main) тусдаа serial4 дараалалтай тул user_id
	// мөргөлддөг — иймд лавлагаа нь хэрэглэгчийг ОЛСОН репогоор явах ёстой.
	OrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error)
	// ValuationOrgIDForUser — хэрэглэгч ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН идэвхтэй
	// ажилтан бол тухайн байгууллагын id (valuation_org_profile.id), эс бөгөөс
	// nil. HR-ийн бүтэц рүү fallback ХИЙХГҮЙ — иймд буцах ID нь ҮРГЭЛЖ
	// professional_org_id / independent_org_id-тай ижил дараалалынх.
	//
	// Токены `valuation_org` нэхэмжлэл үүн дээр тогтоогддог ба тэр нэхэмжлэл нь
	// эдгээр хэрэглэгчийн ЦОРЫН ГАНЦ эрхийн эх сурвалж (роль хэрэглэхгүй).
	ValuationOrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error)
	// InvalidateSessions — тухайн хэрэглэгчийн одоо хүчинтэй бүх токеныг
	// (энэ агшингаас өмнө олгогдсоныг) хүчингүй болгоно.
	InvalidateSessions(ctx context.Context, userID domain.ID) error
	// IsSessionValid — тухайн issuedAt хугацаанд олгогдсон токен хэрэглэгчийн
	// эх цэгээс хойш байвал (буюу эх цэг тавиагүй бол) true.
	//
	// source нь ТОКЕНООС ирнэ (domain.UserSource*). user_token_epoch нь
	// (user_source, user_id)-ээр түлхүүрлэгддэг ба хоёр сангийн user_id
	// мөргөлддөг тул эх сурвалжгүйгээр өөр хэрэглэгчийн эх цэг шалгагдана.
	IsSessionValid(ctx context.Context, source string, userID domain.ID, issuedAt time.Time) (bool, error)
}

type OrganizationRepository interface {
	ListOrganizations(ctx context.Context, filter domain.HRFilter) ([]*domain.Organization, int64, error)
	GetOrganizationByID(ctx context.Context, id domain.ID) (*domain.Organization, error)
	CreateOrganization(ctx context.Context, org *domain.Organization) error
	UpdateOrganization(ctx context.Context, org *domain.Organization) error
	DeleteOrganization(ctx context.Context, id domain.ID) error
}

type DepartmentRepository interface {
	ListDepartments(ctx context.Context, filter domain.HRFilter) ([]*domain.Department, int64, error)
	GetDepartmentByID(ctx context.Context, id domain.ID) (*domain.Department, error)
	CreateDepartment(ctx context.Context, dep *domain.Department) error
	UpdateDepartment(ctx context.Context, dep *domain.Department) error
	DeleteDepartment(ctx context.Context, id domain.ID) error
}

type PositionRepository interface {
	ListPositions(ctx context.Context, filter domain.HRFilter) ([]*domain.Position, int64, error)
	GetPositionByID(ctx context.Context, id domain.ID) (*domain.Position, error)
	CreatePosition(ctx context.Context, pos *domain.Position) error
	UpdatePosition(ctx context.Context, pos *domain.Position) error
	DeletePosition(ctx context.Context, id domain.ID) error
}

type PersonRepository interface {
	ListPersons(ctx context.Context, filter domain.HRFilter) ([]*domain.Person, int64, error)
	GetPersonByID(ctx context.Context, id domain.ID) (*domain.Person, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id domain.ID) error
}

type EmployeeRepository interface {
	ListEmployees(ctx context.Context, filter domain.HRFilter) ([]*domain.Employee, int64, error)
	GetEmployeeByID(ctx context.Context, id domain.ID) (*domain.Employee, error)
	CreateEmployee(ctx context.Context, employee *domain.Employee) error
	CreateEmployeeWithPerson(ctx context.Context, employee *domain.Employee, person *domain.Person) error
	UpdateEmployee(ctx context.Context, employee *domain.Employee) error
	DeleteEmployee(ctx context.Context, id domain.ID) error
}

// ValuationOrgRepository — app-owned мэргэжлийн байгууллага БА түүний
// ажилтнууд (valuation_org_profile / valuation_org_employee).
//
// Create/Update-ийн emps нь nil бол ажилтнуудад ХҮРЭХГҮЙ (зөвхөн байгууллагын
// мэдээлэл засварлана). nil биш бол жагсаалт нь БҮРЭН төлөвийг илэрхийлнэ:
// ороогүй ажилтан идэвхгүй болно.
type ValuationOrgRepository interface {
	List(ctx context.Context, filter domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error)
	Create(ctx context.Context, org *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error
	Update(ctx context.Context, org *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error
	Delete(ctx context.Context, id domain.ID) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id domain.ID) (*domain.Role, error)
	GetByName(ctx context.Context, name string) (*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context) ([]*domain.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error
	RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error
	SetMenuPermissions(ctx context.Context, roleID, menuID domain.ID, permissionIDs []domain.ID) error
	RemoveMenuPermissions(ctx context.Context, roleID, menuID domain.ID) error
	AssignToUser(ctx context.Context, userID, roleID domain.ID) error
	RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error
}

type PermissionRepository interface {
	Create(ctx context.Context, p *domain.Permission) error
	GetByID(ctx context.Context, id domain.ID) (*domain.Permission, error)
	GetByName(ctx context.Context, name string) (*domain.Permission, error)
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context) ([]*domain.Permission, error)
}

// MenuRepository — эрхийн 1-р давхарга: цэс харагдах эсэх (sd_menu / sd_menu_role).
type MenuRepository interface {
	List(ctx context.Context) ([]*domain.Menu, error)
	ListByRole(ctx context.Context, roleID domain.ID) ([]*domain.Menu, error)
	ListByUser(ctx context.Context, userID domain.ID) ([]*domain.Menu, error)
	ListPermissions(ctx context.Context, menuID domain.ID) ([]domain.Permission, error)
	AttachToRole(ctx context.Context, roleID, menuID domain.ID) error
	DetachFromRole(ctx context.Context, roleID, menuID domain.ID) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type TokenRevocationRepository interface {
	RevokeToken(ctx context.Context, token *domain.RevokedToken) error
	IsTokenRevoked(ctx context.Context, jti string) (bool, error)
	DeleteExpired(ctx context.Context) error
}

// PlanStoreRepository — appdb-ийн plan хүснэгт. Энэ нь төлөвлөгөөний ЦОРЫН
// ГАНЦ хадгалах ба унших байрлал: дундын сервисээс (middleware /plan/project)
// татсан мэдээллийг л агуулах бөгөөд хайх/сонгох үйлдэл эндээс уншина.
// authdb-д plan хүснэгт БАЙХГҮЙ.
type PlanStoreRepository interface {
	Upsert(ctx context.Context, plan *domain.Plan) (*domain.Plan, error)
	GetByCode(ctx context.Context, planCode string) (*domain.Plan, error)
	Suggest(ctx context.Context, q string) ([]*domain.Plan, error)
}

type LandLookupRepository interface {
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error)
	CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error)
	UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error)
	DeleteAcquisitionCategory(ctx context.Context, id int) error
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
}

type LandAcquisitionWriteRepository interface {
	Create(ctx context.Context, acq *domain.LandAcquisition) error
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	// ListFilterOptions — dropdown-д зориулсан хөнгөн жагсаалт (List-ийн
	// parcel COUNT(*) / join-гүй). Дэлгэрэнгүйг domain.LandAcquisitionOption.
	ListFilterOptions(ctx context.Context) ([]*domain.LandAcquisitionOption, error)
	// ListAcquisitionDistricts — чөлөөлөлт бүртгэгдсэн сум/дүүргүүд
	// (land_acquisition_au-аас давхардалгүй). Шүүлтүүрийн сонголтод.
	ListAcquisitionDistricts(ctx context.Context) ([]domain.AU2Option, error)
	Suggest(ctx context.Context, q string, status int) ([]*domain.LandAcquisition, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error)
	// FindAU2Coverage — чөлөөлөх хил ямар сум/дүүрэгтэй (au2) давхцаж, аль
	// нь түүнийг бүрэн бүрхэж байгааг тооцно (дамналт шалгахад).
	FindAU2Coverage(ctx context.Context, geomWKT string) (*domain.AU2Coverage, error)
	// ComputeAreaM2 — чөлөөлөх хилийн талбайг м²-ээр PostGIS-аар тооцно.
	ComputeAreaM2(ctx context.Context, geomWKT string) (float64, error)
	// CheckContainment — geomWKT нь containerWKT дотор БҮРЭН багтсан эсэх,
	// гадна гарсан талбай ба container-ийн талбайг нэг дуудалтаар тооцно.
	CheckContainment(ctx context.Context, geomWKT, containerWKT string) (*domain.GeomContainment, error)
	CreateAU(ctx context.Context, au *domain.LandAcquisitionAU) error
	// InsertParcelCodes — дугаар + БҮРТГЭЛИЙН баганаар (геометр, талбай,
	// зориулалт, хаяг, эрхийн хугацаа) нэгж талбарын мөр үүсгэнэ. Эрх зүйн
	// дэлгэрэнгүй нь дараа нь SyncParcel-ээр татагдана. Дугаар бүрийг
	// бүртгэхийн өмнө байгаа эсэхийг шалгаж, давхардлыг үүсгэхгүй; БАЙГАА
	// мөрийн бүртгэлийн баганыг ГУС-аас ИРСЭН утгаар нэг бүрчлэн ШИНЭЧИЛНЭ
	// (хэрэглэгчийн өгөгдөл болон parcel_detail хөндөгдөхгүй).
	InsertParcelCodes(ctx context.Context, acquisitionID uuid.UUID, parcels []*domain.SourceParcel, acqWKT string) (*domain.ParcelDiscoveryResult, error)
	// ListParcelRefs — чөлөөлөлтийн нэгж талбаруудын id/дугаар/төлөв.
	ListParcelRefs(ctx context.Context, acquisitionID uuid.UUID) ([]domain.ParcelRef, error)
	// PurgeParcels — нэгж талбаруудыг хүү мөрүүдтэйгээ БҮРМӨСӨН устгана.
	// Устгасан мөрийн тоо болон хадгалалтаас устгах файлын URL-уудыг буцаана.
	// Хил солиход АВТОМАТААР ДУУДАГДАХГҮЙ (хилээс гарсан талбар хэвээр үлдэнэ).
	PurgeParcels(ctx context.Context, acquisitionID uuid.UUID, parcelIDs []uuid.UUID, parcelCodes []string) (int, []string, error)
	// ListParcelsOutsideBoundary — чөлөөлөлтийн нэгж талбаруудаас boundaryWKT-тэй
	// ОГТ давхцахгүйг (геометртэйг нь) буцаана. Хил солихын өмнө харуулахад.
	ListParcelsOutsideBoundary(ctx context.Context, acquisitionID uuid.UUID, boundaryWKT string) ([]domain.BoundaryOutsideParcel, error)
	CreateBoundaryHistory(ctx context.Context, history *domain.LandAcquisitionBoundaryHistory) error
	ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error)
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error)
	GetParcelByCode(ctx context.Context, parcelCode string) (*domain.Parcel, error)
	GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error)
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	UpdateParcelAcquisitionGeom(ctx context.Context, parcelID uuid.UUID, acqWKT string) error
	// ApplyParcelGeometry — эх системээс ирсэн геометрийг хадгалж, чөлөөлөгдөх
	// талбайн хил/хэмжээ болон үлдэх талбайг тооцно.
	ApplyParcelGeometry(ctx context.Context, parcelID uuid.UUID, parcelWKT, acqWKT string) (*domain.ParcelGeometryResult, error)
	// FillParcelAUCodesFromGeometry — геометрээр харьяалах баг/хороог олж
	// аймаг/сум/багийн кодыг хадгална.
	FillParcelAUCodesFromGeometry(ctx context.Context, parcelID uuid.UUID) (domain.LandAcquisitionAU, error)
	SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *domain.ID) error
	SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *domain.ID) error
	UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error)
	// ReplaceParcelDetail — parcel_detail-ийг хуучин мөрүүдийг устгаж бүрэн шинэчилнэ.
	ReplaceParcelDetail(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error
	// ReplaceParcelHolders — нэгж талбарын ОЛОН эзэмшигчийг бүрэн шинэчилж,
	// үндсэн хүсэлт гаргагчийн right_holder.id-г буцаана.
	ReplaceParcelHolders(ctx context.Context, parcelID, acquisitionID uuid.UUID, appNo string, holders []*domain.ParcelHolder) (uuid.UUID, error)
	ListParcelHolders(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelHolder, error)
	// SetParcelPaymentRecipient — нөхөн төлбөр хүлээн авах эзэмшигчийг сонгоно
	// (нэгж талбар тутамд зөвхөн нэг).
	SetParcelPaymentRecipient(ctx context.Context, parcelID, parcelHolderID uuid.UUID) error
	// AddParcelRepresentative — итгэмжлэгдсэн төлөөлөгчийг эзэмшигчийн НЭГ
	// ТӨРӨЛ болгож бүртгэнэ; бүртгэмэгц шууд төлбөр авагч болно.
	AddParcelRepresentative(ctx context.Context, parcelID, acquisitionID uuid.UUID, holder *domain.ParcelHolder) (uuid.UUID, error)
	// DeleteParcelHolder — ЗӨВХӨН итгэмжлэгдсэн төлөөлөгчийн мөрийг устгана
	// (ГУС-аас татагдсан эзэмшигч sync дээр дахин үүсэх тул устгах утгагүй).
	DeleteParcelHolder(ctx context.Context, parcelID, parcelHolderID uuid.UUID) error
	// Барьцаа, шүүхийн шийдвэр — ГУС-аас татагдаж ОРЛУУЛАХ семантикаар хадгалагдана
	ReplaceParcelMortgages(ctx context.Context, parcelID, acquisitionID uuid.UUID, appNo string, items []*domain.ParcelMortgage) (int, error)
	ListParcelMortgages(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelMortgage, error)
	ReplaceParcelCourtDecisions(ctx context.Context, parcelID, acquisitionID uuid.UUID, appNo string, items []*domain.ParcelCourtDecision) (int, error)
	ListParcelCourtDecisions(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelCourtDecision, error)
	ReplaceParcelInvoices(ctx context.Context, parcelID, acquisitionID uuid.UUID, items []*domain.ParcelInvoice) (int, error)
	ListParcelInvoices(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelInvoice, error)
	// Газрын төлбөрийн бодолт, суурь үнийн хүчин зүйл — ГУС-аас татагдаж
	// БҮХЭЛДЭЭ орлуулагдана: бодолт нь бүсээр задардаг тул хэсэгчлэн upsert
	// хийвэл эх системд байхгүй болсон бүс үлдэж нийт төлбөр буруу гарна.
	ReplaceParcelFeeItems(ctx context.Context, parcelID, acquisitionID uuid.UUID, items []domain.ParcelFeeItem) (int, error)
	ListParcelFeeItems(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelFeeItem, error)
	ReplaceParcelBasePriceFactors(ctx context.Context, parcelID, acquisitionID uuid.UUID, items []domain.ParcelBasePriceFactor) (int, error)
	ListParcelBasePriceFactors(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelBasePriceFactor, error)
	UpsertParcelMonitorings(ctx context.Context, parcelID, acquisitionID uuid.UUID, items []*domain.ParcelMonitoring) (int, error)
	ListParcelMonitorings(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelMonitoring, error)
	// УБЕГ-ийн өмчлөлийн бүртгэл (ХУР/XYP). Мөрд тогтвортой түлхүүр байхгүй
	// тул нэгж талбарын мөрүүд БҮХЭЛДЭЭ орлуулагдана.
	ReplaceParcelOwnerships(ctx context.Context, parcelID, acquisitionID uuid.UUID, items []domain.ParcelOwnership) (int, error)
	ListParcelOwnerships(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelOwnership, error)
	// SetParcelDetailHolder — parcel_detail.holder_id-г үндсэн эзэмшигчид холбоно.
	SetParcelDetailHolder(ctx context.Context, parcelID, holderID uuid.UUID) error
	Update(ctx context.Context, acq *domain.LandAcquisition) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status int) error
	UpdateDecree(ctx context.Context, id uuid.UUID, number string, date *time.Time) error
	HasNonFinalParcels(ctx context.Context, acquisitionID uuid.UUID) (bool, error)
	// CountParcels — чөлөөлөлтөд бүртгэгдсэн нэгж талбарын тоо.
	CountParcels(ctx context.Context, acquisitionID uuid.UUID) (int, error)
	// SetParcelEstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээ (₮). Value nil → арилгана.
	// Өөрчлөлт бүр parcel_estimated_value_history-д мөн бичигдэнэ (нэг tx).
	SetParcelEstimatedValue(ctx context.Context, ch domain.ParcelEstimatedValueChange) error
	// ListParcelEstimatedValueHistory — өөрчлөлтийн түүх (шинэ → хуучин).
	ListParcelEstimatedValueHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID) ([]*domain.ParcelEstimatedValueHistory, error)
	// DetectParcelOverlaps — чөлөөлөлтийн нэгж талбаруудын БАЙРШЛЫН давхардлыг
	// ДАХИН тооцож parcel_overlap-д бичнэ (хуучин мөрүүд устгагдана). Буцаах нь
	// давхардалтай нэгж талбарын ТОО (хос биш).
	DetectParcelOverlaps(ctx context.Context, acquisitionID uuid.UUID) (int, error)
	// ListParcelOverlaps — НЭГ нэгж талбарын давхардлууд (дэлгэрэнгүйд харуулна).
	ListParcelOverlaps(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelOverlap, error)
	// DeleteParcelsOutsideBoundary — чөлөөлөх ХИЛД БАГТААГҮЙ нэгж талбаруудыг
	// устгана (зөвхөн "Шинэ" төлөвт автоматаар дуудагдана). Буцаах нь
	// устгагдсан нэгж талбарын дугаарууд.
	DeleteParcelsOutsideBoundary(ctx context.Context, acquisitionID uuid.UUID) ([]string, error)
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged string, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error
	DeleteAUs(ctx context.Context, acquisitionID uuid.UUID) error
	DeleteParcelsAndAUs(ctx context.Context, acquisitionID uuid.UUID) error
	// CheckAcquisitionDeletable — устгахыг зогсоох шалтгаануудыг (Чөлөөлсөн
	// нэгж талбар, баталгаажсан захирамж) нэг дуудалтаар тооцно.
	CheckAcquisitionDeletable(ctx context.Context, id uuid.UUID) (*domain.AcquisitionDeleteCheck, error)
	// ListAcquisitionFileURLs — чөлөөлөлтөд хамаарах ХАДГАЛАЛТ дээрх бүх
	// файлын зам (хавсралт, нэгж талбарын хавсралт, дроны зураг, хөрөнгийн
	// зураг, үнэлгээний тайлан). Устгасны дараа цэвэрлэхэд.
	ListAcquisitionFileURLs(ctx context.Context, id uuid.UUID) ([]string, error)
	// Delete — SOFT delete: мөрийг устгахгүй, deleted_at-ыг тэмдэглэнэ.
	// Уншилтууд deleted_at IS NULL-ээр шүүгддэг тул API-д харагдахаа болино.
	// CASCADE-ээр 23 хүснэгт цэвэрлэгдэхээс сэргийлнэ.
	Delete(ctx context.Context, id uuid.UUID, deletedBy string) error
}

type LandProgressRepository interface {
	ListProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	CreateProgress(ctx context.Context, p *domain.AcquisitionProgress) error
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error)
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
	// GetChronos — «Хронос» цаг хугацааны изометр зургийн өгөгдөл (acqIDs хоосон = бүгд).
	GetChronos(ctx context.Context, acqIDs []uuid.UUID) (*domain.ChronosData, error)
}

type LandAssetRepository interface {
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) error
	UpdateAsset(ctx context.Context, u *domain.AssetUpdate) (*domain.Asset, error)
	// DeleteAsset — allowedStreams хоосон бол хязгаарлалтгүй; эс бөгөөс мөрийн
	// valuation_type тэдгээрийн аль нэгтэй таарсан үед л устгана (МИКА өөр
	// байгууллагын үнэлгээг устгахаас сэргийлнэ).
	DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) (string, error)
	// GetAssetAcquisition — хөрөнгийн харьяалагдах чөлөөлөлт. Файлын түлхүүрийн
	// эрх шалгахад (asset-photo/<asset_id>/...) хэрэглэнэ. Мөр нь устсан
	// (үнэлгээ цуцлагдсан) бол архивын snapshot-оос олно.
	GetAssetAcquisition(ctx context.Context, assetID uuid.UUID) (uuid.UUID, error)
	// GetCompensationAcquisition — олговрын харьяалагдах чөлөөлөлт
	// (compensation-report/<comp_id>/...). GetAssetAcquisition-тэй ижилхэн
	// устсан мөрийг архивын snapshot-оос олно.
	GetCompensationAcquisition(ctx context.Context, compID uuid.UUID) (uuid.UUID, error)
	UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error
	ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error)
	UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error
	ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error)
	ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error)
	CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	DeleteAssetSpecType(ctx context.Context, id int) error
	ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error)
	CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	DeleteAssetCalcType(ctx context.Context, id int) error
	UpsertLandValuation(ctx context.Context, v *domain.LandValuation) error
	GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error)
	DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error
	ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error)
	ListValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.ValuationSectionNote, error)
	UpsertValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string, notes map[string]string, updatedBy string) error
}

type LandPaymentRepository interface {
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
}

type LandCompensationRepository interface {
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) error
	UpdateCompensation(ctx context.Context, c *domain.Compensation) error
	DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error
	GetCompensation(ctx context.Context, id uuid.UUID) (*domain.Compensation, error)
	ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	UpdateCompensationReport(ctx context.Context, id uuid.UUID, reportURL, reportName string) error
	UpdateAssetPhoto(ctx context.Context, acquisitionID, assetID uuid.UUID, photoURL, photoName string) (string, error)
	ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error)
	ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error)
	GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error)
}

type LandCompensationGrantRepository interface {
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error
}

type LandDocumentRepository interface {
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	AddAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, doc *domain.Document) error
	// Нийгэм, эдийн засгийн судалгаа — чөлөөлөлтөд НЭГ бүртгэл.
	// Байхгүй үед GetAcquisitionSocioSurvey нь ErrNotFound буцаана.
	GetAcquisitionSocioSurvey(ctx context.Context, acquisitionID uuid.UUID) (*domain.AcquisitionSocioSurvey, error)
	// UpsertAcquisitionSocioSurvey — мөрийг орлуулж, СОЛИГДСОН хуучин файлын
	// URL-ийг буцаана (дуудагч хадгалалтаас цэвэрлэнэ).
	UpsertAcquisitionSocioSurvey(ctx context.Context, s *domain.AcquisitionSocioSurvey) (oldFileURL string, err error)
	// Мэдэгдэх хуудсыг имэйлээр илгээсэн түүх — зөвхөн НЭМЭГДЭНЭ.
	AddParcelNoticeEmail(ctx context.Context, m *domain.ParcelNoticeEmail) error
	ListParcelNoticeEmails(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelNoticeEmail, error)
	// E-Mongolia-гаар хүргүүлсэн мэдэгдлийн түүх — зөвхөн НЭМЭГДЭНЭ.
	AddParcelEMongoliaNotice(ctx context.Context, m *domain.ParcelEMongoliaNotice) error
	ListParcelEMongoliaNotices(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelEMongoliaNotice, error)
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error
	// AddSourceParcelDocuments — эх системээс татсан хавсралтуудаас ЗӨВХОН
	// ШИНИЙГ бүртгэнэ (source_doc_id-ээр таних). Хадгалагдсан хавсралт
	// устгагдахгүй, дарагдахгүй. Шинээр бүртгэгдсэн тоог буцаана.
	AddSourceParcelDocuments(ctx context.Context, parcelID uuid.UUID, docs []*domain.Document) (int, error)
	DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error
	DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error
	GetDocument(ctx context.Context, documentID uuid.UUID) (*domain.Document, error)
	// GetParcelDocument — баримт ТУХАЙН нэгж талбарт холбогдсон эсэхийг
	// шалгаж уншина (IDOR-оос сэргийлнэ).
	GetParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) (*domain.Document, error)
	GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error)
	// GetFinanceDashboard — санхүүгийн хяналтын самбарын нэгтгэл (он/дүүрэг/хороо).
	GetFinanceDashboard(ctx context.Context, filter domain.FinanceDashboardFilter) (*domain.FinanceDashboard, error)
}

type LandAssigneeRepository interface {
	SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error
	GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error)
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID domain.ID) (bool, error)
	IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID domain.ID) (bool, error)
	// DecisionDraftParcelIDs — захирамжид идэвхтэй холбогдсон нэгж талбарын id.
	DecisionDraftParcelIDs(ctx context.Context, draftID uuid.UUID) ([]uuid.UUID, error)
	IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (bool, error)
	IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, orgID domain.ID) (bool, error)
	// OrgUserIDs — тухайн байгууллагын НЭВТРЭХ ЭРХТЭЙ идэвхтэй ажилтнуудын
	// хэрэглэгчийн ID-ууд. Байгууллагад мэдэгдэл илгээхэд хэрэглэнэ: ажил нь
	// нэг хүнд биш байгууллагад оноогддог тул бүх ажилтанд мэдэгдэнэ.
	OrgUserIDs(ctx context.Context, orgID domain.ID) ([]domain.ID, error)
}

type LandWorkflowRepository interface {
	ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error)
	CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error)
	DeleteParcelWorkflow(ctx context.Context, id int) error
	GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error)
	// UpdateParcelStatus — төлөв солих (түүхийн мөр хавсралттайгаа бичигдэнэ).
	UpdateParcelStatus(ctx context.Context, ch domain.ParcelStatusChange) error
	// GetParcelStatusRules — төлөвийн дүрэм (parcel_status-ийн тохиргоо).
	GetParcelStatusRules(ctx context.Context, statusID int) (*domain.ParcelStatusRules, error)
	// ParcelProgressLocked — нэгж талбарын ОДООГИЙН төлөв «Явц түгжинэ» эсэх.
	ParcelProgressLocked(ctx context.Context, parcelID uuid.UUID) (bool, error)
	// Шилжилтэд ЗААВАЛ байх хавсралт (тохиргоо ба нэгж талбарын төлөв).
	SetParcelWorkflowRequiredDocuments(ctx context.Context, workflowID int, documentTypeIDs []int) error
	ParcelStatusRequirements(ctx context.Context, acquisitionID, parcelID uuid.UUID, statusID int) ([]domain.ParcelStatusRequirement, error)
	InsertParcelStatusHistory(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy, reason string) error
	ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error)
	GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error)
	// TransitionValuationSubmission — submit | approve | return. Хавсралт нь
	// ЗӨВХӨН буцаалтад хамаарах ба ЗААВАЛ БИШ (domain.ValuationTransition).
	TransitionValuationSubmission(ctx context.Context, in domain.ValuationTransition) (*domain.ValuationSubmission, error)
	ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error)
	ListValuationSnapshots(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSnapshot, error)
	ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error)
	CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error)
	DeleteAcquisitionWorkflow(ctx context.Context, id int) error
}

type LandFundingRepository interface {
	// Дроны ортофото — GeoServer дээрх давхаргатай холбогдоно
	ListDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error)
	GetDroneImage(ctx context.Context, id uuid.UUID) (*domain.DroneImage, error)
	CreateDroneImage(ctx context.Context, img *domain.DroneImage) error
	UpdateDroneImageBounds(ctx context.Context, id uuid.UUID, minX, minY, maxX, maxY float64) error
	DeleteDroneImage(ctx context.Context, id uuid.UUID) error

	ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error)
	GetFundingSource(ctx context.Context, id uuid.UUID) (*domain.FundingSource, error)
	CreateFundingSource(ctx context.Context, fs *domain.FundingSource) error
	UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) error
	DeleteFundingSource(ctx context.Context, id uuid.UUID) error
}

type LandAuthorizedRepresentativeRepository interface {
	ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error)
	CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) error
	DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error
}

// LandAcquisitionRepository — main DB-д газар чөлөөлөлтын мэдээлэл бичнэ.
type LandAcquisitionRepository interface {
	LandLookupRepository
	LandAcquisitionWriteRepository
	LandProgressRepository
	LandAssetRepository
	LandPaymentRepository
	LandCompensationRepository
	LandCompensationGrantRepository
	LandDocumentRepository
	LandAssigneeRepository
	LandWorkflowRepository
	LandFundingRepository
	LandAuthorizedRepresentativeRepository
}

type ParcelStatusRepository interface {
	List(ctx context.Context) ([]*domain.ParcelStatus, error)
	GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error)
	Create(ctx context.Context, ps *domain.ParcelStatus) error
	Update(ctx context.Context, ps *domain.ParcelStatus) error
	Delete(ctx context.Context, id int) error
}

// DecisionOptionRepository — Ажлын төрөл / Төсөв сонголтын жагсаалт (ижил
// бүтэцтэй тул нэг интерфейсээр, хоёр өөр хэрэгжүүлэлтээр).
type DecisionOptionRepository interface {
	List(ctx context.Context) ([]*domain.DecisionOption, error)
	GetByID(ctx context.Context, id int) (*domain.DecisionOption, error)
	Create(ctx context.Context, o *domain.DecisionOption) error
	Update(ctx context.Context, o *domain.DecisionOption) error
	Delete(ctx context.Context, id int) error
}

type DecisionDraftRepository interface {
	List(ctx context.Context, f domain.DecisionDraftFilter) ([]*domain.DecisionDraft, int64, error)
	GetByID(ctx context.Context, id string) (*domain.DecisionDraft, error)
	Create(ctx context.Context, d *domain.DecisionDraft) error
	Update(ctx context.Context, d *domain.DecisionDraft) error
	// Confirm — doc != nil бол хавсралтыг захирамж ба түүнд холбогдсон бүх
	// нэгж талбарт НЭГ гүйлгээнд холбоно.
	Confirm(ctx context.Context, id, decreeNumber, decisionDate, note, actor string, doc *domain.Document) error
	// DecreeDocumentTypeID — «Захирамж» (decree) хавсралтын төрлийн id.
	DecreeDocumentTypeID(ctx context.Context) (int, error)
	// GetDocument — захирамжийн хавсралт; байхгүй бол (nil, nil).
	GetDocument(ctx context.Context, draftID string) (*domain.DecisionDraftDocument, error)
	Delete(ctx context.Context, id string) error
	SetAcquisition(ctx context.Context, id, acquisitionID string) error
	ListProgressHistory(ctx context.Context, draftID string) ([]*domain.DecisionDraftProgressHistory, error)
	AddProgressHistory(ctx context.Context, h *domain.DecisionDraftProgressHistory) error

	ListParcels(ctx context.Context, draftID string) ([]*domain.DecisionDraftParcel, error)
	ListByParcel(ctx context.Context, parcelUUID string) ([]*domain.DecisionDraftParcel, error)
	GetParcelInfo(ctx context.Context, parcelUUID string) (*domain.ParcelLinkInfo, error)
	LinkParcel(ctx context.Context, draftID, parcelUUID, fundingLinkID, actor string) (string, error)
	UnlinkParcel(ctx context.Context, draftID, linkID, actor string) error
	ActiveLinkOf(ctx context.Context, parcelUUID string) (string, error)

	ListFundingSourceOptions(ctx context.Context) ([]*domain.FundingSourceOption, error)
	CreateFundingSourceOption(ctx context.Context, o *domain.FundingSourceOption, note, actor string) error
	ListDraftFundingSources(ctx context.Context, draftID string) ([]*domain.DecisionDraftFundingLink, error)
	AddFundingSource(ctx context.Context, draftID, fundingSourceID string, amount float64, actor string) error
	RemoveFundingSource(ctx context.Context, draftID, linkID string) error
	FundingLinkBelongsTo(ctx context.Context, draftID, fundingLinkID string) (bool, error)
	FundingLinkInUse(ctx context.Context, fundingLinkID string) (bool, error)
	SetParcelFundingLink(ctx context.Context, draftID, linkID, fundingLinkID string) error
	// BindFundingSourceToAcquisition — нэгж талбар холбоход сонгосон эх үүсвэрийг
	// тухайн чөлөөлөлтийн санхүүжилт болгож хадгална.
	BindFundingSourceToAcquisition(ctx context.Context, fundingLinkID, acquisitionID string) error
}

// ParcelSurveyRepository — чөлөөлөгдсөн газрын судалгаа (parcel_survey).
// Нэгж талбарт НЭГ бүртгэл (нэгж талбар үүсэхэд trigger-ээр үүснэ); байхгүй
// үед GetByParcel / ClearByParcel нь ErrNotFound буцаана.
type ParcelSurveyRepository interface {
	ParcelExists(ctx context.Context, parcelID uuid.UUID) (bool, error)
	GetByParcel(ctx context.Context, parcelID uuid.UUID) (*domain.ParcelSurvey, error)
	Upsert(ctx context.Context, s *domain.ParcelSurvey) error
	// ClearByParcel — хариултыг хоосолно; мөр нь нэгж талбартайгаа үлдэнэ.
	ClearByParcel(ctx context.Context, parcelID uuid.UUID, clearedBy string) error
	List(ctx context.Context, filter domain.ParcelSurveyFilter) ([]*domain.ParcelSurveyRow, int64, error)
	Summary(ctx context.Context, filter domain.ParcelSurveyFilter) (*domain.ParcelSurveySummary, error)
}

// SettingRepository — системийн тохиргоо (app_setting): key → JSON текст.
// Мөр байхгүй бол Get нь domain.ErrNotFound буцаана ("тохируулаагүй").
type SettingRepository interface {
	Get(ctx context.Context, key string) (*domain.SettingRecord, error)
	Upsert(ctx context.Context, key, valueJSON, updatedBy string) error
	Delete(ctx context.Context, key string) error
}

type DocumentTypeRepository interface {
	List(ctx context.Context, target string) ([]*domain.DocumentType, error)
	GetByID(ctx context.Context, id int) (*domain.DocumentType, error)
	Create(ctx context.Context, dt *domain.DocumentType) error
	Update(ctx context.Context, dt *domain.DocumentType) error
	Delete(ctx context.Context, id int) error
}

// NotificationRepository — хэрэглэгчийн мэдэгдэл (main DB).
// Бүх унших/бичих үйлдэл recipient_id-аар шүүгдэж, хэрэглэгч зөвхөн
// өөрийн мэдэгдэлд ханддаг.
// source нь ЗААВАЛ (domain.UserSource*): recipientID нь authdb ба appdb хоёрт
// мөргөлддөг тул эх сурвалжгүйгээр өөр хэрэглэгчийн мэдэгдэл уншигдана.
type NotificationRepository interface {
	CreateBatch(ctx context.Context, ns []*domain.Notification) error
	ListByRecipient(ctx context.Context, source string, recipientID domain.ID, onlyUnread bool, page, pageSize int) ([]*domain.Notification, int64, error)
	UnreadCount(ctx context.Context, source string, recipientID domain.ID) (int64, error)
	MarkRead(ctx context.Context, source string, recipientID domain.ID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, source string, recipientID domain.ID) error
}

// AuditLogRepository — ажилтны бизнес үйлдлийн append-only лог.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error)
}

// ClientRequestRepository — бусад системрүү илгээсэн HTTP хүсэлтийн append-only лог.
type ClientRequestRepository interface {
	Create(ctx context.Context, r *domain.ClientRequest) error
	List(ctx context.Context, filter domain.ClientRequestFilter) ([]*domain.ClientRequest, int64, error)
}
