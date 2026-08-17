package seed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// SeedValuationOrgRegistry — professional_org рольтой БАЙГАА хэрэглэгч бүрийг
// үнэлгээний байгууллагын бүртгэлтэй холбоно.
//
// ЯАГААД ЗААВАЛ: хандалтын шалгалт хэрэглэгчийн ID-аас БАЙГУУЛЛАГЫН ID
// (JWT-ийн org_id) руу шилжсэн. org_id нь ажилтны бүртгэлээр тогтоогддог
// (sd_user → hr_employee → hr_department.organization) тул ажилтны бүртгэлгүй
// хуучин мэргэжлийн байгууллагын хэрэглэгч ямар ч чөлөөлөлт харахгүй болно.
// Энэ алхам тэдгээрт байгууллага + үндсэн нэгж + ажилтны бүртгэлийг үүсгэж
// өгснөөр өмнөх хэрэглэгчид хэвийн үргэлжлүүлэн ажиллана.
//
// Idempotent: аль хэдийн байгууллагатай хэрэглэгчийг алгасна.
func (s *Seeder) SeedValuationOrgRegistry(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `
		SELECT u.user_id,
		       COALESCE(NULLIF(TRIM(COALESCE(u.firstname, '') || ' ' || COALESCE(u.lastname, '')), ''), u.username),
		       COALESCE(u.username, ''),
		       COALESCE(u.email, '')
		FROM sdplatform.sd_user u
		JOIN sdplatform.sd_user_role ur ON ur.user_id = u.user_id
		JOIN sdplatform.sd_role r ON r.role_id = ur.role_id
		WHERE u.deleted_at IS NULL
		  AND `+roleLevelSQL+` = 'professional_org'
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.hr_employee e
			WHERE e.user_id = u.user_id AND COALESCE(e.is_active, 1) <> 0
		  )
		GROUP BY u.user_id, u.firstname, u.lastname, u.username, u.email`)
	if err != nil {
		return err
	}

	type pending struct {
		userID   domain.ID
		orgName  string
		username string
		email    string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.userID, &p.orgName, &p.username, &p.email); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(todo) == 0 {
		return nil
	}

	for _, p := range todo {
		if err := s.seedOneValuationOrg(ctx, p.userID, p.orgName, p.username, p.email); err != nil {
			return err
		}
	}
	slog.Info("valuation org registry seeded", "count", len(todo))
	return nil
}

// roleLevelSQL — domain.Role.Level()-ийн SQL хувилбар ("land_professional_org"
// → "professional_org").
const roleLevelSQL = `CASE
	WHEN COALESCE(r.role_type_code, '') <> '' AND r."name" LIKE r.role_type_code || '\_%'
	THEN substring(r."name" from length(r.role_type_code) + 2)
	ELSE r."name"
END`

func (s *Seeder) seedOneValuationOrg(ctx context.Context, userID domain.ID, orgName, username, email string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()

	// Ижил нэртэй байгууллага байвал түүнийг дахин ашиглана (idempotent).
	var orgID domain.ID
	err = tx.QueryRow(ctx,
		`SELECT id FROM sdplatform.sd_organization WHERE "name" = LEFT($1, 250) LIMIT 1`, orgName).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx,
			`INSERT INTO sdplatform.sd_organization ("name") VALUES (LEFT($1, 250)) RETURNING id`, orgName).
			Scan(&orgID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// valuation_org_profile нь АППЫН сангийн (appdb) хүснэгт — sdplatform-ынх
	// биш. Тиймээс authdb-ийн tx-ээр биш, аппын холболтоор бичнэ.
	if _, err := s.appDB.Exec(ctx, `
		INSERT INTO valuation_org_profile (organization_id, is_active, created_at, updated_at)
		VALUES ($1, true, $2, $2)
		ON CONFLICT (organization_id) DO NOTHING`, orgID, now); err != nil {
		return err
	}

	var deptID domain.ID
	err = tx.QueryRow(ctx,
		`SELECT department_id FROM sdplatform.hr_department WHERE organization = $1 ORDER BY department_id LIMIT 1`, orgID).
		Scan(&deptID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, `
			INSERT INTO sdplatform.hr_department (organization, "name", created_at, updated_at)
			VALUES ($1, LEFT($2, 250), $3, $3) RETURNING department_id`,
			orgID, domain.ValuationOrgDepartmentName, now).Scan(&deptID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var positionID domain.ID
	err = tx.QueryRow(ctx,
		`SELECT position_id FROM sdplatform.hr_position WHERE "name" = LEFT($1, 250) ORDER BY position_id LIMIT 1`,
		domain.ValuationOrgPositionName).Scan(&positionID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, `
			INSERT INTO sdplatform.hr_position ("name", created_at, updated_at)
			VALUES (LEFT($1, 250), $2, $2) RETURNING position_id`,
			domain.ValuationOrgPositionName, now).Scan(&positionID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// bs_person.person_register нь NOT NULL бөгөөд 10 тэмдэгтээр хязгаарлагдана.
	// Хуучин хэрэглэгчийн жинхэнэ регистр бидэнд байхгүй тул давхардахгүй
	// орлуулагч утга үүсгэнэ (админ дараа нь дэлгэцээс засна).
	register := fmt.Sprintf("U%09d", userID)
	var personID domain.ID
	err = tx.QueryRow(ctx,
		`SELECT person_id FROM base.bs_person WHERE person_register = $1`, register).Scan(&personID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, `
			INSERT INTO base.bs_person ("type", "name", first_name, person_register, email_address, created_at, updated_at)
			VALUES (1, LEFT($1, 250), NULL, $2, LEFT(NULLIF($3, '')::text, 256), $4, $4)
			RETURNING person_id`, orgName, register, email, now).Scan(&personID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.hr_employee (
			person_id, department_id, position_id, user_id, lastname, email,
			is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, LEFT($5, 250), LEFT(NULLIF($6, '')::text, 250), 1, $7, $7)`,
		personID, deptID, positionID, userID, orgName, email, now); err != nil {
		return err
	}

	slog.Info("valuation org linked to legacy user", "username", username, "org", orgName)
	return tx.Commit(ctx)
}
