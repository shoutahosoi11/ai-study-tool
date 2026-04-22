package domain

import "errors"

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")
var ErrInvalidInput = errors.New("invalid input")
var ErrInvalidSourceType = errors.New("invalid source type")
var ErrSourceTextUnavailable = errors.New("source text unavailable")
var ErrAllCopyProtected = errors.New("all highlights are copy protected")
var ErrStorageNotConfigured = errors.New("storage not configured")
