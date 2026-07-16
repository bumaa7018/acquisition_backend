package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/actorctx"

	"github.com/google/uuid"
)

// Энэ файл — газар чөлөөлөлтийн төлөв өөрчлөгдөх бүх цэгээс холбогдох
// хэрэглэгчдэд мэдэгдэл илгээх hook-ууд. Мэдэгдэл нь туслах үйлдэл:
// үндсэн бизнес үйлдлийг хэзээ ч унагаахгүй (алдааг зөвхөн логлоно),
// хүсэлтийг удаашруулахгүй (бүгд async ажиллана).

var valuationTypeLabels = map[string]string{
	domain.ValuationTypeAsset:       "Хөрөнгийн үнэлгээ",
	domain.ValuationTypeIndependent: "Хөндлөнгийн үнэлгээ",
	domain.ValuationTypeMika:        "МИКА үнэлгээ",
}

// notifyAsync — hook-ийг үндсэн хүсэлтээс тусад нь, өөрийн timeout-той
// context дээр ажиллуулна. Actor-ийг ХҮСЭЛТИЙН ctx-ээс урьдчилан хуулж өгнө.
func (s *landService) notifyAsync(reqCtx context.Context, fn func(ctx context.Context, actor actorctx.Actor)) {
	if s.notifier == nil {
		return
	}
	actor, _ := actorctx.From(reqCtx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("notification hook panic", "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		fn(ctx, actor)
	}()
}

func actorPtr(a actorctx.Actor) *uuid.UUID {
	if a.ID == uuid.Nil {
		return nil
	}
	id := a.ID
	return &id
}

// acquisitionRecipients — чөлөөлөлттэй холбоотой хэрэглэгчид:
// хуваарилагдсан ажилтнууд + (сонголтоор) оноогдсон мэргэжлийн байгууллага.
func (s *landService) acquisitionRecipients(ctx context.Context, acqID uuid.UUID, includeProfOrg bool) []uuid.UUID {
	var out []uuid.UUID
	if assignees, err := s.landRepo.GetAssignees(ctx, acqID); err == nil {
		for _, a := range assignees {
			out = append(out, a.UserID)
		}
	}
	if includeProfOrg {
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil && acq.ProfessionalOrgID != nil {
			out = append(out, *acq.ProfessionalOrgID)
		}
	}
	return out
}

// usersByRole — тухайн ролтой бүх идэвхтэй хэрэглэгч (auth DB).
func (s *landService) usersByRole(ctx context.Context, role string) []uuid.UUID {
	if s.userRepo == nil {
		return nil
	}
	active := true
	users, _, err := s.userRepo.List(ctx, domain.UserFilter{Role: role, IsActive: &active, Page: 1, PageSize: 200})
	if err != nil {
		slog.Error("notification recipient lookup failed", "role", role, "error", err)
		return nil
	}
	out := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		out = append(out, u.ID)
	}
	return out
}

// --- Emitters -----------------------------------------------------------

// Чөлөөлөлтийн статус шилжилт → хуваарилагдсан ажилтнууд + мэрг. байгууллага.
func (s *landService) notifyAcquisitionStatusChanged(reqCtx context.Context, acq *domain.LandAcquisition, toStatus int) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		acqID := acq.ID
		fromLabel := domain.AcquisitionStatusLabels[acq.Status]
		toLabel := domain.AcquisitionStatusLabels[toStatus]
		s.notifier.Notify(ctx, s.acquisitionRecipients(ctx, acqID, true), domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifAcquisitionStatus,
			Title:         "Газар чөлөөлөлтийн статус өөрчлөгдлөө",
			Body:          fmt.Sprintf("«%s»: %s → %s", acq.AcquisitionName, fromLabel, toLabel),
			ResourceType:  domain.NotifResourceAcquisition,
			ResourceID:    acqID.String(),
			AcquisitionID: &acqID,
		})
	})
}

