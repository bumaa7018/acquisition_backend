package service

// Газрын суурь үнэ, төлбөр, барьцаа, шүүхийн шийдвэр — "Мэдээлэл татах"-ын
// "Газрын төлбөр үнэлгээний мэдээлэл" бүлгийн жинхэнэ дуудалтууд.
//
// Барьцаа, төлбөр, суурь үнэ нь нэгж талбарын дугаараар, шүүхийн шийдвэр нь
// хадгалагдсан app_no-гоор middleware рүү хандана.

import (
	"context"
	"log/slog"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// resolveParcelWithAppNo — чөлөөлөлт + нэгж талбар + хадгалагдсан app_no.
func (s *landService) resolveParcelWithAppNo(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.LandAcquisition, *domain.Parcel, string, error) {
	if s.parcelInfo == nil {
		return nil, nil, "", domain.ErrInternalServer
	}
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, nil, "", domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, nil, "", domain.ErrNotFound
	}
	full, err := s.landRepo.GetParcelDetailByID(ctx, p.ID)
	if err != nil {
		return nil, nil, "", domain.ErrNotFound
	}
	var appNo string
	if full.Detail != nil {
		appNo = strings.TrimSpace(full.Detail.AppNo)
	}
	if appNo == "" {
		slog.Warn("нэгж талбарт app_no байхгүй", "parcel_id", p.ID, "parcel_code", parcelCode)
		return nil, nil, "", domain.ErrParcelAppNoMissing
	}
	return acq, p, appNo, nil
}

// SyncParcelBasePrice — газрын СУУРЬ ҮНИЙГ ГУС-аас татаж parcel_detail-д бичнэ.
func (s *landService) SyncParcelBasePrice(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelBasePrice, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	bp, err := s.parcelInfo.GetParcelBasePrice(ctx, parcelCode)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrNotFound
		}
		slog.Error("газрын суурь үнэ татахад алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}

	// Суурь үнийг нэгж талбарын үнэлгээний талбарт бичнэ. Бусад талбар
	// (бүс, дуудлагын коэф/үнэ) нь ГАРААР бөглөгддөг тул хадгалагдсанаар нь
	// дамжуулж, дарж бичихээс сэргийлнэ.
	var zone string
	var auctionCoeff, auctionPrice *float64
	if full, derr := s.landRepo.GetParcelDetailByID(ctx, p.ID); derr == nil && full.Detail != nil {
		zone = full.Detail.ValuationZone
		auctionCoeff = full.Detail.AuctionCoeff
		auctionPrice = full.Detail.AuctionPrice
	}
	// ГУС-д суурь үнэ байхгүй бол (base_price null/0) хадгалагдсан үнийг 0-оор
	// ДАРЖ БИЧИХГҮЙ — "Олдоогүй" гэж мэдэгдээд хуучин утгыг хэвээр үлдээнэ.
	if bp.BasePrice > 0 {
		if err := s.landRepo.UpdateParcelValuation(ctx, p.ID, zone, &bp.BasePrice, auctionCoeff, auctionPrice); err != nil {
			slog.Error("суурь үнэ хадгалахад алдаа", "parcel_id", p.ID, "error", err)
			return nil, domain.ErrInternalServer
		}
	} else {
		slog.Warn("ГУС-д газрын суурь үнэ олдсонгүй", "parcel_code", parcelCode)
	}
	// Хүчин зүйлийн задаргаа — суурь үнэ хадгалагдсаны дараа. Задаргаа
	// хадгалагдаагүй ч суурь үнэ бичигдсэн тул алхмыг унагахгүй.
	if _, ferr := s.landRepo.ReplaceParcelBasePriceFactors(ctx, p.ID, acquisitionID, bp.Factors); ferr != nil {
		slog.Error("суурь үнийн хүчин зүйл хадгалахад алдаа", "parcel_id", p.ID, "error", ferr)
	}

	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "base_price": bp.BasePrice, "factors": len(bp.Factors),
		})
	return bp, nil
}

