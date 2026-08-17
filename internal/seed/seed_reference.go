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

// referenceRows — хүснэгт тус бүрийн лавлах мөрүүд, эх нь архивласан миграцууд:
//
//	parcel_status      ← 000020 / 000030
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

	// main DB-д ч sdplatform/codelists схем байдаг (гадаад хэрэглэгчид тэнд
	// амьдардаг) тул ижил лавлахууд хэрэгтэй.
	{"codelists.cl_person_type", `
		INSERT INTO codelists.cl_person_type (code, description, description_en)
		VALUES (1, 'Иргэн', 'citizen'), (2, 'Хуулийн этгээд', 'legal')
		ON CONFLICT (code) DO NOTHING`},

	{"sdplatform.sd_system", `
		INSERT INTO sdplatform.sd_system (system_id, "name", description)
		VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
		ON CONFLICT (system_id) DO NOTHING`},
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
