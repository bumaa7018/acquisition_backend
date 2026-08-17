package seed

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MainSeeder үндсэн DB-д жишээ өгөгдөл нэмнэ.
// Аймаг → Сум → Баг → Газар чөлөөлөлт → Нэгж талбар → Эзэмшигч → Эрх зүй → Хөрөнгө
type MainSeeder struct {
	db *pgxpool.Pool
}

func NewMainSeeder(db *pgxpool.Pool) *MainSeeder {
	return &MainSeeder{db: db}
}

func (s *MainSeeder) Seed(ctx context.Context) error {
	slog.Info("seeding main database...")

	if err := s.seedAUs(ctx); err != nil {
		return err
	}

	acqID, err := s.seedLandAcquisition(ctx)
	if err != nil {
		return err
	}

	if err := s.seedLandAcquisitionAU(ctx, acqID); err != nil {
		return err
	}

	holderID, err := s.seedLandHolder(ctx)
	if err != nil {
		return err
	}

	parcelID, err := s.seedParcel(ctx, acqID)
	if err != nil {
		return err
	}

	if err := s.seedParcelDetail(ctx, parcelID, acqID, holderID); err != nil {
		return err
	}

	if err := s.seedExtraParcels(ctx, acqID); err != nil {
		return err
	}

	return s.seedAssets(ctx, acqID)
}

// seedSelectedValuationBackfill — зөвшөөрөгдсөн үнэлгээтэй хуучин парцелуудад
// selected_valuation_type='asset' онооно (migration 000048-ийн schema-д харгалзах
// өгөгдлийн шилжүүлэг — data өөрчлөлт зөвхөн seeder-ээр хийгдэнэ).
func (s *MainSeeder) seedSelectedValuationBackfill(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		UPDATE parcel SET selected_valuation_type = 'asset'
		WHERE selected_valuation_type IS NULL
		  AND parcel_id IN (SELECT parcel_id FROM parcel_valuation_submission WHERE status = 'approved')
	`)
	return err
}

// SeedPlanBackfill — migration 000068-ийн plan хүснэгтэд харгалзах өгөгдлийн
// шилжүүлэг: одоо байгаа чөлөөлөлтүүдийн төлөвлөгөөний мэдээллийг (plan_code,
// plan_name, plan_geom, plan_area_m2) plan хүснэгт руу зөөнө.
//
// Яагаад: төлөвлөгөөгөөр хайх/сонгох үйлдэл цаашид ЗӨВХӨН plan хүснэгтээс
// уншина. Backfill байхгүй бол хуучин чөлөөлөлтүүдийн төлөвлөгөө сонголтод
// харагдахгүй болно.
//
// Идемпотент: plan_code-оор ON CONFLICT DO NOTHING, дараа нь чөлөөлөлтийн
// plan_id-г шинэ мөрүүдтэй тааруулна (өмнө нь auth DB-ийн plan.id-г хэлдэг байсан).
func (s *MainSeeder) SeedPlanBackfill(ctx context.Context) error {
	if _, err := s.db.Exec(ctx, `
		INSERT INTO plan (id, plan_code, name, boundary_wkt, area_m2, status)
		SELECT DISTINCT ON (la.plan_code)
		       gen_random_uuid(),
		       la.plan_code,
		       COALESCE(la.plan_name, ''),
		       la.plan_geom,
		       la.plan_area_m2,
		       2
		FROM land_acquisition la
		WHERE COALESCE(la.plan_code, '') <> ''
		ORDER BY la.plan_code, la.created_at
		ON CONFLICT (plan_code) DO NOTHING
	`); err != nil {
		return err
	}

	_, err := s.db.Exec(ctx, `
		UPDATE land_acquisition la
		SET plan_id = p.id
		FROM plan p
		WHERE p.plan_code = la.plan_code
		  AND la.plan_id IS DISTINCT FROM p.id
	`)
	return err
}

// seedDocumentTypeBackfill — хуучин land_acquisition_document.file_type-д хадгалагдаж
// байсан хавсралтын төрлийг document.document_type_id руу шилжүүлнэ.
// Багана аль хэдийн устсан (migration 000050) орчинд юу ч хийхгүй.
func (s *MainSeeder) seedDocumentTypeBackfill(ctx context.Context) error {
	var hasColumn bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'land_acquisition_document' AND column_name = 'file_type'
		)`).Scan(&hasColumn); err != nil {
		return err
	}
	if !hasColumn {
		return nil
	}
	_, err := s.db.Exec(ctx, `
		UPDATE document d SET document_type_id = lad.file_type
		FROM land_acquisition_document lad
		WHERE lad.document_id = d.id AND d.document_type_id IS NULL
	`)
	return err
}

