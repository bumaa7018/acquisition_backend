package seed

import (
	"context"
	"log/slog"

	"acquisition_backend/internal/domain"
)

// rejectionDocumentType — «Татгалзсан» явцад шилжихэд заавал хавсаргах баримт.
// Өмнө нь төлөв солих цонхонд ТӨРӨЛГҮЙ файлаар (document_type_id = NULL)
// шаарддаг байсан тул хавсралтын төрлийн жагсаалтад байгаагүй.
const rejectionDocumentType = "rejection_document"

// SeedParcelWorkflowRequiredDocuments — ажлын урсгалын шилжилтэд ЗААВАЛ байх
// хавсралтын өгөгдмөл тохиргоо (migration 000047). Өмнө КОДОНД очих төлөвөөр
// нь хатуу шаарддаг байсныг ТУХАЙН төлөв рүү очих БҮХ шилжилтэд оноож зан
// төлөвийг хэвээр үлдээнэ:
//
//	… → Чөлөөлсөн (released) — Үнэлгээний тайлан (valuation_report)
//	… → Татгалзсан (rejected) — Татгалзсан тухай баримт (rejection_document)
//
// Төлөвийг domain.ParcelStatus*-аар (id = code, 000053-аас хойш бүх орчинд
// ижил), төрлийг кодоор нь олно. Тохиргоо аль хэдийн байвал хөндөхгүй (ON CONFLICT DO NOTHING) — seed_history-гаар НЭГ л
// удаа ажиллах тул дараа нь админ хассан бол дахин нэмэгдэхгүй.
//
// Мөн өмнө «Татгалзсан» шилжилтэд хавсаргасан ТӨРӨЛГҮЙ баримтуудыг (түүхийн
// attachment_url-аар таньж) энэ төрлөөр тэмдэглэнэ — дахин татгалзах үед
// «аль хэдийн орсон» гэж тооцогдоно.
func (s *MainSeeder) SeedParcelWorkflowRequiredDocuments(ctx context.Context) error {
	if _, err := s.db.Exec(ctx, `
		INSERT INTO document_type (type, name, target)
		VALUES ($1, 'Татгалзсан тухай баримт', 'parcel')
		ON CONFLICT (type) DO NOTHING`, rejectionDocumentType); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO parcel_workflow_required_document (workflow_id, document_type_id)
		SELECT pw.id, dt.id
		FROM (VALUES ($2::int, 'valuation_report'), ($3::int, $1)) AS req(status_id, doc_type)
		JOIN parcel_workflow pw ON pw.to_status_id = req.status_id
		JOIN document_type dt ON dt.type = req.doc_type
		ON CONFLICT DO NOTHING`, rejectionDocumentType, domain.ParcelStatusReleased, domain.ParcelStatusRejected); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE document d SET document_type_id = dt.id
		FROM document_type dt
		WHERE dt.type = $1
		  AND d.document_type_id IS NULL
		  AND d.file_url <> ''
		  AND EXISTS (
			SELECT 1 FROM parcel_status_history h
			JOIN parcel p ON p.parcel_id = h.parcel_id AND p.acquisition_id = h.acquisition_id
			JOIN parcel_document pd ON pd.parcel_id = p.id AND pd.document_id = d.id
			WHERE h.status_id = $2 AND h.attachment_url = d.file_url
		  )`, rejectionDocumentType, domain.ParcelStatusRejected)
	if err != nil {
		return err
	}
	slog.Info("parcel workflow required documents seeded", "rejection_documents_typed", tag.RowsAffected())
	return nil
}
