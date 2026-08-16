package domain

import (
	"time"

	"github.com/google/uuid"
)

// Мэдэгдлийн төрлүүд — frontend икон/өнгө ялгахад ашиглана.
const (
	NotifAcquisitionStatus   = "acquisition_status"   // чөлөөлөлтийн статус шилжилт
	NotifParcelStatus        = "parcel_status"        // нэгж талбарын статус шилжилт
	NotifValuationSubmitted  = "valuation_submitted"  // үнэлгээ илгээгдсэн
	NotifValuationApproved   = "valuation_approved"   // үнэлгээ батлагдсан
	NotifValuationReturned   = "valuation_returned"   // үнэлгээ буцаагдсан
	NotifCompensationCreated = "compensation_created" // нөхөх олговор бүртгэгдсэн
	NotifCompensationReview  = "compensation_review"  // олговор батлагдсан/татгалзсан
	NotifAssigneeAdded       = "assignee_added"       // ажилтан хуваарилагдсан
	NotifOrgAssigned         = "org_assigned"         // мэрг./хөндлөнгийн байгууллага оноогдсон
	NotifValuationRequested  = "valuation_requested"  // талбар «Үнэлгээ хийх» төлөвт орж байгууллагаас үнэлгээ хүлээгдэж байна
)

// Холбогдох мэдээллийн төрөл — frontend дэлгэрэнгүй рүү үсрэх замыг үүнээс тогтооно.
const (
	NotifResourceAcquisition  = "acquisition"
	NotifResourceParcel       = "parcel"
	NotifResourceCompensation = "compensation"
)

type Notification struct {
	ID           uuid.UUID  `json:"id"`
	RecipientID  ID         `json:"recipient_id"`
	ActorID      *ID        `json:"actor_id,omitempty"`
	ActorName    string     `json:"actor_name"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	// AcquisitionID/ParcelID — дэлгэрэнгүй хуудасны замд шаардлагатай нэмэлт холбоос
	// (жишээ нь parcel хуудас /parcel/{id}?acq={acquisition_id} гэж нээгдэнэ).
	AcquisitionID *uuid.UUID `json:"acquisition_id,omitempty"`
	ParcelID      *uuid.UUID `json:"parcel_id,omitempty"`
	IsRead        bool       `json:"is_read"`
	CreatedAt     time.Time  `json:"created_at"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
}
