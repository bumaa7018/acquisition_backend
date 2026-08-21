package seed

import (
	"context"
	"errors"
	"log/slog"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedValuationOrgs — SeedExternalValuationOrgUsers-ээр үүсдэг гадаад
// хэрэглэгчид ба тэдний харьяалагдах үнэлгээний байгууллага.
//
// Хоёр нь ХОЛБОГДООГҮЙ бол хэрэглэгч нэвтэрч чадах ч токенд org_id гарахгүй
// тул /prof маршрутууд 403 буцаана (RequireProfOrgMember-ийг үз). Энэ seed нь
// тэр холбоосыг тавина.
var seedValuationOrgs = []struct {
	orgName   string
	username  string
	lastName  string
	firstName string
}{
	{"Үнэлгээний Байгууллага 1", "valuation_org_1", "Үнэлгээний", "Байгууллага 1"},
	{"Үнэлгээний Байгууллага 2", "valuation_org_2", "Үнэлгээний", "Байгууллага 2"},
	{"Үнэлгээний Байгууллага 3", "valuation_org_3", "Үнэлгээний", "Байгууллага 3"},
}

// SeedValuationOrgEmployees — үнэлгээний байгууллагын бүртгэл (appdb) ба гадаад
// хэрэглэгчийн нэвтрэх эрх (authdb)-ийг холбоно.
//
// Хоёр САНГ хөнддөг тул Seeder-ийн нэг pool-д тохирохгүй — pool-уудыг шууд
// авна. Хүснэгтийн бүтэц 000004 миграцад; энд ЗӨВХӨН мөр нэмнэ.
//
// Идемпотент: хэрэглэгч аль хэдийн ямар нэг байгууллагад холбогдсон бол
// хөндөхгүй, байгууллага байвал дахин үүсгэхгүй. Гараар үүсгэсэн бүртгэлийг
// дарж бичихгүй.
func SeedValuationOrgEmployees(ctx context.Context, mainDB, authDB *pgxpool.Pool) error {
	for _, o := range seedValuationOrgs {
		var userID domain.ID
		err := authDB.QueryRow(ctx,
			`SELECT user_id FROM sdplatform.sd_user WHERE username = $1 AND deleted_at IS NULL`,
			o.username).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			// Гадаад хэрэглэгчийн seed алгасагдсан (sd_user хоосон биш байсан)
			// — бодит бүртгэлд хиймэл байгууллага үүсгэх шаардлагагүй.
			continue
		}
		if err != nil {
			return err
		}

		var orgID domain.ID
		err = mainDB.QueryRow(ctx,
			`SELECT id FROM valuation_org_profile WHERE name = $1`, o.orgName).Scan(&orgID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			if err := mainDB.QueryRow(ctx, `
				INSERT INTO valuation_org_profile (name, short_name, is_active)
				VALUES ($1, $1, true)
				RETURNING id`, o.orgName).Scan(&orgID); err != nil {
				return err
			}
		case err != nil:
			return err
		}

		// register_no-г хоосон үлдээнэ: uq_valuation_org_employee_register нь
		// хэсэгчилсэн индекс тул хоосон регистр давхардлын шалгалтад орохгүй.
		if _, err := mainDB.Exec(ctx, `
			INSERT INTO valuation_org_employee (org_id, user_id, last_name, first_name, position_name, is_active)
			SELECT $1, $2, $3, $4, $5, true
			WHERE NOT EXISTS (
				SELECT 1 FROM valuation_org_employee WHERE user_id = $2
			)`, orgID, userID, o.lastName, o.firstName, domain.ValuationOrgDefaultPosition); err != nil {
			return err
		}
		slog.Info("valuation org employee linked", "org", o.orgName, "username", o.username)
	}
	return nil
}