// Нэгж талбарын статус шилжилт → ажилтнуудад ерөнхий мэдэгдэл.
// «Үнэлгээ хийх» төлөвт орсон бол оноогдсон мэргэжлийн байгууллага болон
// сонгогдсон (хөндлөнгийн) байгууллагад үнэлгээ эхлүүлэх ТУСГАЙ мэдэгдэл очно.
func (s *landService) notifyParcelStatusChanged(reqCtx context.Context, acqID, parcelPK uuid.UUID) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		parcel, err := s.landRepo.GetParcelByID(ctx, parcelPK)
		if err != nil {
			slog.Error("notification parcel lookup failed", "parcel", parcelPK, "error", err)
			return
		}

		// 1) Хуваарилагдсан ажилтнуудад — статус шилжилтийн мэдэгдэл.
		s.notifier.Notify(ctx, s.acquisitionRecipients(ctx, acqID, false), domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifParcelStatus,
			Title:         "Нэгж талбарын статус өөрчлөгдлөө",
			Body:          fmt.Sprintf("Талбар %s: шинэ статус «%s»", parcel.ParcelID, parcel.StatusName),
			ResourceType:  domain.NotifResourceParcel,
			ResourceID:    parcelPK.String(),
			AcquisitionID: &acqID,
			ParcelID:      &parcelPK,
		})

		// 2) «Үнэлгээ хийх» төлөвт орсон бол байгууллагуудад үнэлгээ
		//    эхлүүлэхийг хүссэн тусгай мэдэгдэл.
		if parcel.StatusID != domain.ParcelStatusEvaluating {
			return
		}
		var orgs []uuid.UUID
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil && acq.ProfessionalOrgID != nil {
			orgs = append(orgs, *acq.ProfessionalOrgID)
		}
		if parcel.IndependentOrgID != nil {
			orgs = append(orgs, *parcel.IndependentOrgID)
		}
		if len(orgs) == 0 {
			return
		}
		s.notifier.Notify(ctx, orgs, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifValuationRequested,
			Title:         "Үнэлгээ хийх талбар ирлээ",
			Body:          fmt.Sprintf("Талбар %s «Үнэлгээ хийх» төлөвт орлоо — үнэлгээгээ оруулна уу", parcel.ParcelID),
			ResourceType:  domain.NotifResourceParcel,
			ResourceID:    parcelPK.String(),
			AcquisitionID: &acqID,
			ParcelID:      &parcelPK,
		})
	})
}

// Үнэлгээний илгээх/батлах/буцаах шилжилт.
//   - submit → санхүүгийн мэргэжилтнүүд + ажилтнууд (баталгаажуулалт хүлээж байна)
//   - approve/return → илгээсэн этгээд + оноогдсон байгууллага + ажилтнууд
func (s *landService) notifyValuationTransition(reqCtx context.Context, acqID, parcelPK uuid.UUID, valuationType, action string, sub *domain.ValuationSubmission) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		typeLabel := valuationTypeLabels[valuationType]
		if typeLabel == "" {
			typeLabel = valuationType
		}
		parcelCode := ""
		if parcel, err := s.landRepo.GetParcelByID(ctx, parcelPK); err == nil {
			parcelCode = parcel.ParcelID
		}

		var (
			recipients []uuid.UUID
			nType      string
			title      string
			body       string
		)
		switch action {
		case "submit":
			nType = domain.NotifValuationSubmitted
			title = "Үнэлгээ баталгаажуулалт хүлээж байна"
			body = fmt.Sprintf("Талбар %s: %s илгээгдлээ", parcelCode, typeLabel)
			recipients = append(s.usersByRole(ctx, "finance_specialist"), s.acquisitionRecipients(ctx, acqID, false)...)
		case "approve":
			nType = domain.NotifValuationApproved
			title = "Үнэлгээ батлагдлаа"
			body = fmt.Sprintf("Талбар %s: %s батлагдаж үндсэн үнэлгээгээр сонгогдлоо", parcelCode, typeLabel)
			recipients = s.valuationParticipants(ctx, acqID, parcelPK, sub)
		case "return":
			nType = domain.NotifValuationReturned
			title = "Үнэлгээ буцаагдлаа"
			body = fmt.Sprintf("Талбар %s: %s засварлуулахаар буцаагдлаа", parcelCode, typeLabel)
			recipients = s.valuationParticipants(ctx, acqID, parcelPK, sub)
		default:
			return
		}
		s.notifier.Notify(ctx, recipients, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          nType,
			Title:         title,
			Body:          body,
			ResourceType:  domain.NotifResourceParcel,
			ResourceID:    parcelPK.String(),
			AcquisitionID: &acqID,
			ParcelID:      &parcelPK,
		})
	})
}

// valuationParticipants — үнэлгээний урсгалд оролцогчид: илгээсэн этгээд,
// оноогдсон байгууллагууд, хуваарилагдсан ажилтнууд.
func (s *landService) valuationParticipants(ctx context.Context, acqID, parcelPK uuid.UUID, sub *domain.ValuationSubmission) []uuid.UUID {
	recipients := s.acquisitionRecipients(ctx, acqID, true)
	if parcel, err := s.landRepo.GetParcelByID(ctx, parcelPK); err == nil && parcel.IndependentOrgID != nil {
		recipients = append(recipients, *parcel.IndependentOrgID)
	}
	if sub != nil && sub.SubmittedBy != "" {
		if submitter, err := uuid.Parse(sub.SubmittedBy); err == nil {
			recipients = append(recipients, submitter)
		}
	}
	return recipients
}

