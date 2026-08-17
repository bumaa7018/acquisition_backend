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
