package seed

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReferenceSeeder — appdb (main)-ийн ЛАВЛАХ өгөгдөл.
//
// Эдгээр мөрүүд урьд нь миграцад INSERT-ээр орж байсан. Миграц нь baseline
// болж ЗӨВХӨН бүтэц агуулах болсон тул лавлах өгөгдөл энд ирэв.
//
// Бүгд "байхгүй бол нэмнэ" (ON CONFLICT DO NOTHING) — байгаа мөрийг дарж
// бичихгүй, устгахгүй. Гараар засварласан лавлах хэвээр үлдэнэ.
type ReferenceSeeder struct {
	db *pgxpool.Pool
}

func NewReferenceSeeder(db *pgxpool.Pool) *ReferenceSeeder {
	return &ReferenceSeeder{db: db}
}

const acquisitionCategorySeedSQL = `
	DO $seed_acquisition_category$
	DECLARE
		road_id int;
		ger_id int;
		green_id int;
	BEGIN
		INSERT INTO acquisition_category (name, parent_id, sort_order)
		SELECT 'Авто зам, замын байгууламж', NULL, 1
		WHERE NOT EXISTS (
			SELECT 1 FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Авто зам, замын байгууламж'
		)
		RETURNING id INTO road_id;
		IF road_id IS NULL THEN
			SELECT id INTO road_id FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Авто зам, замын байгууламж'
			ORDER BY sort_order, id LIMIT 1;
		END IF;

		INSERT INTO acquisition_category (name, parent_id, sort_order)
		SELECT 'Гэр хорооллын дахин төлөвлөлт', NULL, 2
		WHERE NOT EXISTS (
			SELECT 1 FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Гэр хорооллын дахин төлөвлөлт'
		)
		RETURNING id INTO ger_id;
		IF ger_id IS NULL THEN
			SELECT id INTO ger_id FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Гэр хорооллын дахин төлөвлөлт'
			ORDER BY sort_order, id LIMIT 1;
		END IF;

		INSERT INTO acquisition_category (name, parent_id, sort_order)
		SELECT 'Ногоон байгууламж, нийтийн эзэмшил', NULL, 3
		WHERE NOT EXISTS (
			SELECT 1 FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Ногоон байгууламж, нийтийн эзэмшил'
		)
		RETURNING id INTO green_id;
		IF green_id IS NULL THEN
			SELECT id INTO green_id FROM acquisition_category
			WHERE parent_id IS NULL AND name = 'Ногоон байгууламж, нийтийн эзэмшил'
			ORDER BY sort_order, id LIMIT 1;
		END IF;

		INSERT INTO acquisition_category (name, parent_id, sort_order)
		SELECT name, parent_id, sort_order
		FROM (VALUES
			('Шинэ авто зам', road_id, 1),
			('Зам өргөтгөл, шинэчлэл', road_id, 2),
			('Гүүр, нүхэн гарц', road_id, 3),
			('Орон сууцны хороолол', ger_id, 1),
			('Инженерийн дэд бүтэц', ger_id, 2),
			('Сургууль, цэцэрлэгийн талбай', ger_id, 3),
			-- Хуучин мэдээллийн архивын "Бүтээн байгуулалтын төрөл" баганад
			-- гарах боловч дээрх ангиллуудад багтахгүй төрлүүд.
			('Барилга, байгууламж', ger_id, 4),
			('Үер, халиа тошин', ger_id, 5),
			('Бусад', ger_id, 6),
			('Цэцэрлэгт хүрээлэн', green_id, 1),
			('Явган зам, дугуйн зам', green_id, 2),
			('Тоглоомын талбай', green_id, 3)
		) AS v(name, parent_id, sort_order)
		WHERE parent_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM acquisition_category c
			WHERE c.parent_id = v.parent_id AND c.name = v.name
		);
	END
	$seed_acquisition_category$`

// referenceRows — хүснэгт тус бүрийн лавлах мөрүүд, эх нь архивласан миграцууд:
//
//	parcel_status      ← 000020 / 000030
//	acquisition_category
//	construction_type  ← 000017
//	asset_spec_type    ← 000041
//	asset_calc_type    ← 000041
var referenceRows = []struct {
	name string
	sql  string
}{
	{"parcel_status", `
		INSERT INTO parcel_status (id, code, name, sort_order) VALUES
			(0, 'pending',      'Хүлээгдэж буй',        0),
			(1, 'processing',   'Зөвшилцөх шатандаа',   1),
			(2, 'evaluating',   'Үнэлгээ хийх',         2),
			(3, 'affected_out', 'Нөлөөлөгдсөн гарсан',  3),
			(4, 'rejected',     'Татгалзсан',           4),
			(5, 'released',     'Чөлөөлсөн',            5)
			ON CONFLICT (id) DO NOTHING`},

	{"acquisition_category", acquisitionCategorySeedSQL},

	{"construction_type", `
			INSERT INTO construction_type (code, name, sort_order) VALUES
				('road',        'Зам, дэд бүтэц',              1),
			('residential', 'Орон сууцны барилга',         2),
			('industrial',  'Аж үйлдвэрийн байгууламж',    3),
			('public',      'Нийтийн барилга байгууламж',  4),
			('power_heat',  'Цахилгаан, дулааны шугам',    5),
			('water',       'Усан хангамжийн байгууламж',  6),
			('other',       'Бусад',                       7)
		ON CONFLICT (code) DO NOTHING`},

	{"asset_spec_type", `
		INSERT INTO asset_spec_type (code, name, sort_order) VALUES
			('construction_type', 'Хийцийн төрөл',   1),
			('foundation',        'Суурь бүтэц',     2),
			('walls',             'Хана',            3),
			('roof',              'Хучилт-дээвэр',   4),
			('door_type',         'Хаалга',          5),
			('window_type',       'Цонх',            6),
			('heating',           'Халаалтын шийдэл', 7),
			('quality',           'Чанар',           8)
		ON CONFLICT (code) DO NOTHING`},

	{"asset_calc_type", `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order) VALUES
			('unit_cost',             'Барилгын нэгжийн өртөг',              'төгрөг/м²', 1),
			('coeff_price_increase',  'Үнийн өсөлт',                         '%',         2),
			('coeff_interior',        'Гоёлын дотор засалтай',               '%',         3),
			('coeff_casting',         'Хийц болон давхар хоорондын цутгалт', '%',         4),
			('coeff_brick',           'Тоосгон өнгөлгөө',                    '%',         5),
			('height',                'Өндөр',                               'м',         6),
			('full_replacement_cost', 'Бүрэн орлуулах өртөг',                '₮',         7),
			('depreciation_percent',  'Элэгдлийн хувь',                      '%',         8),
			('depreciation_amount',   'Элэгдлийн дүн',                       '₮',         9),
			('net_replacement_cost',  'Барилгын нөхөн орлуулах өртөг',       '₮',        10)
		ON CONFLICT (code) DO NOTHING`},
}

// Seed — лавлах бүрийг байхгүй бол нэмнэ.
func (s *ReferenceSeeder) Seed(ctx context.Context) error {
	for _, r := range referenceRows {
		tag, err := s.db.Exec(ctx, r.sql)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			slog.Info("reference rows seeded", "table", r.name, "rows", tag.RowsAffected())
		}
	}
	return nil
}

func (s *ReferenceSeeder) SeedAcquisitionCategories(ctx context.Context) error {
	_, err := s.db.Exec(ctx, acquisitionCategorySeedSQL)
	return err
}
