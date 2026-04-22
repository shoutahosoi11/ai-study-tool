package dto

import "time"

type CreateUploadSignedURLRequest struct {
	FileName         string `json:"file_name"`
	ContentType      string `json:"content_type"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type CreateDownloadSignedURLRequest struct {
	ObjectName       string `json:"object_name"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type SignedURLResponse struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Bucket      string            `json:"bucket"`
	ObjectName  string            `json:"object_name"`
	ContentType string            `json:"content_type,omitempty"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Headers     map[string]string `json:"headers"`
}
