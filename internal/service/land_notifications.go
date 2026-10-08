package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
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

// actorPtrFor — actor нь ЗӨВХӨН ӨӨРИЙН эх сурвалжийн хүлээн авагчдаас
// хасагдана (notificationService нь ActorID-тай тэнцүү хүлээн авагчийг
// алгасдаг). ID нь authdb ба appdb хоёрт мөргөлддөг тул өөр эх сурвалж руу
// дамжуулбал ижил дугаартай ӨӨР хүн мэдэгдлээ авахгүй үлдэнэ.
func actorPtrFor(a actorctx.Actor, source string) *domain.ID {
	if a.ID == 0 || a.SourceOrAuth() != source {
		return nil
	}
	id := a.ID
	return &id
}

// notifyTargets — мэдэгдэл хүргэх хаягууд. Дотоод ажилтан (sdplatform.sd_user)
// ба үнэлгээний байгууллага гэсэн ХОЁР эх сурвалж нь тус тусдаа serial
// дугаартай бөгөөд мөргөлддөг тул ЗААВАЛ тусад нь цуглуулж, тусад нь
// илгээгдэнэ. БАЙГУУЛЛАГЫН id нь хэрэглэгчийн id БИШ — ажилтнууд руу
// (OrgUserIDs) задарч байж хүнд хүрнэ.
type notifyTargets struct {
	users []domain.ID // sdplatform.sd_user (domain.UserSourceAuth)
	orgs  []domain.ID // valuation_org.id (domain.UserSourceValuationOrg руу задарна)
}

func (t *notifyTargets) addUsers(ids ...domain.ID) {
	for _, id := range ids {
		if id != 0 {
			t.users = append(t.users, id)
		}
	}
}

func (t *notifyTargets) addOrgs(ids ...*domain.ID) {
	for _, id := range ids {
		if id != nil && *id != 0 {
			t.orgs = append(t.orgs, *id)
		}
	}
}

func (t *notifyTargets) merge(other notifyTargets) {
	t.addUsers(other.users...)
	t.orgs = append(t.orgs, other.orgs...)
}

// dispatch — цуглуулсан хаягууд руу мэдэгдлийг эх сурвалж тус бүрээр илгээнэ.
func (s *landService) dispatch(ctx context.Context, actor actorctx.Actor, t notifyTargets, n domain.Notification) {
	n.ActorName = actor.Name
	if len(t.users) > 0 {
		toUsers := n
		toUsers.ActorID = actorPtrFor(actor, domain.UserSourceAuth)
		s.notifier.Notify(ctx, t.users, toUsers)
	}
	if len(t.orgs) == 0 {
		return
	}
	var employees []domain.ID
	seen := make(map[domain.ID]struct{}, len(t.orgs))
	for _, orgID := range t.orgs {
		if _, dup := seen[orgID]; dup {
			continue
		}
		seen[orgID] = struct{}{}
		ids, err := s.landRepo.OrgUserIDs(ctx, orgID)
		if err != nil {
			slog.Error("notification org employee lookup failed", "org", orgID, "error", err)
			continue
		}
		employees = append(employees, ids...)
	}
	if len(employees) == 0 {
		// Нэвтрэх эрхтэй ажилтангүй байгууллага — мэдэгдэх хүн байхгүй.
		return
	}
	toOrg := n
	toOrg.ActorID = actorPtrFor(actor, domain.UserSourceValuationOrg)
	s.notifier.NotifyTo(ctx, domain.UserSourceValuationOrg, employees, toOrg)
}

// acquisitionRecipients — чөлөөлөлттэй холбоотой хаягууд:
// хуваарилагдсан ажилтнууд + (сонголтоор) оноогдсон мэргэжлийн байгууллага.
func (s *landService) acquisitionRecipients(ctx context.Context, acqID uuid.UUID, includeProfOrg bool) notifyTargets {
	var t notifyTargets
	if assignees, err := s.landRepo.GetAssignees(ctx, acqID); err == nil {
		for _, a := range assignees {
			t.addUsers(a.UserID)
		}
	}
	if includeProfOrg {
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil {
			t.addOrgs(acq.ProfessionalOrgID)
		}
	}
	return t
}

