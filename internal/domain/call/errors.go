package call

import "errors"

var (
	ErrCallNotFound      = errors.New("call not found")
	ErrRecordingNotFound = errors.New("recording not found")
	ErrVoicemailNotFound = errors.New("voicemail not found")
	ErrCallAlreadyEnded  = errors.New("call already ended")
	ErrInvalidDirection  = errors.New("invalid call direction")
	ErrMissingCallee     = errors.New("callee number is required")
	ErrMissingCaller     = errors.New("caller number is required")
	ErrCallNotActive     = errors.New("call is not active")
	ErrCallNotHeld       = errors.New("call is not held")
	ErrMissingTransferTarget = errors.New("transfer target is required")
	ErrMissingDTMF       = errors.New("DTMF digits are required")
	ErrCallbackNotFound  = errors.New("callback request not found")
)
