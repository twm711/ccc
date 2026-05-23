package telephony

import "errors"

var (
	ErrCarrierNotFound     = errors.New("carrier not found")
	ErrSIPTrunkNotFound    = errors.New("sip trunk not found")
	ErrPhoneNumberNotFound = errors.New("phone number not found")
	ErrPhoneNumberExists   = errors.New("phone number already exists")
)
