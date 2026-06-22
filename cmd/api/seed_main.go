package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MainSeeder үндсэн DB-д жишээ өгөгдөл нэмнэ.
// Аймаг → Сум → Баг → Газар чөлөөлөлт → Нэгж талбар → Эзэмшигч → Эрх зүй → Барилга
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

	if err := s.seedBuildings(ctx, acqID); err != nil {
		return err
	}

	return s.seedCompensations(ctx, acqID)
}

func (s *MainSeeder) seedDefaultBuildings(ctx context.Context) error {
	acqID, err := s.seedLandAcquisition(ctx)
	if err != nil {
		return err
	}

	if _, err := s.seedParcel(ctx, acqID); err != nil {
		return err
	}

	return s.seedBuildings(ctx, acqID)
}

// seedAUs — Аймаг, Сум, Баг жишээ өгөгдөл
// Координат: Монгол улсын хил дотор (WGS84/EPSG:4326)
func (s *MainSeeder) seedAUs(ctx context.Context) error {
	// Аймаг — Төв аймаг (ойролцоо хил, хялбарчилсан)
	if _, err := s.db.Exec(ctx, `
		INSERT INTO au1 (id, code, name, area_m2, geometry, created_by, updated_by)
			VALUES (
				uuid_generate_v4(), $1, $2, $3,
				ST_GeomFromText($4, 4326)::geometry(Polygon, 4326),
				'system', 'system'
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
		INSERT INTO au2 (id, code, name, au1_code, area_m2, geometry, created_by, updated_by)
			VALUES (
				uuid_generate_v4(), $1, $2, $3, $4,
				ST_GeomFromText($5, 4326)::geometry(Polygon, 4326),
				'system', 'system'
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
		INSERT INTO au3 (id, code, name, au1_code, au2_code, area_m2, geometry, created_by, updated_by)
			VALUES (
				uuid_generate_v4(), $1, $2, $3, $4, $5,
				ST_GeomFromText($6, 4326)::geometry(Polygon, 4326),
				'system', 'system'
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

	// Аль хэдийн оруулсан бол ID-г буцаана
	var id uuid.UUID
	err := s.db.QueryRow(ctx,
		"SELECT id FROM land_acquisition WHERE plan_code = $1 LIMIT 1", "LA-2024-001",
	).Scan(&id)
	if err == nil {
		slog.Info("land_acquisition already seeded", "id", id)
		return id, nil
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
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO parcel (
			id, parcel_id, old_parcel_id,
			acquisition_id,
			valid_from, valid_till,
			au1_code, au2_code, au3_code,
			geometry, right_type, landuse, area_m2,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), $1, $2,
				$3,
				$4, NULL,
				$5, $6, $7,
				ST_GeomFromText($8, 4326)::geometry(Polygon, 4326), $9, $10, $11,
				'system', 'system'
			)
		ON CONFLICT (parcel_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`,
		"010101000001",
		"01010100000001",
		acqID,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"01", "0101", "010101",
		"POLYGON((106.76 47.66, 106.82 47.66, 106.82 47.72, 106.76 47.72, 106.76 47.66))",
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

// seedBuildings — газар чөлөөлөлтөд хамрагдсан барилгын жишээ мэдээлэл
func (s *MainSeeder) seedBuildings(ctx context.Context, acqID uuid.UUID) error {
	buildings := []struct {
		parcelID       string
		buildingNumber string
		buildingType   string
		floorCount     int
		areaM2         float64
		ownerName      string
		address        string
		notes          string
		geometryWKT    string
	}{
		{
			parcelID:       "010101000001",
			buildingNumber: "B-001",
			buildingType:   "Амины орон сууц",
			floorCount:     1,
			areaM2:         96.5,
			ownerName:      "Дорж Батбаяр",
			address:        "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:          "Тоосгон хийцтэй, инженерийн шугамд холбогдсон.",
			geometryWKT:    "POLYGON((106.7700 47.6700, 106.7710 47.6700, 106.7710 47.6710, 106.7700 47.6710, 106.7700 47.6700))",
		},
		{
			parcelID:       "010101000001",
			buildingNumber: "B-002",
			buildingType:   "Гарааш",
			floorCount:     1,
			areaM2:         32.0,
			ownerName:      "Дорж Батбаяр",
			address:        "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:          "Металл хаалгатай туслах байгууламж.",
			geometryWKT:    "POLYGON((106.7720 47.6700, 106.7725 47.6700, 106.7725 47.6706, 106.7720 47.6706, 106.7720 47.6700))",
		},
		{
			parcelID:       "010101000001",
			buildingNumber: "B-003",
			buildingType:   "Агуулах",
			floorCount:     1,
			areaM2:         120.0,
			ownerName:      "Дорж Батбаяр",
			address:        "Төв аймаг, Зуунмод сум, 1-р баг, 12-1 тоот",
			notes:          "Модон карказтай, улирлын чанартай ашигладаг.",
			geometryWKT:    "POLYGON((106.7740 47.6695, 106.7752 47.6695, 106.7752 47.6705, 106.7740 47.6705, 106.7740 47.6695))",
		},
	}

	for _, b := range buildings {
		tag, err := s.db.Exec(ctx, `
			INSERT INTO building (
				acquisition_id,
				parcel_id,
				building_number,
				building_type,
				floor_count,
				area_m2,
				owner_name,
				address,
				notes,
				geometry
			)
			SELECT
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				ST_GeomFromText($10, 4326)::geometry(Polygon, 4326)
			WHERE NOT EXISTS (
				SELECT 1
				FROM building
				WHERE acquisition_id = $1
				  AND parcel_id = $2
				  AND building_number = $3
			)
		`,
			acqID,
			b.parcelID,
			b.buildingNumber,
			b.buildingType,
			b.floorCount,
			b.areaM2,
			b.ownerName,
			b.address,
			b.notes,
			b.geometryWKT,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			slog.Info("building seeded", "parcel_id", b.parcelID, "building_number", b.buildingNumber)
		}
	}

	return nil
}