// usersByRole — тухайн ролтой бүх идэвхтэй хэрэглэгч (auth DB).
func (s *landService) usersByRole(ctx context.Context, role string) []domain.ID {
	if s.userRepo == nil {
		return nil
	}
	active := true
	users, _, err := s.userRepo.List(ctx, domain.UserFilter{Role: role, IsActive: &active, Page: 1, PageSize: 200})
	if err != nil {
		slog.Error("notification recipient lookup failed", "role", role, "error", err)
		return nil
	}
	out := make([]domain.ID, 0, len(users))
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
		s.dispatch(ctx, actor, s.acquisitionRecipients(ctx, acqID, true), domain.Notification{
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
		s.dispatch(ctx, actor, s.acquisitionRecipients(ctx, acqID, false), domain.Notification{
			Type:          domain.NotifParcelStatus,
			Title:         "Нэгж талбарын статус өөрчлөгдлөө",
			Body:          fmt.Sprintf("Талбар %s: шинэ статус «%s»", parcel.ParcelID, parcel.StatusName),
			ResourceType:  domain.NotifResourceParcel,
			ResourceID:    parcelPK.String(),
			AcquisitionID: &acqID,
			ParcelID:      &parcelPK,
		})

		// 2) «Үнэлгээний шат» (is_valuation_stage) төлөвт орсон бол
		//    байгууллагуудад үнэлгээ эхлүүлэхийг хүссэн тусгай мэдэгдэл.
		if !parcel.StatusIsValuationStage {
			return
		}
		var orgs notifyTargets
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil {
			orgs.addOrgs(acq.ProfessionalOrgID)
		}
		orgs.addOrgs(parcel.IndependentOrgID)
		if len(orgs.orgs) == 0 {
			return
		}
		s.dispatch(ctx, actor, orgs, domain.Notification{
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
			recipients notifyTargets
			nType      string
			title      string
			body       string
		)
		switch action {
		case "submit":
			nType = domain.NotifValuationSubmitted
			title = "Үнэлгээ баталгаажуулалт хүлээж байна"
			body = fmt.Sprintf("Талбар %s: %s илгээгдлээ", parcelCode, typeLabel)
			recipients.addUsers(s.usersByRole(ctx, "finance_specialist")...)
			recipients.merge(s.acquisitionRecipients(ctx, acqID, false))
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
		s.dispatch(ctx, actor, recipients, domain.Notification{
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
func (s *landService) valuationParticipants(ctx context.Context, acqID, parcelPK uuid.UUID, sub *domain.ValuationSubmission) notifyTargets {
	recipients := s.acquisitionRecipients(ctx, acqID, true)
	if parcel, err := s.landRepo.GetParcelByID(ctx, parcelPK); err == nil {
		recipients.addOrgs(parcel.IndependentOrgID)
	}
	// SubmittedBy нь хүний НЭР бичигддэг (дугаар биш) тохиолдол бий — тоо
	// болж хөрвөвөл дотоод ажилтны ID гэж үзнэ.
	if sub != nil && sub.SubmittedBy != "" {
		if submitter, err := parseUserID(sub.SubmittedBy); err == nil {
			recipients.addUsers(submitter)
		}
	}
	return recipients
}

// Нөхөх олговор бүртгэгдэх → санхүүгийн мэргэжилтнүүд + ажилтнууд.
func (s *landService) notifyCompensationCreated(reqCtx context.Context, comp *domain.Compensation) {
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		acqID := comp.AcquisitionID
		compID := comp.ID
		var recipients notifyTargets
		recipients.addUsers(s.usersByRole(ctx, "finance_specialist")...)
		recipients.merge(s.acquisitionRecipients(ctx, acqID, false))
		s.dispatch(ctx, actor, recipients, domain.Notification{
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
			if creator, err := parseUserID(comp.CreatedBy); err == nil {
				recipients.addUsers(creator)
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
		s.dispatch(ctx, actor, recipients, domain.Notification{
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
		prevSet := make(map[domain.ID]struct{}, len(previous))
		for _, a := range previous {
			prevSet[a.UserID] = struct{}{}
		}
		var added notifyTargets
		for _, a := range current {
			if _, existed := prevSet[a.UserID]; !existed {
				added.addUsers(a.UserID)
			}
		}
		if len(added.users) == 0 {
			return
		}
		acqName := ""
		if acq, err := s.landRepo.GetByID(ctx, acqID); err == nil {
			acqName = acq.AcquisitionName
		}
		s.dispatch(ctx, actor, added, domain.Notification{
			Type:          domain.NotifAssigneeAdded,
			Title:         "Танд газар чөлөөлөлт хуваарилагдлаа",
			Body:          fmt.Sprintf("«%s» чөлөөлөлтөд хариуцагчаар томилогдлоо", acqName),
			ResourceType:  domain.NotifResourceAcquisition,
			ResourceID:    acqID.String(),
			AcquisitionID: &acqID,
		})
	})
}

// Мэргэжлийн / хараат бус байгууллага оноогдох → тухайн байгууллагын БҮХ
// нэвтрэх эрхтэй ажилтанд. Өмнө нь байгууллага = нэг хэрэглэгч байсан тул
// ганц хүнд илгээдэг байв.
func (s *landService) notifyOrgAssigned(reqCtx context.Context, acqID uuid.UUID, parcelPK *uuid.UUID, assignedOrgID *domain.ID, independent bool) {
	if assignedOrgID == nil {
		return
	}
	s.notifyAsync(reqCtx, func(ctx context.Context, actor actorctx.Actor) {
		var recipients notifyTargets
		recipients.addOrgs(assignedOrgID)
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
		s.dispatch(ctx, actor, recipients, domain.Notification{
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

// parseUserID — хэрэглэгчийн ID нь sdplatform-д int4 тул текстээс хөрвүүлнэ.
func parseUserID(s string) (domain.ID, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, err
	}
	return domain.ID(v), nil
}
