package seed

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MainSeeder үндсэн DB-ийн лавлах болон шилжүүлгийн seed-үүдийг ажиллуулна.
type MainSeeder struct {
	db *pgxpool.Pool
}

func NewMainSeeder(db *pgxpool.Pool) *MainSeeder {
	return &MainSeeder{db: db}
}

func (s *MainSeeder) Seed(ctx context.Context) error {
	slog.Info("main demo data seed disabled")
	return nil
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

// SeedCancelledValuationFiles — migration 000039-д харгалзах ӨГӨГДЛИЙН засвар:
// ӨМНӨ нь цуцлагдсан үнэлгээнүүдийн файлыг тухайн snapshot-д холбоно.
//
// Яагаад: цуцлах логик өмнө нь зөвхөн тайлан/эх хүснэгтийг parcel_document-оос
// хассан ба илгээлтийн БАТАЛГААЖААГҮЙ тайланг (draft_report_*) хөндөөгүй тул тэр
// нь шинэ үнэлгээний "Нөхөх олговор" дэлгэц дээр хуучнаараа харагдсаар байв.
// Мөн snapshot.files хоосон учир аль файл АЛЬ цуцлалтынх нь тодорхойгүй байсан.
//
// Идемпотент: files аль хэдийн бөглөгдсөн snapshot-ыг алгасна, draft_report нь
// хоосон болсон мөрийг дахин хөндөхгүй.
func (s *MainSeeder) SeedCancelledValuationFiles(ctx context.Context) error {
	// 1) Хуучин snapshot-уудын files-ыг report_url/source_url-ээс бүрдүүлнэ.
	if _, err := s.db.Exec(ctx, `
		UPDATE parcel_valuation_snapshot s
		SET files = (
			SELECT COALESCE(jsonb_agg(f), '[]'::jsonb)
			FROM (
				SELECT jsonb_build_object(
				         'label', 'Баталгаажсан тайлан',
				         'name', COALESCE(NULLIF(s.report_name,''), 'Баталгаажсан тайлан'),
				         'url', s.report_url) AS f
				WHERE COALESCE(s.report_url,'') <> ''
				UNION ALL
				SELECT jsonb_build_object(
				         'label', 'Үнэлгээний хүснэгт',
				         'name', COALESCE(NULLIF(s.source_name,''), 'Үнэлгээний хүснэгт'),
				         'url', s.source_url)
				WHERE COALESCE(s.source_url,'') <> ''
			) rows
		)
		WHERE COALESCE(jsonb_array_length(s.files), 0) = 0
		  AND (COALESCE(s.report_url,'') <> '' OR COALESCE(s.source_url,'') <> '')
	`); err != nil {
		return err
	}

	// 2) Цуцлагдсаны дараа илгээлтийн мөрөнд үлдсэн БАТАЛГААЖААГҮЙ тайланг
	//    хамгийн сүүлийн snapshot руу зөөнө (давхардвал нэмэхгүй).
	if _, err := s.db.Exec(ctx, `
		WITH leftover AS (
			SELECT sub.id AS submission_id,
			       sub.draft_report_url AS url,
			       COALESCE(NULLIF(sub.draft_report_name,''), 'Тайлан') AS name,
			       (SELECT snap.id
			          FROM parcel_valuation_snapshot snap
			         WHERE snap.acquisition_id = sub.acquisition_id
			           AND snap.parcel_id = sub.parcel_id
			           AND snap.valuation_type = sub.valuation_type
			         ORDER BY snap.cancelled_at DESC
			         LIMIT 1) AS snapshot_id
			FROM parcel_valuation_submission sub
			WHERE COALESCE(sub.draft_report_url,'') <> ''
			  AND sub.status = 'draft'
		), moved AS (
			UPDATE parcel_valuation_snapshot snap
			SET files = snap.files || jsonb_build_array(
			    jsonb_build_object('label','Тайлан','name',l.name,'url',l.url))
			FROM leftover l
			WHERE snap.id = l.snapshot_id
			  AND NOT snap.files @> jsonb_build_array(jsonb_build_object('url', l.url))
			RETURNING l.submission_id
		)
		UPDATE parcel_valuation_submission sub
		SET draft_report_url = '', draft_report_name = ''
		FROM leftover l
		WHERE sub.id = l.submission_id AND l.snapshot_id IS NOT NULL
	`); err != nil {
		return err
	}
	return nil
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
		  ('valuation_report',       'Баталгаажсан тайлан',                                                     'parcel'),
		  ('valuation_source',       'Үнэлгээний хүснэгт',                                                      'parcel'),
		  ('other',                  'БУСАД',                                                                   'both')
		ON CONFLICT (type) DO NOTHING
	`)
	if err != nil {
		return err
	}
	// НЭР солигдсон төрлүүд: ON CONFLICT DO NOTHING нь байгаа мөрийг хөнддөггүй
	// тул нэрийг тусад нь тэгшитгэнэ (зөвхөн ӨГӨГДМӨЛ нэр хэвээр байвал —
	// хэрэглэгч гараас өөрчилсөн байвал хүндэтгэнэ).
	if _, err = s.db.Exec(ctx, `
		UPDATE document_type SET name = 'Баталгаажсан тайлан'
		WHERE type = 'valuation_report' AND name = 'Хөрөнгийн үнэлгээний тайлан'
	`); err != nil {
		return err
	}
	slog.Info("document types seeded")
	return nil
}