// SyncParcelFee — ГАЗРЫН ТӨЛБӨРийн бодолтыг ГУС-аас татаж хадгална.
// ӨМЧЛӨЛИЙН (right_type=3) газарт төлбөр байдаггүй тул ГУС руу хандахгүй.
func (s *landService) SyncParcelFee(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelFee, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if p.RightType == domain.RightTypeOwnership {
		return nil, domain.ErrParcelOwnedNoPayment
	}

	fee, err := s.parcelInfo.CalculateParcelFee(ctx, parcelCode)
	if err != nil {
		slog.Error("газрын төлбөр бодоход алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	// status=false бол ГУС төлбөрийг бодож ЧАДААГҮЙ (шалтгаан msg-д). Тэр
	// тохиолдолд хадгалагдсан бодолтыг УСТГАХГҮЙ — хоосон биш, хуучин үлдэнэ.
	if !fee.Status {
		slog.Warn("газрын төлбөр бодогдсонгүй", "parcel_code", parcelCode, "msg", fee.Msg)
		return fee, nil
	}
	if _, err := s.landRepo.ReplaceParcelFeeItems(ctx, p.ID, acquisitionID, fee.Items); err != nil {
		slog.Error("газрын төлбөрийн бодолт хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "fee_items": len(fee.Items), "total_payment": fee.TotalPayment,
		})
	return fee, nil
}

// SyncParcelInvoices — газрын төлбөрийн нэхэмжлэл. ӨМЧЛӨЛИЙН (right_type=3)
// газарт төлбөр байдаггүй тул ГУС руу огт хандахгүй, анхааруулга буцаана.
func (s *landService) SyncParcelInvoices(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) ([]*domain.ParcelInvoice, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if p.RightType == domain.RightTypeOwnership {
		return nil, domain.ErrParcelOwnedNoPayment
	}

	invoices, err := s.parcelInfo.ListParcelInvoices(ctx, parcelCode)
	if err != nil {
		slog.Error("төлбөрийн мэдээлэл татахад алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.ReplaceParcelInvoices(ctx, p.ID, acquisitionID, invoices); err != nil {
		slog.Error("нэхэмжлэл хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "invoice_count": len(invoices),
		})
	return invoices, nil
}

// SyncParcelMortgages — барьцааны бүртгэлийг татаж хадгална (давхар барьцаатай).
func (s *landService) SyncParcelMortgages(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelSyncCountResult, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	items, err := s.parcelInfo.ListParcelMortgages(ctx, parcelCode)
	if err != nil {
		slog.Error("барьцааны мэдээлэл татахад алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	saved, err := s.landRepo.ReplaceParcelMortgages(ctx, p.ID, acq.ID, "", items)
	if err != nil {
		slog.Error("барьцаа хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acq.ID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "mortgage_count": saved,
		})
	return &domain.ParcelSyncCountResult{Found: len(items), Saved: saved}, nil
}

// SyncParcelCourtDecisions — шүүхийн шийдвэрийг татаж хадгална.
func (s *landService) SyncParcelCourtDecisions(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelSyncCountResult, error) {
	acq, p, appNo, err := s.resolveParcelWithAppNo(ctx, acquisitionID, parcelCode)
	if err != nil {
		return nil, err
	}
	items, err := s.parcelInfo.ListCourtDecisions(ctx, appNo)
	if err != nil {
		slog.Error("шүүхийн шийдвэр татахад алдаа", "app_no", appNo, "error", err)
		return nil, domain.ErrInternalServer
	}
	saved, err := s.landRepo.ReplaceParcelCourtDecisions(ctx, p.ID, acq.ID, appNo, items)
	if err != nil {
		slog.Error("шүүхийн шийдвэр хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acq.ID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "app_no": appNo, "court_decision_count": saved,
		})
	return &domain.ParcelSyncCountResult{Found: len(items), Saved: saved}, nil
}

// SyncParcelMonitorings — хянан баталгааны мэдээллийг нэгж талбарын дугаараар
// middleware-ээс татаж monitoring_id-гаар давхардуулахгүй хадгална.
func (s *landService) SyncParcelMonitorings(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelSyncCountResult, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	items, err := s.parcelInfo.ListParcelMonitorings(ctx, parcelCode)
	if err != nil {
		slog.Error("хянан баталгааны мэдээлэл татахад алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	saved, err := s.landRepo.UpsertParcelMonitorings(ctx, p.ID, acquisitionID, items)
	if err != nil {
		slog.Error("хянан баталгааны мэдээлэл хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "monitoring_count": saved,
		})
	return &domain.ParcelSyncCountResult{Found: len(items), Saved: saved}, nil
}

// SyncParcelOwnerships — УБЕГ-ийн газар өмчлөлийн бүртгэлийг нэгж талбарын
// дугаараар ХУР (XYP) гарцаас татаж хадгална.
//
// Эх сурвалж нь ГУС-ийн дундын сервис БИШ тул app_no шаардахгүй. Мөрүүд нь
// өмчлөлийн ТҮҮХ (хамтран өмчлөгчид + өмчлөгч өөрчлөгдсөн бүртгэл) байдаг тул
// бүхэлдээ орлуулагдана.
func (s *landService) SyncParcelOwnerships(
	ctx context.Context, acquisitionID uuid.UUID, parcelCode string,
) (*domain.ParcelSyncCountResult, error) {
	if s.ownershipInfo == nil {
		return nil, domain.ErrOwnershipNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	items, err := s.ownershipInfo.ListParcelOwnerships(ctx, parcelCode)
	if err != nil {
		slog.Error("УБЕГ-ийн өмчлөлийн мэдээлэл татахад алдаа", "parcel_code", parcelCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	saved, err := s.landRepo.ReplaceParcelOwnerships(ctx, p.ID, acquisitionID, items)
	if err != nil {
		slog.Error("УБЕГ-ийн өмчлөлийн мэдээлэл хадгалахад алдаа", "parcel_id", p.ID, "error", err)
		return nil, domain.ErrInternalServer
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode, "ownership_count": saved,
		})
	return &domain.ParcelSyncCountResult{Found: len(items), Saved: saved}, nil
}