func (s *MainSeeder) seedDocumentTypes(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO document_type (type, name, target) VALUES
		  ('photo_valuation',        'Эд хөрөнгийн үнэлгээний ажил хийсэн Фото зураг',                          'parcel'),
		  ('tax_receipt',            'Бизнесийн үйл ажиллагаатай холбоотой татвар төлсөн баримт',                'parcel'),
		  ('engineering_blueprint',  'Инженерийн бүтээн байгуулалт хийсэн зураг төсөл, баримт нотолгоо',         'parcel'),
		  ('id_card',                'Иргэний үнэмлэх',                                                          'parcel'),
		  ('company_certificate',    'ААНБ-ын гэрчилгээ',                                                        'parcel'),
		  ('land_certificate',       'Газар гэрчилгээ',                                                          'parcel'),
		  ('cadastral_map',          'Кадастрын зураг',                                                          'parcel'),
		  ('bank_statement',         'Банкны тодорхойлолт /хамтран өмчлөгч тус бүр хувь авах тохиолдолд/',      'parcel'),
		  ('power_of_attorney',      'Итгэмжлэл',                                                                'parcel'),
		  ('inheritance_agreement',  'Өв залгамжлах гэрээ',                                                      'parcel'),
		  ('death_certificate',      'Нас барсны гэрчилгээ',                                                     'parcel'),
		  ('control_card',           'Хяналтын карт',                                                            'parcel'),
		  ('land_handover_act',      'Чөлөөлсөн газрын хүлээлцсэн акт',                                         'parcel'),
		  ('payment_table',          'Төлбөрийн гүйцэтгэлийн хүснэгт',                                          'parcel'),
		  ('decree_draft',           'Захирамжийн төсөл',                                                        'both'),
		  ('settlement_act',         'Тооцоо нийлсэн акт',                                                      'parcel'),
		  ('monitoring_report',      'Мониторингийн дүгнэлт',                                                    'parcel'),
		  ('contract_closing_act',   'Гэрээ дүгнэсэн акт',                                                      'parcel'),
		  ('disability_certificate', 'Хөгжлийн бэрхшээлтэй иргэний тодорхойлолт',                              'parcel'),
		  ('signature_guarantee',    'Гарын үсгийн баталгаа',                                                   'parcel'),
		  ('topographic_map',        'Трас зураг',                                                              'parcel'),
		  ('prior_notice',           'Урьдчилан мэдэгдэх хуудас',                                              'parcel'),
		  ('survey_form',            'Санал асуулгын хуудас',                                                   'parcel'),
		  ('socioeconomic_study',    'Нийгэм эдийн засгийн судалгаа',                                          'parcel'),
		  ('checklist',              'Шалгах хуудас',                                                           'parcel'),
		  ('meeting_minutes',        'Хурлын тэмдэглэл',                                                       'parcel'),
		  ('contract',               'Гэрээ',                                                                   'parcel'),
		  ('consent_note',           'Зөвшилцлөлийн тэмдэглэл',                                               'parcel'),
		  ('valuation_report',       'Хөрөнгийн үнэлгээний тайлан',                                            'parcel'),
		  ('other',                  'БУСАД',                                                                   'both')
		ON CONFLICT (type) DO NOTHING
	`)
	if err != nil {
		return err
	}
	slog.Info("document types seeded")
	return nil
}

func (s *MainSeeder) seedDefaultAssets(ctx context.Context) error {
	acqID, err := s.seedLandAcquisition(ctx)
	if err != nil {
		return err
	}

	if _, err := s.seedParcel(ctx, acqID); err != nil {
		return err
	}

	return s.seedAssets(ctx, acqID)
}

// seedAUs — Аймаг, Сум, Баг жишээ өгөгдөл
// Координат: Монгол улсын хил дотор (WGS84/EPSG:4326)
func (s *MainSeeder) seedAUs(ctx context.Context) error {
	// Аймаг — Төв аймаг (ойролцоо хил, хялбарчилсан)
	if _, err := s.db.Exec(ctx, `
		INSERT INTO au1 (code, name, area_m2, geometry)
			VALUES (
				$1, $2, $3,
				ST_Multi(ST_GeomFromText($4, 4326))::geometry(MultiPolygon, 4326)
			)
		ON CONFLICT (code) DO NOTHING
	`,
		"01",
		"Төв аймаг",
		74000000000.0,
		"POLYGON((104.0 46.5, 109.0 46.5, 109.0 49.0, 104.0 49.0, 104.0 46.5))",
	); err != nil {
		return err
	}
	slog.Info("au1 seeded", "code", "01", "name", "Төв аймаг")

	// Сум — Зуунмод сум
	if _, err := s.db.Exec(ctx, `
		INSERT INTO au2 (code, name, au1_code, area_m2, geometry)
			VALUES (
				$1, $2, $3, $4,
				ST_GeomFromText($5, 4326)::geometry(Polygon, 4326)
			)
		ON CONFLICT (code) DO NOTHING
	`,
		"0101",
		"Зуунмод сум",
		"01",
		2800000000.0,
		"POLYGON((106.3 47.5, 107.8 47.5, 107.8 48.3, 106.3 48.3, 106.3 47.5))",
	); err != nil {
		return err
	}
	slog.Info("au2 seeded", "code", "0101", "name", "Зуунмод сум")

	// Баг — 1-р баг
	if _, err := s.db.Exec(ctx, `
		INSERT INTO au3 (code, name, au1_code, au2_code, area_m2, geometry)
			VALUES (
				$1, $2, $3, $4, $5,
				ST_Multi(ST_GeomFromText($6, 4326))::geometry(MultiPolygon, 4326)
			)
		ON CONFLICT (code) DO NOTHING
	`,
		"010101",
		"1-р баг",
		"01",
		"0101",
		350000000.0,
		"POLYGON((106.7 47.6, 107.1 47.6, 107.1 47.95, 106.7 47.95, 106.7 47.6))",
	); err != nil {
		return err
	}
	slog.Info("au3 seeded", "code", "010101", "name", "1-р баг")

	return nil
}

// seedLandAcquisition — газар чөлөөлөлтын мэдээлэл
func (s *MainSeeder) seedLandAcquisition(ctx context.Context) (uuid.UUID, error) {
	const planWKT = "POLYGON((106.75 47.65, 106.90 47.65, 106.90 47.80, 106.75 47.80, 106.75 47.65))"

	// ЧУХАЛ: land_acquisition.plan_code нь UNIQUE БИШ (нэг төлөвлөгөөний код
	// хэд хэдэн чөлөөлөлтөд давтагдаж болно) тул ON CONFLICT (plan_code)
	// хэрэглэж БОЛОХГҮЙ — Postgres 42P10 алдаа өгнө. Идемпотент байдлыг
	// урьдчилсан хайлтаар хангана.
	var id uuid.UUID
	err := s.db.QueryRow(ctx,
		`SELECT id FROM land_acquisition WHERE plan_code = $1 ORDER BY created_at LIMIT 1`,
		"LA-2024-001").Scan(&id)
	if err == nil {
		if _, err := s.db.Exec(ctx, `
			UPDATE land_acquisition SET
				plan_geom  = COALESCE(plan_geom, ST_GeomFromText($2, 4326)::geometry(Polygon, 4326)),
				geometry   = COALESCE(geometry,  ST_GeomFromText($2, 4326)::geometry(Polygon, 4326)),
				updated_at = NOW()
			WHERE id = $1`, id, planWKT); err != nil {
			return uuid.Nil, err
		}
		slog.Info("land_acquisition already seeded", "id", id, "plan_code", "LA-2024-001")
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	err = s.db.QueryRow(ctx, `
		INSERT INTO land_acquisition (
			id, plan_id, plan_code,
			plan_geom, plan_area_m2,
			geometry, area_m2,
			status, start_date, end_date,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(),
			uuid_generate_v4(),
			$1,
			ST_GeomFromText($2, 4326)::geometry(Polygon, 4326), $3,
			ST_GeomFromText($4, 4326)::geometry(Polygon, 4326), $5,
			$6, $7, $8,
			'system', 'system'
		)
		RETURNING id
	`,
		"LA-2024-001",
		planWKT, 1500000.0,
		planWKT, 1500000.0,
		2, // ACTIVE
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	slog.Info("land_acquisition seeded", "id", id, "plan_code", "LA-2024-001")
	return id, nil
}

// seedLandAcquisitionAU — газар чөлөөлөлт ↔ захиргааны нэгж холбоос
func (s *MainSeeder) seedLandAcquisitionAU(ctx context.Context, acqID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO land_acquisition_au (
			land_acquisition_id,
			au1_code, au1_name,
			au2_code, au2_name,
			au3_code, au3_name,
			created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'system', 'system')
		ON CONFLICT (land_acquisition_id, au3_code) DO NOTHING
	`,
		acqID,
		"01", "Төв аймаг",
		"0101", "Зуунмод сум",
		"010101", "1-р баг",
	)
	if err != nil {
		return err
	}
	slog.Info("land_acquisition_au seeded")
	return nil
}

