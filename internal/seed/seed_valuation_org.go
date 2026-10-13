package seed

import (
	"context"
	"errors"
	"log/slog"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedValuationOrgPassword — жишээ бүртгэлийн анхны нууц үг. Эдгээр нь ЗӨВХӨН
// хоосон санд үүсдэг туршилтын бүртгэл (доорх шалгуурыг үз).
const seedValuationOrgPassword = "Testpass123!"

// seedValuationOrgs — үнэлгээний байгууллага ба тэдний ажилтны нэвтрэх эрх.
//
// БҮГД appdb-д: байгууллага нь valuation_org_profile, ажилтан+нэвтрэх эрх нь
// valuation_org_employee (000024 миграц). sdplatform-д ЮУ Ч бичигдэхгүй.
var seedValuationOrgs = []struct {
	orgName   string
	username  string
	email     string
	lastName  string
	firstName string
}{
	{"Үнэлгээний Байгууллага 1", "valuation_org_1", "valuation.org1@example.mn", "Үнэлгээний", "Байгууллага 1"},
}

// SeedValuationOrgEmployees — үнэлгээний байгууллагын ЖИШЭЭ бүртгэл (байгууллага
// + нэвтрэх эрхтэй ажилтан) -ийг appdb-д үүсгэнэ.
//
// ХЭЗЭЭ АЖИЛЛАХ ВЭ: зөвхөн valuation_org_employee хүснэгт БҮРЭН ХООСОН үед.
// Бодит бүртгэлтэй санд хиймэл байгууллага/хэрэглэгч нэмэх ёсгүй — sd_user
// хоосон эсэхээр шалгадаг байсан хуучин дүрмийн appdb дахь хувилбар.
//
// authDB-г ОГТ ХӨНДӨХГҮЙ: эдгээр хэрэглэгч sdplatform-д бүртгэгддэггүй,
// нэвтрэлт нь user_source='valuation_org'-оор энэ хүснэгтээс шалгагдана
// (postgres.ValuationOrgLoginRepo).
//
// Идемпотент: байгууллага нэрээр, ажилтан нэвтрэх нэрээр шалгагдана.
func SeedValuationOrgEmployees(ctx context.Context, mainDB *pgxpool.Pool) error {
	var any bool
	if err := mainDB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM valuation_org_employee)`).Scan(&any); err != nil {
		return err
	}
	if any {
		slog.Info("үнэлгээний байгууллагын seed алгасагдав: valuation_org_employee хоосон биш")
		return nil
	}

	hashed, err := hash.Password(seedValuationOrgPassword)
	if err != nil {
		return err
	}

	for _, o := range seedValuationOrgs {
		var orgID domain.ID
		err := mainDB.QueryRow(ctx,
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
		// $2 нь INSERT-ийн баганад (varchar) БОЛОН lower($2)-д (text) хоёуланд
		// нь ордог тул ил cast ЗААВАЛ: эс бөгөөс Postgres хоёр өөр төрөл
		// дүгнээд "inconsistent types deduced for parameter $2" (42P08) өгнө.
		if _, err := mainDB.Exec(ctx, `
			INSERT INTO valuation_org_employee (
				org_id, username, "password", email, last_name, first_name,
				position_name, is_active)
			SELECT $1, LEFT($2::text, 50), LEFT($3::text, 150), LEFT($4::text, 250),
			       LEFT($5::text, 250), LEFT($6::text, 250), LEFT($7::text, 250), true
			WHERE NOT EXISTS (
				SELECT 1 FROM valuation_org_employee
				WHERE lower(username) = lower($2::text)
			)`, orgID, o.username, hashed, o.email, o.lastName, o.firstName,
			domain.ValuationOrgDefaultPosition); err != nil {
			return err
		}
		slog.Info("valuation org login seeded (appdb)", "org", o.orgName, "username", o.username)
	}
	return nil
}