// Нөхөх олговор бүртгэгдэх → санхүүгийн мэргэжилтнүүд + ажилтнууд.
func (s *landService) notifyCompensationCreated(reqCtx context.Context, comp *domain.Compensation) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		acqID := comp.AcquisitionID
		compID := comp.ID
		recipients := append(s.usersByRole(ctx, "finance_specialist"), s.acquisitionRecipients(ctx, acqID, false)...)
		s.notifier.Notify(ctx, recipients, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifCompensationCreated,
			Title:         "Шинэ нөхөх олговор баталгаажуулалт хүлээж байна",
			Body:          fmt.Sprintf("Талбар %s: %.0f₮ дүнтэй олговор бүртгэгдлээ", comp.ParcelID, comp.Amount),
			ResourceType:  domain.NotifResourceCompensation,
			ResourceID:    compID.String(),
			AcquisitionID: &acqID,
		})
	})
}

// Нөхөх олговор батлагдах/татгалзах → бүртгэсэн этгээд + ажилтнууд + мэрг. байгууллага.
func (s *landService) notifyCompensationReviewed(reqCtx context.Context, acqID, compID uuid.UUID, approved bool, note string) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		recipients := s.acquisitionRecipients(ctx, acqID, true)
		parcelCode := ""
		if comp, err := s.landRepo.GetCompensation(ctx, compID); err == nil {
			parcelCode = comp.ParcelID
			if creator, err := uuid.Parse(comp.CreatedBy); err == nil {
				recipients = append(recipients, creator)
			}
		}
		title := "Нөхөх олговор батлагдлаа"
		body := fmt.Sprintf("Талбар %s: олговор батлагдлаа", parcelCode)
		if !approved {
			title = "Нөхөх олговор татгалзагдлаа"
			body = fmt.Sprintf("Талбар %s: олговор татгалзагдлаа", parcelCode)
			if note != "" {
				body += " — " + note
			}
		}
		s.notifier.Notify(ctx, recipients, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifCompensationReview,
			Title:         title,
			Body:          body,
			ResourceType:  domain.NotifResourceCompensation,
			ResourceID:    compID.String(),
			AcquisitionID: &acqID,
		})
	})
}

// Ажилтан хуваарилагдах → шинээр хуваарилагдсан хэрэглэгч бүрд.
func (s *landService) notifyAssigneesSet(reqCtx context.Context, acqID uuid.UUID, previous, current []domain.AcquisitionAssignee) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		prevSet := make(map[uuid.UUID]struct{}, len(previous))
		for _, a := range previous {
			prevSet[a.UserID] = struct{}{}
		}
		var added []uuid.UUID
		for _, a := range current {
			if _, existed := prevSet[a.UserID]; !existed {
				added = append(added, a.UserID)
			}
		}
		if len(added) == 0 {
			return
		}
		acqName := ""
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil {
			acqName = acq.AcquisitionName
		}
		s.notifier.Notify(ctx, added, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifAssigneeAdded,
			Title:         "Танд газар чөлөөлөлт хуваарилагдлаа",
			Body:          fmt.Sprintf("«%s» чөлөөлөлтөд хариуцагчаар томилогдлоо", acqName),
			ResourceType:  domain.NotifResourceAcquisition,
			ResourceID:    acqID.String(),
			AcquisitionID: &acqID,
		})
	})
}

// Мэргэжлийн / хөндлөнгийн байгууллага оноогдох → тухайн байгууллагын хэрэглэгчид.
func (s *landService) notifyOrgAssigned(reqCtx context.Context, acqID uuid.UUID, parcelPK *uuid.UUID, orgUserID *uuid.UUID, independent bool) {
	if orgUserID == nil {
		return
	}
	orgID := *orgUserID
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		acqName := ""
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil {
			acqName = acq.AcquisitionName
		}
		title := "Танд үнэлгээний ажил оноогдлоо"
		body := fmt.Sprintf("«%s» чөлөөлөлтөд мэргэжлийн байгууллагаар томилогдлоо", acqName)
		resourceType := domain.NotifResourceAcquisition
		resourceID := acqID.String()
		if independent && parcelPK != nil {
			body = fmt.Sprintf("«%s» чөлөөлөлтийн нэгж талбарт хөндлөнгийн үнэлгээчээр томилогдлоо", acqName)
			resourceType = domain.NotifResourceParcel
			resourceID = parcelPK.String()
		}
		s.notifier.Notify(ctx, []uuid.UUID{orgID}, domain.Notification{
			ActorID:       actorPtr(actor),
			ActorName:     actor.Name,
			Type:          domain.NotifOrgAssigned,
			Title:         title,
			Body:          body,
			ResourceType:  resourceType,
			ResourceID:    resourceID,
			AcquisitionID: &acqID,
			ParcelID:      parcelPK,
		})
	})
}