// seedLandHolder — газар эзэмшигч / өмчлөгч
func (s *MainSeeder) seedLandHolder(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO right_holder (
			id, last_name, name, register_no,
			address, phone, email, holder_type, civil_id,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), $1, $2, $3,
			$4, $5, $6, $7, $8,
			'system', 'system'
		)
		ON CONFLICT (register_no) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`,
		"Дорж",
		"Батбаяр",
		"УЦ89102501",
		"Улаанбаатар хот, Баянзүрх дүүрэг, 10-р хороо",
		"+97688001234",
		"batbayar@example.mn",
		"INDIVIDUAL",
		"МН1234567",
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	slog.Info("right_holder seeded", "id", id, "name", "Дорж Батбаяр")
	return id, nil
}

// seedParcel — нэгж талбар (газрын мэдээлэл)
// Газрын талбай: чөлөөлөх зоны дотор (~85,000 м²)
func (s *MainSeeder) seedParcel(ctx context.Context, acqID uuid.UUID) (uuid.UUID, error) {
	const geomWKT = "POLYGON((106.76 47.66, 106.82 47.66, 106.82 47.72, 106.76 47.72, 106.76 47.66))"
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO parcel (
			id, parcel_id, old_parcel_id,
			acquisition_id,
			valid_from, valid_till,
			au1_code, au2_code, au3_code,
			geometry,
			acquisition_geom,
			right_type, landuse, area_m2,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), $1, $2,
				$3,
				$4, NULL,
				$5, $6, $7,
				ST_GeomFromText($8, 4326)::geometry(Polygon, 4326),
				ST_Intersection(
					ST_GeomFromText($8, 4326)::geometry(Polygon, 4326),
					(SELECT geometry FROM land_acquisition WHERE id = $3)
				)::geometry(Polygon, 4326),
				$9, $10, $11,
				'system', 'system'
			)
		ON CONFLICT (parcel_id) DO UPDATE SET
			geometry         = CASE WHEN parcel.geometry IS NULL
			                        THEN EXCLUDED.geometry         ELSE parcel.geometry         END,
			acquisition_geom = CASE WHEN parcel.acquisition_geom IS NULL
			                        THEN EXCLUDED.acquisition_geom ELSE parcel.acquisition_geom END,
			updated_at       = NOW()
		RETURNING id
	`,
		"010101000001",
		"01010100000001",
		acqID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"01", "0101", "010101",
		geomWKT,
		3, // OWNERSHIP
		"AGRICULTURAL",
		85000.0,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	slog.Info("parcel seeded", "id", id, "parcel_id", "010101000001")
	return id, nil
}

