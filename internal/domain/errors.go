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
)
