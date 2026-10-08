package legacyimport

import (
	"context"
	"strings"
)

// maxAttachmentNames — нэг удаад тулгах PDF-ийн дээд тоо (хавтас бүхэлдээ).
const maxAttachmentNames = 10000

// AttachmentParcel — PDF-ийн нэртэй (нэгж талбарын дугаар) таарсан нэгж талбар.
type AttachmentParcel struct {
	ID              string `json:"id"`
	AcquisitionID   string `json:"acquisitionId"`
	AcquisitionName string `json:"acquisitionName"`
	PlanCode        string `json:"planCode"`
	// Receipts — өмнө оруулсан «Нөхөх олговор олгосон баримт»-ын тоо (давхар оруулахаас сэргийлнэ).
	Receipts int `json:"receipts"`
}

// AttachmentMatch — нэг PDF-ийн нэрээр (яг тэнцүү) олдсон нэгж талбар;
// олдоогүй бол Parcel = nil. parcel.parcel_id нь UNIQUE тул давхцалгүй.
type AttachmentMatch struct {
	ParcelNumber string            `json:"parcelNumber"`
	Parcel       *AttachmentParcel `json:"parcel"`
}

// MatchAttachments — хавтсын PDF-үүдийн нэрийг (өргөтгөлгүй) нэгж талбарын
// дугаартай ЯГ тулгана. Хариу нь оролтын дарааллаар, давхардалгүй.
func (s *Service) MatchAttachments(ctx context.Context, names []string) ([]AttachmentMatch, error) {
	seen := map[string]bool{}
	var keys []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		keys = append(keys, n)
	}
	if len(keys) > maxAttachmentNames {
		return nil, &StepError{Message: "Нэг удаад хамгийн ихдээ 10000 файл тулгана"}
	}
	byNumber := map[string]*AttachmentParcel{}
	if len(keys) > 0 {
		rows, err := s.db.Query(ctx, `
			SELECT p.parcel_id, p.id::text, p.acquisition_id::text,
			       COALESCE(la.acquisition_name, ''), COALESCE(la.plan_code, ''),
			       (SELECT COUNT(*) FROM parcel_document pd
			          JOIN document d ON d.id = pd.document_id
			          JOIN document_type dt ON dt.id = d.document_type_id
			         WHERE pd.parcel_id = p.id AND dt.type = 'compensation_paid_receipt')
			  FROM parcel p
			  LEFT JOIN land_acquisition la ON la.id = p.acquisition_id
			 WHERE p.parcel_id = ANY($1)`, keys)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var number string
			p := &AttachmentParcel{}
			if err := rows.Scan(&number, &p.ID, &p.AcquisitionID, &p.AcquisitionName, &p.PlanCode, &p.Receipts); err != nil {
				return nil, err
			}
			byNumber[number] = p
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	out := make([]AttachmentMatch, 0, len(keys))
	for _, k := range keys {
		out = append(out, AttachmentMatch{ParcelNumber: k, Parcel: byNumber[k]})
	}
	return out, nil
}