// seedExtraParcels — 2-р болон 3-р нэгж талбарыг эзэмшигч, эрх зүйн мэдээлэлтэй нэмнэ.
// source_parcel-тай тохирсон geometry, holder мэдээллийг ашиглана.
func (s *MainSeeder) seedExtraParcels(ctx context.Context, acqID uuid.UUID) error {
	type parcelEntry struct {
		parcelID    string
		oldParcelID string
		geometryWKT string
		rightType   int
		landuse     string
		areaM2      float64
		lastName    string
		holderName  string
		regNo       string
		phone       string
		email       string
		holderType  string
		civilID     string
		appNo       string
		decisionNo  string
		contractNo  string
		certNo      string
	}

	entries := []parcelEntry{
		{
			parcelID:    "010101000002",
			oldParcelID: "01010100000002",
			geometryWKT: "POLYGON((106.82 47.66, 106.88 47.66, 106.88 47.72, 106.82 47.72, 106.82 47.66))",
			rightType:   2, landuse: "RESIDENTIAL", areaM2: 1200.0,
			lastName: "Гантулга", holderName: "Мөнхбаяр", regNo: "ДА95041512",
			phone: "+97699112233", email: "munkhbayar@example.mn",
			holderType: "INDIVIDUAL", civilID: "МН9087654",
			appNo: "ГМТ-2024-002", decisionNo: "ТЗД-2024/002",
			contractNo: "ГЭ-2024-002", certNo: "01-002-2024",
		},
		{
			parcelID:    "010101000003",
			oldParcelID: "01010100000003",
			geometryWKT: "POLYGON((106.76 47.72, 106.84 47.72, 106.84 47.78, 106.76 47.78, 106.76 47.72))",
			rightType:   3, landuse: "COMMERCIAL", areaM2: 5600.0,
			lastName: "", holderName: "Монгол Орон ХХК", regNo: "7700012345",
			phone: "+97677001234", email: "info@mongolornon.mn",
			holderType: "LEGAL_ENTITY", civilID: "",
			appNo: "ГМТ-2024-003", decisionNo: "ТЗД-2024/003",
			contractNo: "ГЭ-2024-003", certNo: "01-003-2024",
		},
	}

	for _, e := range entries {
		var holderID uuid.UUID
		if err := s.db.QueryRow(ctx, `
			INSERT INTO right_holder (
				id, last_name, name, register_no,
				address, phone, email, holder_type, civil_id,
				created_by, updated_by
			) VALUES (
				uuid_generate_v4(), $1, $2, $3,
				$4, $5, $6, $7, $8,
				'system', 'system'
			)
			ON CONFLICT (register_no) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`,
			e.lastName, e.holderName, e.regNo,
			"", e.phone, e.email, e.holderType, e.civilID,
		).Scan(&holderID); err != nil {
			return err
		}
		slog.Info("right_holder seeded", "register_no", e.regNo, "name", e.holderName)

		var parcelID uuid.UUID
		if err := s.db.QueryRow(ctx, `
			INSERT INTO parcel (
				id, parcel_id, old_parcel_id,
				acquisition_id,
				valid_from, valid_till,
				au1_code, au2_code, au3_code,
				geometry,
				acquisition_geom,
				right_type, landuse, area_m2,
				created_by, updated_by
			) VALUES (
				uuid_generate_v4(), $1, $2,
				$3,
				$4, NULL,
				$5, $6, $7,
				ST_GeomFromText($8, 4326)::geometry(Polygon, 4326),
				ST_Intersection(
					ST_GeomFromText($8, 4326)::geometry(Polygon, 4326),
					(SELECT geometry FROM land_acquisition WHERE id = $3)
				)::geometry(Polygon, 4326),
				$9, $10, $11,
				'system', 'system'
			)
			ON CONFLICT (parcel_id) DO UPDATE SET
				geometry         = CASE WHEN parcel.geometry IS NULL
				                        THEN EXCLUDED.geometry         ELSE parcel.geometry         END,
				acquisition_geom = CASE WHEN parcel.acquisition_geom IS NULL
				                        THEN EXCLUDED.acquisition_geom ELSE parcel.acquisition_geom END,
				updated_at       = NOW()
			RETURNING id
		`,
			e.parcelID, e.oldParcelID,
			acqID,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			"01", "0101", "010101",
			e.geometryWKT,
			e.rightType, e.landuse, e.areaM2,
		).Scan(&parcelID); err != nil {
			return err
		}
		slog.Info("parcel seeded", "parcel_id", e.parcelID)

		var count int
		if err := s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM parcel_detail WHERE parcel_id = $1", parcelID,
		).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		if _, err := s.db.Exec(ctx, `
			INSERT INTO parcel_detail (
				id,
				parcel_id, acquisition_id, right_type, holder_id,
				app_no, app_timestamps,
				decision_no, decision_date,
				contract_no, contract_date,
				certificate_no, certificate_date,
				valuation_zone, base_price_per_ha, auction_coeff, auction_price,
				created_by, updated_by
			) VALUES (
				uuid_generate_v4(),
				$1, $2, $3, $4,
				$5, $6,
				$7, $8,
				$9, $10,
				$11, $12,
				$13, $14, $15, $16,
				'system', 'system'
			)
		`,
			parcelID, acqID, e.rightType, holderID,
			e.appNo, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			e.decisionNo, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			e.contractNo, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			e.certNo, time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
			"А бүс, 1-р зэрэглэл", 450_000_000.0, 1.2, 540_000_000.0,
		); err != nil {
			return err
		}
		slog.Info("parcel_detail seeded", "parcel_id", e.parcelID)
	}
	return nil
}

