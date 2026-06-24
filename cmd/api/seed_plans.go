package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlansSeeder auth DB-д төлөвлөгөө болон эх нэгж талбарын жишээ өгөгдөл нэмнэ.
type PlansSeeder struct {
	db *pgxpool.Pool
}

func NewPlansSeeder(db *pgxpool.Pool) *PlansSeeder {
	return &PlansSeeder{db: db}
}

func (s *PlansSeeder) Seed(ctx context.Context) error {
	slog.Info("seeding plans into auth database...")

	if err := s.seedPlan(ctx); err != nil {
		return err
	}
	return s.seedSourceParcels(ctx)
}

// seedPlan — газар зохион байгуулалтын төлөвлөгөө
// Хилийн WKT: Зуунмод сумын орчим (EPSG:4326)
func (s *PlansSeeder) seedPlan(ctx context.Context) error {
	const boundary = "POLYGON((106.5 47.5, 107.5 47.5, 107.5 48.0, 106.5 48.0, 106.5 47.5))"

	_, err := s.db.Exec(ctx, `
			INSERT INTO plan (id, plan_code, name, boundary_wkt, area_m2, status)
			VALUES (uuid_generate_v4(), $1, $2, ST_GeomFromText($3, 4326)::geometry(Polygon, 4326), $4, 2)
			ON CONFLICT (plan_code) DO UPDATE SET
				boundary_wkt = CASE WHEN plan.boundary_wkt IS NULL
				                    THEN EXCLUDED.boundary_wkt ELSE plan.boundary_wkt END
		`,
		"PLAN-2024-001",
		"Зуунмод сумын газар зохион байгуулалтын ерөнхий төлөвлөгөө 2024",
		boundary,
		8500000000.0,
	)
	if err != nil {
		return err
	}
	slog.Info("plan seeded", "plan_code", "PLAN-2024-001")
	return nil
}

// seedSourceParcels — эх нэгж талбарын мэдээлэл (чөлөөлөх бүсийн дотор)
func (s *PlansSeeder) seedSourceParcels(ctx context.Context) error {
	parcels := []struct {
		parcelID                                                 string
		geom                                                     string
		au1, au2, au3                                            string
		rightType                                                int
		landuse                                                  string
		areaM2                                                   float64
		lastName, name, regNo, phone, email, holderType, civilID string
		appNo, decisionNo, contractNo, certNo                    string
	}{
		{
			parcelID: "010101000001",
			geom:     "POLYGON((106.76 47.66, 106.82 47.66, 106.82 47.72, 106.76 47.72, 106.76 47.66))",
			au1:      "01", au2: "0101", au3: "010101",
			rightType: 3, landuse: "AGRICULTURAL", areaM2: 85000, // 3=OWNERSHIP
			lastName: "Дорж", name: "Батбаяр", regNo: "УЦ89102501",
			phone: "+97688001234", email: "batbayar@example.mn",
			holderType: "INDIVIDUAL", civilID: "МН1234567",
			appNo: "ГМТ-2024-001", decisionNo: "ТЗД-2024/001",
			contractNo: "ГЭ-2024-001", certNo: "01-001-2024",
		},
		{
			parcelID: "010101000002",
			geom:     "POLYGON((106.82 47.66, 106.88 47.66, 106.88 47.72, 106.82 47.72, 106.82 47.66))",
			au1:      "01", au2: "0101", au3: "010101",
			rightType: 2, landuse: "RESIDENTIAL", areaM2: 1200, // 2=LEASE
			lastName: "Гантулга", name: "Мөнхбаяр", regNo: "ДА95041512",
			phone: "+97699112233", email: "munkhbayar@example.mn",
			holderType: "INDIVIDUAL", civilID: "МН9087654",
			appNo: "ГМТ-2024-002", decisionNo: "ТЗД-2024/002",
			contractNo: "ГЭ-2024-002", certNo: "01-002-2024",
		},
		{
			parcelID: "010101000003",
			geom:     "POLYGON((106.76 47.72, 106.84 47.72, 106.84 47.78, 106.76 47.78, 106.76 47.72))",
			au1:      "01", au2: "0101", au3: "010101",
			rightType: 3, landuse: "COMMERCIAL", areaM2: 5600, // 3=OWNERSHIP
			lastName: "", name: "Монгол Орон ХХК", regNo: "7700012345",
			phone: "+97677001234", email: "info@mongolornon.mn",
			holderType: "LEGAL_ENTITY", civilID: "",
			appNo: "ГМТ-2024-003", decisionNo: "ТЗД-2024/003",
			contractNo: "ГЭ-2024-003", certNo: "01-003-2024",
		},
	}

	for _, p := range parcels {
		_, err := s.db.Exec(ctx, `
			INSERT INTO source_parcel (
				id, parcel_id, plan_code, geometry_wkt,
				au1_code, au2_code, au3_code,
				right_type, landuse, area_m2,
				holder_last_name, holder_name, holder_register_no,
				holder_phone, holder_email, holder_type, holder_civil_id,
				app_no, decision_no, contract_no, certificate_no
				) VALUES (
					uuid_generate_v4(), $1, 'PLAN-2024-001', ST_GeomFromText($2, 4326)::geometry(Polygon, 4326),
					$3, $4, $5,
				$6, $7, $8,
				$9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19
			) ON CONFLICT (parcel_id) DO NOTHING
		`,
			p.parcelID, p.geom,
			p.au1, p.au2, p.au3,
			p.rightType, p.landuse, p.areaM2,
			p.lastName, p.name, p.regNo,
			p.phone, p.email, p.holderType, p.civilID,
			p.appNo, p.decisionNo, p.contractNo, p.certNo,
		)
		if err != nil {
			return err
		}
		slog.Info("source_parcel seeded", "parcel_id", p.parcelID)
	}
	return nil
}
