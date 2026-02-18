package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type AppError struct {
	StatusCode int
	Message    string
	Cause      error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewBadRequestError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusBadRequest, Message: message, Cause: cause}
}

func NewRequestTooLargeError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusRequestEntityTooLarge, Message: message, Cause: cause}
}

func NewUnauthorizedError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusUnauthorized, Message: message, Cause: cause}
}

func NewForbiddenError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusForbidden, Message: message, Cause: cause}
}

func NewMethodNotAllowedError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusMethodNotAllowed, Message: message, Cause: cause}
}

func NewInternalError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusInternalServerError, Message: message, Cause: cause}
}

func writeHTTPError(ctx context.Context, logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	attributes := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"cause", err,
	}

	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Error, err.Error())

	var appErr *AppError
	if errors.As(err, &appErr) {
		attributes = append(attributes,
			"status", appErr.StatusCode,
			"message", appErr.Message,
		)
		level := slog.LevelError
		if appErr.StatusCode < 500 {
			level = slog.LevelWarn
		}
		logger.Log(ctx, level, "request failed", attributes...)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}

	attributes = append(attributes, "status", http.StatusInternalServerError)

	logger.ErrorContext(ctx, "request failed with unexpected error type", attributes...)
	http.Error(w, "Failed to process request.", http.StatusInternalServerError)
}