// seedParcelDetail — нэгж талбарын эрх зүйн мэдээлэл (parcel_detail хүснэгтэд)
func (s *MainSeeder) seedParcelDetail(ctx context.Context, parcelID, acqID, holderID uuid.UUID) error {
	var count int
	if err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM parcel_detail WHERE parcel_id = $1", parcelID,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO parcel_detail (
			id,
			parcel_id, acquisition_id, right_type, holder_id,
			app_no, app_timestamps,
			decision_no, decision_date,
			contract_no, contract_date,
			certificate_no, certificate_date,
			valuation_zone, base_price_per_ha, auction_coeff, auction_price,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(),
			$1, $2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10,
			$11, $12,
			$13, $14, $15, $16,
			'system', 'system'
		)
	`,
		parcelID, acqID, 3, holderID, // 3=OWNERSHIP
		"ГМТ-2024-001", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		"ТЗД-2024/001", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		"ГЭ-2024-001", time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		"01-001-2024", time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
		"А бүс, 1-р зэрэглэл", 450_000_000.0, 1.2, 540_000_000.0,
	)
	if err != nil {
		return err
	}
	slog.Info("parcel_detail seeded")
	return nil
}

// seedAssets — газар чөлөөлөлтөд хамрагдсан хөрөнгийн жишээ мэдээлэл
func (s *MainSeeder) seedAssets(ctx context.Context, acqID uuid.UUID) error {
	assets := []struct {
		parcelID    string
		assetNumber string
		assetType   string
		assetName   string
		floorCount  int
		areaM2      float64
		ownerName   string
		address     string
		notes       string
		geometryWKT string
	}{
		{
			parcelID:    "010101000001",
			assetNumber: "A-001",
			assetType:   "real_state",
			assetName:   "Амины орон сууц",
			floorCount:  1,
			areaM2:      96.5,
			ownerName:   "Дорж Батбаяр",
			address:     "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:       "Тоосгон хийцтэй, инженерийн шугамд холбогдсон.",
			geometryWKT: "POLYGON((106.7700 47.6700, 106.7710 47.6700, 106.7710 47.6710, 106.7700 47.6710, 106.7700 47.6700))",
		},
		{
			parcelID:    "010101000001",
			assetNumber: "A-002",
			assetType:   "property",
			assetName:   "Гарааш",
			floorCount:  1,
			areaM2:      32.0,
			ownerName:   "Дорж Батбаяр",
			address:     "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:       "Металл хаалгатай туслах байгууламж.",
			geometryWKT: "POLYGON((106.7720 47.6700, 106.7725 47.6700, 106.7725 47.6706, 106.7720 47.6706, 106.7720 47.6700))",
		},
		{
			parcelID:    "010101000001",
			assetNumber: "A-003",
			assetType:   "property",
			assetName:   "Агуулах",
			floorCount:  1,
			areaM2:      120.0,
			ownerName:   "Дорж Батбаяр",
			address:     "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:       "Модон карказтай, улирлын чанартай ашигладаг.",
			geometryWKT: "POLYGON((106.7740 47.6695, 106.7752 47.6695, 106.7752 47.6705, 106.7740 47.6705, 106.7740 47.6695))",
		},
	}

	for _, a := range assets {
		tag, err := s.db.Exec(ctx, `
			INSERT INTO assets (
				acquisition_id,
				parcel_id,
				asset_number,
				asset_type,
				asset_name,
				floor_count,
				area_m2,
				owner_name,
				address,
				notes,
				geometry
			)
			-- INSERT ... SELECT дээр Postgres параметрийн төрлийг зорилтот
			-- баганаас гаргадаггүй. $1..$3 нь WHERE-д мөн орох тул cast-гүй
			-- бол "inconsistent types deduced" (42P08) алдаа өгнө.
			SELECT
				$1::uuid, $2::varchar, $3::text, $4::varchar, $5::text,
				$6::int4, $7::numeric, $8::text, $9::text, $10::text,
				ST_GeomFromText($11::text, 4326)::geometry(Polygon, 4326)
			WHERE NOT EXISTS (
				SELECT 1
				FROM assets
				WHERE acquisition_id = $1::uuid
				  AND parcel_id = $2::varchar
				  AND asset_number = $3::text
			)
		`,
			acqID,
			a.parcelID,
			a.assetNumber,
			a.assetType,
			a.assetName,
			a.floorCount,
			a.areaM2,
			a.ownerName,
			a.address,
			a.notes,
			a.geometryWKT,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			slog.Info("asset seeded", "parcel_id", a.parcelID, "asset_number", a.assetNumber)
		}
	}

	return nil
}
