package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// seedCompensations — нөхөн олговрын жишээ өгөгдөл
// Compensation-г эхлээд оруулаад, дараа нь compensation_grant-г compensation_id-тай холбоно.
func (s *MainSeeder) seedCompensations(ctx context.Context, acqID uuid.UUID) error {
	var count int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM compensation WHERE acquisition_id = $1`, acqID,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		slog.Info("compensations already seeded")
		return nil
	}

	// Хөрөнгийн ID-г asset_number-ээр авна
	assetIDs := map[string]uuid.UUID{}
	rows, err := s.db.Query(ctx,
		`SELECT id, asset_number FROM assets WHERE acquisition_id = $1`, acqID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var num string
		if err := rows.Scan(&id, &num); err != nil {
			return err
		}
		assetIDs[num] = id
	}
	if err := rows.Err(); err != nil {
		return err
	}

	a001 := assetIDs["A-001"]
	if a001 == uuid.Nil {
		a001 = assetIDs["B-001"]
	}
	a002 := assetIDs["A-002"]
	if a002 == uuid.Nil {
		a002 = assetIDs["B-002"]
	}
	a003 := assetIDs["A-003"]
	if a003 == uuid.Nil {
		a003 = assetIDs["B-003"]
	}
	compDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)

	// ── 1. Compensation-уудыг эхлээд оруулна ─────────────────────────────────
	// ЖИЧ: compensation.asset_type-ыг 000016 миграц target_type-аар сольсон.
	// target_type нь хөрөнгийн төрөл БИШ — олговор юунд өгөгдсөнийг заана:
	// asset_id хоосон бол 'parcel' (нэгж талбар), эс бөгөөс 'asset' (хөрөнгө).
	type compSeed struct {
		parcelID         string
		assetID          *uuid.UUID
		compensationType string
		coveragePct      float64
		amount           float64
		note             string
		needsGrant       bool
	}

	assetPtr := func(id uuid.UUID) *uuid.UUID {
		if id == uuid.Nil {
			return nil
		}
		return &id
	}

	seeds := []compSeed{
		// Нэгж талбар: 80% мөнгөн
		{parcelID: "010101000001",
			compensationType: "cash", coveragePct: 80, amount: 68_000_000,
			note: "Нэгж талбарын 80%-д мөнгөн олговор олгосон"},
		// Нэгж талбар: 20% газраар → grant холбогдоно
		{parcelID: "010101000001",
			compensationType: "land_grant", coveragePct: 20, amount: 17_000_000,
			note: "Нэгж талбарын 20%-д газраар дүйцүүлэх олговор", needsGrant: true},
		// A-001 амины орон сууц: 100% мөнгөн
		{parcelID: "010101000001", assetID: assetPtr(a001),
			compensationType: "cash", coveragePct: 100, amount: 45_000_000,
			note: "Амины орон сууц барилгын бүрэн мөнгөн олговор"},
		// A-002 гарааш: 100% мөнгөн
		{parcelID: "010101000001", assetID: assetPtr(a002),
			compensationType: "cash", coveragePct: 100, amount: 12_000_000,
			note: "Гарааш барилгын бүрэн мөнгөн олговор"},
		// A-003 агуулах: 60% мөнгөн
		{parcelID: "010101000001", assetID: assetPtr(a003),
			compensationType: "cash", coveragePct: 60, amount: 25_000_000,
			note: "Агуулах барилгын 60%-д мөнгөн олговор"},
		// A-003 агуулах: 40% газраар → grant холбогдоно
		{parcelID: "010101000001", assetID: assetPtr(a003),
			compensationType: "land_grant", coveragePct: 40, amount: 10_000_000,
			note: "Агуулах барилгын 40%-д газраар дүйцүүлэх олговор", needsGrant: true},
	}

	// Compensation-уудын ID-г хадгалах (grant хэрэгтэй зөвхөн тэдгээрийн)
	var landGrantParcelCompID uuid.UUID
	var landGrantB003CompID uuid.UUID

	for i, c := range seeds {
		targetType := "parcel"
		if c.assetID != nil {
			targetType = "asset"
		}
		var compID uuid.UUID
		err := s.db.QueryRow(ctx, `
			INSERT INTO compensation (
				acquisition_id, target_type, parcel_id, asset_id,
				compensation_type, coverage_percent, amount,
				compensation_date, note, created_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'system')
			RETURNING id
		`,
			acqID, targetType, c.parcelID, c.assetID,
			c.compensationType, c.coveragePct, c.amount,
			compDate, c.note,
		).Scan(&compID)
		if err != nil {
			return err
		}
		slog.Info("compensation seeded",
			"target_type", targetType, "parcel_id", c.parcelID,
			"type", c.compensationType, "pct", c.coveragePct, "amount", c.amount,
		)
		if c.needsGrant {
			if i == 1 { // нэгж талбарын land_grant
				landGrantParcelCompID = compID
			} else { // B-003-ын land_grant
				landGrantB003CompID = compID
			}
		}
	}

	// ── 2. Compensation grant-уудыг compensation_id-тай холбон оруулна ────────
	grantDate1 := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	if _, err := s.db.Exec(ctx, `
		INSERT INTO compensation_grant (
			acquisition_id, compensation_id,
			amount, grant_date, note,
			land_area_m2, land_price,
			land_location, land_purpose, land_use_type,
			parcel_number, decree_number, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'system')
	`,
		acqID, landGrantParcelCompID,
		17_000_000, grantDate1,
		"Нэгж талбарын 20%-д хамаарах газраар дүйцүүлэх олговор",
		850, 20000,
		"Баянзүрх дүүрэг, 15-р хороо, А блок",
		"Орон сууцны зориулалттай",
		"Өмчийн",
		"БЗД-АБ-0042",
		"ТЗЭГ-2024-0128",
	); err != nil {
		return err
	}
	slog.Info("compensation_grant seeded", "compensation_id", landGrantParcelCompID, "note", "нэгж талбар 20%")

	grantDate2 := time.Date(2024, 7, 5, 0, 0, 0, 0, time.UTC)
	if _, err := s.db.Exec(ctx, `
		INSERT INTO compensation_grant (
			acquisition_id, compensation_id,
			amount, grant_date, note,
			land_area_m2, land_price,
			land_location, land_purpose, land_use_type,
			parcel_number, decree_number, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'system')
	`,
		acqID, landGrantB003CompID,
		10_000_000, grantDate2,
		"Агуулах (B-003) барилгын 40%-д хамаарах газраар дүйцүүлэх олговор",
		300, 33333,
		"Хан-Уул дүүрэг, 11-р хороо, В хэсэг",
		"Үйлдвэрлэл, үйлчилгээний зориулалттай",
		"Эзэмших",
		"ХУД-ВХ-0019",
		"ТЗЭГ-2024-0215",
	); err != nil {
		return err
	}
	slog.Info("compensation_grant seeded", "compensation_id", landGrantB003CompID, "note", "B-003 агуулах 40%")

	return nil
}
