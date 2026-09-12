package domain

import "errors"

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")
var ErrInvalidInput = errors.New("invalid input")
var ErrForbidden = errors.New("forbidden")
var ErrInvalidSourceType = errors.New("invalid source type")
var ErrSourceTextUnavailable = errors.New("source text unavailable")
var ErrQuestionsPreparing = errors.New("questions preparing")
var ErrQuestionGenerationFailed = errors.New("question generation failed")
var ErrAllCopyProtected = errors.New("all highlights are copy protected")
var ErrQuestionBudgetExceeded = errors.New("question budget exceeded")
var ErrGlobalLLMBudgetExceeded = errors.New("global llm budget exceeded")
var ErrQuestionQueueDepthExceeded = errors.New("question generation queue depth exceeded")
var ErrPairingNotApproved = errors.New("pairing not approved yet")
var ErrRateLimitExceeded = errors.New("rate limit exceeded")
var ErrAccountHasAdminRole = errors.New("account has admin role and cannot be deleted")

// ValidationError carries a user-facing validation message that handlers may
// return verbatim in a 400 response. Anything not wrapped in this type must
// never reach the client as-is.
type ValidationError struct {
	Message string
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Is lets errors.Is(err, ErrInvalidInput) match validation errors so existing
// invalid-input branches keep working.
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}
