package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/email"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

// AuthService — нэвтрэлт. Хэрэглэгч ҮҮСГЭХ үйлдэл ЭНД БАЙХГҮЙ: дотоод систем
// тул аккаунтыг зөвхөн эрх бүхий ажилтан UserService.Create (`users:create`)
// -ээр үүсгэнэ.
type AuthService interface {
	// Login — дотоод хэрэглэгч (authdb). LoginProfessionalOrg-оос ТУСДАА.
	Login(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error)
	// LoginProfessionalOrg — мэргэжлийн байгууллагын ажилтан (appdb).
	LoginProfessionalOrg(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ForgotPassword(ctx context.Context, emailAddr string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

type LoginRequest struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
}

type authService struct {
	userRepo repository.UserRepository
	// loginSources — нэвтрэлт шалгах эх сурвалжууд, ДАРААЛАЛТАЙ.
	//
	// [0] нь ҮРГЭЛЖ authdb (sd_user). Дараа нь appdb дээрх бүртгэлүүд
	// (үнэлгээний байгууллагын ажилтан). Хоёр сангийн user_id мөргөлддөг тул
	// олдсон эх сурвалжийн тэмдэг токенд бичигдэж, дараагийн бүх лавлагаа
	// ТЭР тэмдгээр чиглүүлэгдэнэ (sourceFor-ыг үз).
	loginSources        []repository.LoginUserRepository
	pwdResetRepo        repository.PasswordResetRepository
	tokenRevocationRepo repository.TokenRevocationRepository
	jwtManager          *jwt.Manager
	emailSender         email.Sender
	resetBaseURL        string
	loginLimiter        *loginLimiter
}

func NewAuthService(
	userRepo repository.UserRepository,
	pwdResetRepo repository.PasswordResetRepository,
	jwtManager *jwt.Manager,
	emailSender email.Sender,
	resetBaseURL string,
	optionalRepos ...any,
) AuthService {
	var revocationRepo repository.TokenRevocationRepository
	// authdb ҮРГЭЛЖ эхэнд — өмнөх зан төлөв хэвээр үлдэнэ.
	sources := []repository.LoginUserRepository{userRepo}
	for _, repo := range optionalRepos {
		switch v := repo.(type) {
		case repository.TokenRevocationRepository:
			revocationRepo = v
			jwtManager.SetRevocationChecker(revocationRepo)
		case repository.LoginUserRepository:
			sources = append(sources, v)
		}
	}
	return &authService{
		userRepo:            userRepo,
		loginSources:        sources,
		pwdResetRepo:        pwdResetRepo,
		tokenRevocationRepo: revocationRepo,
		jwtManager:          jwtManager,
		emailSender:         emailSender,
		resetBaseURL:        resetBaseURL,
		loginLimiter:        newLoginLimiter(),
	}
}

// orgIDFor — хэрэглэгчийн харьяа ҮНЭЛГЭЭНИЙ байгууллага (токены org_id).
//
// Энэ нэхэмжлэл нь land_acquisition.professional_org_id / parcel.
// independent_org_id-тай тулгагддаг ба тэдгээр нь ГАДААД талуудын сан дахь
// sd_organization.id-г заадаг. Иймд лавлагаа зөвхөн ТЭР сангаас явах ёстой.
//
// ЯАГААД ЭНЭ ХАТУУ ЗААГ ХЭРЭГТЭЙ: auth ба main сан тус тусын serial4
// дараалалтай тул ID нь ХОЁР ТҮВШИНД мөргөлддөг —
//   - user_id: authdb-ийн admin(3) ↔ appdb-ийн гадаад ажилтан(3);
//   - organization.id: authdb-ийн "Нийслэлийн Газрын алба"(1) ↔ appdb-ийн
//     "Үнэлгээний Байгууллага 3"(1).
//
// Дотоод ажилтны байгууллагыг энэ нэхэмжлэлд бичвэл тэр нь өөр байгууллагын
// чөлөөлөлтөд тохирч, хөндлөнгийн хандалт үүсгэнэ. Тиймээс дотоод (auth)
// хэрэглэгчид org_id ОГТ олгогдохгүй — тэдний хандалт роль/эрхээр шийдэгддэг.
//
// Тогтоох боломжгүй бол nil: нэвтрэлт ЗОГСОХГҮЙ, зүгээр л org_id-гүй токен.
//
// ХОЁР ЗАМ бий:
//  1. ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН ажилтан (appdb.valuation_org_employee) — РОЛЬ
//     ШАЛГАХГҮЙ. Эдгээр хэрэглэгч системийн үндсэн хэрэглэгчээс тусдаа явдаг
//     тул sdplatform-ийн роль/эрх тэдэнд ямар ч үүрэг хүлээхгүй; токенд
//     нэмээд `valuation_org` тэмдэг тавигдаж, хандалт нь ЗӨВХӨН түүгээр
//     хязгаарлагдана (middleware.ValuationOrgScope-ийг үз).
//  2. mika / finance_specialist зэрэг РОЛЬ-той гадаад хянагчид — HR бүтцээр
//     (өмнөх зан төлөв хэвээр).
//
// externalOrgRoles — 2-р замд org_id олгогдох ролиуд. Хэрэглэгч бүр нэг санд
// байдаг болсон тул "аль репоос уншсан" гэдгээр нь ялгах боломжгүй; ДОТООД
// ажилтанд org_id олгогдохгүй байх шаардлагыг РОЛЬ-оор хангана.
var externalOrgRoles = map[string]bool{
	"professional_org":   true,
	"mika":               true,
	"finance_specialist": true,
	"Мэргэжлийн байгууллага": true,
	"МИКА": true,
	"Санхүүгийн мэргэжилтэн": true,
	"Санхүү": true,
}

// orgContextFor — токены org_id БОЛОН `valuation_org` тэмдэг.
//
// Лавлагаа нь хэрэглэгчийг ОЛСОН эх сурвалжаар явна: хоёр сангийн user_id
// мөргөлддөг тул өөр репогоор асуувал өөр хүний байгууллага гарч ирнэ.
func (s *authService) orgContextFor(ctx context.Context, src repository.LoginUserRepository, roles []string, userID domain.ID) (*domain.ID, bool) {
	// 1. Үнэлгээний байгууллагын ажилтан — роль үл хамаарна.
	orgID, err := src.ValuationOrgIDForUser(ctx, userID)
	if err != nil {
		slog.Warn("valuation org lookup for token failed", "user_id", userID, "error", err)
	} else if orgID != nil {
		return orgID, true
	}

	// 2. Ролийн замаар (mika / санхүү) — HR бүтцээр.
	external := false
	for _, r := range roles {
		if externalOrgRoles[r] {
			external = true
			break
		}
	}
	if !external {
		return nil, false
	}
	orgID, err = src.OrgIDForUser(ctx, userID)
	if err != nil {
		slog.Warn("org lookup for token failed", "user_id", userID, "error", err)
		return nil, false
	}
	return orgID, false
}

// Login — ДОТООД нэвтрэлт: хэрэглэгчийг ЗӨВХӨН authdb (sdplatform.sd_user)
// -аас шалгана. Мэргэжлийн байгууллага энд нэвтрэхгүй.
func (s *authService) Login(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error) {
	return s.loginFrom(ctx, req, domain.UserSourceAuth)
}

// LoginProfessionalOrg — МЭРГЭЖЛИЙН БАЙГУУЛЛАГЫН нэвтрэлт: ЗӨВХӨН appdb-ийн
// valuation_org_employee-ээс шалгана.
//
// Хоёр урсгалыг САЛГАСАН шалтгаан: бүртгэл нь өөр санд, өөр бүтэцтэй бөгөөд
// user_id нь хоёр санд МӨРГӨЛДДӨГ. Нэг эндпойнтоос "эхлээд тэндээс, олдохгүй
// бол наанаас" гэж хайвал нэг сангийн хэрэглэгчийн нэр нөгөөгийнхийг
// халхалж, аль бүртгэлээр нэвтэрснийг таамаглах боломжтой болно.
func (s *authService) LoginProfessionalOrg(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error) {
	return s.loginFrom(ctx, req, domain.UserSourceValuationOrg)
}

func (s *authService) loginFrom(ctx context.Context, req LoginRequest, source string) (*jwt.TokenPair, error) {
	// Brute-force хамгаалалт: цонхонд хэт олон амжилтгүй оролдлого хийсэн бол
	// нууц үг зөв эсэхийг ч шалгалгүй 429 буцаана (username-ээр, IP-ээс хамааралгүй).
	if s.loginLimiter.blocked(req.Username) {
		return nil, domain.ErrTooManyRequests
	}
	sourceRepo := s.sourceFor(source)
	if sourceRepo == nil {
		// Тухайн эх сурвалж холбогдоогүй (жишээ нь appdb репо өгөгдөөгүй).
		return nil, domain.ErrInvalidCredentials
	}
	user, err := sourceRepo.GetActiveByLogin(ctx, req.Username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.loginLimiter.recordFailure(req.Username)
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if !hash.ComparePassword(user.Password, req.Password) {
		s.loginLimiter.recordFailure(req.Username)
		return nil, domain.ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, domain.ErrForbidden
	}
	// Амжилттай — тоолуурыг цэвэрлэнэ.
	s.loginLimiter.reset(req.Username)

	perms, err := sourceRepo.GetPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, err := sourceRepo.GetRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	orgID, isValuationOrg := s.orgContextFor(ctx, sourceRepo, roles, user.ID)
	tokens, err := s.jwtManager.GenerateTokenPair(jwt.UserInfo{
		ID:           user.ID,
		OrgID:        orgID,
		ValuationOrg: isValuationOrg,
		Source:       sourceRepo.Source(),
		Username:     user.Username,
		Email:        user.Email,
		FullName:     strings.TrimSpace(user.FirstName + " " + user.LastName),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Position:     user.Position,
		Roles:        roles,
		Permissions:  perms,
	})
	if err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredRevokedTokens(ctx); err != nil {
		return nil, err
	}
	return tokens, nil
}

// findLoginUser — нэвтрэх нэр ЭСВЭЛ имэйлээр хэрэглэгчийг олно.
//
// `login` нь имэйл байх ШААРДЛАГАГҮЙ — "user0101010" гэх мэт энгийн нэр ч
// хүчинтэй. Хайлт нь ЗӨВХӨН ИДЭВХТЭЙ хэрэглэгчийг олдог тул идэвхгүй бүртгэл
// "олдсонгүй" болж, гаднаас бүртгэл байгаа эсэхийг таамаглах боломжгүй болно.
// sourceFor — эх сурвалжийн тэмдэгт (токены `src`, эсвэл нэвтрэх урсгалын
// төрөл) харгалзах репо.
//
// Танигдаагүй тэмдэг бол ОГТ fallback хийхгүй: буруу санд хайвал өөр хүний
// бүртгэл рүү унах эрсдэлтэй.
func (s *authService) sourceFor(name string) repository.LoginUserRepository {
	if name == "" {
		name = domain.UserSourceAuth
	}
	for _, src := range s.loginSources {
		if src.Source() == name {
			return src
		}
	}
	return nil
}

func (s *authService) getUserForToken(ctx context.Context, source string, userID domain.ID) (*domain.User, repository.LoginUserRepository, error) {
	src := s.sourceFor(source)
	if src == nil {
		return nil, nil, domain.ErrInvalidToken
	}
	user, err := src.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return user, src, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.jwtManager.ValidateRefreshTokenContext(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	user, sourceRepo, err := s.getUserForToken(ctx, claims.UserSource(), claims.UserID)
	if err != nil {
		return nil, err
	}
	// Идэвхгүй болгосон хэрэглэгч refresh-ээр хандах эрхээ сунгаж болохгүй.
	if !user.IsActive {
		return nil, domain.ErrForbidden
	}
	perms, err := sourceRepo.GetPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, err := sourceRepo.GetRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	orgID, isValuationOrg := s.orgContextFor(ctx, sourceRepo, roles, user.ID)
	tokens, err := s.jwtManager.GenerateTokenPair(jwt.UserInfo{
		ID:           user.ID,
		OrgID:        orgID,
		ValuationOrg: isValuationOrg,
		Source:       sourceRepo.Source(),
		Username:     user.Username,
		Email:        user.Email,
		FullName:     strings.TrimSpace(user.FirstName + " " + user.LastName),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Position:     user.Position,
		Roles:        roles,
		Permissions:  perms,
	})
	if err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredRevokedTokens(ctx); err != nil {
		return nil, err
	}
	if err := s.revokeClaims(ctx, claims, "refresh"); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		if claims, err := s.jwtManager.ParseAccessToken(accessToken); err == nil {
			if err := s.revokeClaims(ctx, claims, "access"); err != nil {
				return err
			}
		}
	}
	if refreshToken != "" {
		if claims, err := s.jwtManager.ParseRefreshToken(refreshToken); err == nil {
			if err := s.revokeClaims(ctx, claims, "refresh"); err != nil {
				return err
			}
		}
	}
	return nil
}

// ForgotPassword нь reset token үүсгэж, email-р илгээнэ.
// Email байхгүй ч алдаа буцаахгүй (enumeration хамгаалалт).
func (s *authService) ForgotPassword(ctx context.Context, emailAddr string) error {
	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		// email олдохгүй эсвэл main DB-д байрлах гадны хэрэглэгч бол мэдэгдэхгүй.
		// Одоогийн password_reset_tokens нь authDB.users FK-тэй тул external
		// хэрэглэгчийн reset урсгалыг тусад нь хийх шаардлагатай.
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now()
	resetToken := &domain.PasswordResetToken{
		ID:     uuid.New(),
		UserID: user.ID,
		// Токеныг DB-д hash хэлбэрээр хадгална — DB задарсан ч токен шууд
		// ашиглагдахгүй. Хэрэглэгч рүү зөвхөн түүхий токен (email-ээр) очно.
		Token:     hashResetToken(token),
		ExpiresAt: now.Add(1 * time.Hour),
		Used:      false,
		CreatedAt: now,
	}
	if err := s.pwdResetRepo.Create(ctx, resetToken); err != nil {
		return err
	}

	resetLink := s.resetBaseURL + token
	return s.emailSender.SendPasswordReset(emailAddr, resetLink)
}

// hashResetToken — reset токеныг DB-д хадгалах/хайхад ашиглах SHA-256 hash.
func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	resetToken, err := s.pwdResetRepo.GetByToken(ctx, hashResetToken(token))
	if err != nil {
		return domain.ErrInvalidToken
	}
	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return domain.ErrExpiredToken
	}

	hashed, err := hash.Password(newPassword)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return err
	}
	// UpdatePassword-ээр бичнэ: Update() нь password баганад хүрдэггүй тул
	// түүгээр дамжуулбал нууц үг СОЛИГДОХГҮЙ (хэрэглэгч солигдсон гэж бодох).
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hashed); err != nil {
		return err
	}
	if err := s.pwdResetRepo.MarkUsed(ctx, resetToken.ID); err != nil {
		return err
	}
	// Нууц үг солигдсон тул өмнө олгогдсон бүх токеныг хүчингүй болгоно.
	return s.userRepo.InvalidateSessions(ctx, user.ID)
}

func (s *authService) cleanupExpiredRevokedTokens(ctx context.Context) error {
	if s.tokenRevocationRepo == nil {
		return nil
	}
	return s.tokenRevocationRepo.DeleteExpired(ctx)
}

func (s *authService) revokeClaims(ctx context.Context, claims *jwt.Claims, tokenType string) error {
	if s.tokenRevocationRepo == nil || claims == nil || claims.ID == "" || claims.ExpiresAt == nil {
		return nil
	}
	return s.tokenRevocationRepo.RevokeToken(ctx, &domain.RevokedToken{
		JTI:       claims.ID,
		UserID:    claims.UserID,
		TokenType: tokenType,
		ExpiresAt: claims.ExpiresAt.Time,
		RevokedAt: time.Now(),
	})
}
