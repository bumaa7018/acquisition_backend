package domain

import "errors"

var (
	ErrNotFound                = errors.New("not found")
	ErrAlreadyExists           = errors.New("already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrTooManyRequests         = errors.New("too many requests")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token expired")
	ErrInvalidInput            = errors.New("invalid input")
	ErrInternalServer          = errors.New("internal server error")
	ErrDecreeRequired          = errors.New("decree number and date are required")
	ErrApproved                = errors.New("approved record cannot be modified")
	ErrValuationReportRequired = errors.New("valuation report required before releasing parcel")
	ErrAcquisitionLocked       = errors.New("acquisition is locked")
	ErrInvalidTransition       = errors.New("invalid status transition")
	ErrValuationNotApproved    = errors.New("compensation valuation is not approved yet")
	ErrNonFinalParcels         = errors.New("all parcels must be in final status before confirming acquisition")
	ErrAssetPhotoRequired      = errors.New("real estate assets must have photos before submitting valuation")
	ErrReportBeforeApproval    = errors.New("valuation must be approved before attaching the report")

	// Захирамжийн төсөл
	ErrDecisionConfirmed   = errors.New("confirmed decision draft cannot be modified")
	ErrParcelNotReleased   = errors.New("only released parcels can be linked to a decision draft")
	ErrParcelAlreadyLinked = errors.New("parcel is already linked to another decision draft")
	ErrDecisionAcqMismatch = errors.New("parcel belongs to a different land acquisition")
	// Нэгж талбар холбохын өмнө захирамжид эх үүсвэр нэмсэн байх ба талбар
	// бүрт эх үүсвэр сонгосон байх ёстой.
	ErrFundingSourceRequired = errors.New("a funding source must be added and selected first")
	ErrFundingSourceInUse    = errors.New("funding source is in use by a linked parcel")
)
