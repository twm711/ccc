package call

import "errors"

var (
	ErrCallNotFound      = errors.New("call not found")
	ErrRecordingNotFound = errors.New("recording not found")
	ErrVoicemailNotFound = errors.New("voicemail not found")
	ErrCallAlreadyEnded  = errors.New("call already ended")
	ErrInvalidDirection  = errors.New("invalid call direction")
)
