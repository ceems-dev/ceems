//go:build cgo
// +build cgo

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

var (
	ErrMaxQueryWindow     = errors.New("maximum query window exceeded")
	ErrMalformedTimeStamp = errors.New("malformed timestamp")
)

// Error type in API response.
type errorType string

// APIError response.
type APIError struct {
	Typ errorType
	Err error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Typ, e.Err)
}

// List of predefined errors.
const (
	ErrorNone          errorType = ""
	ErrorUnauthorized  errorType = "unauthorized"
	ErrorForbidden     errorType = "forbidden"
	ErrorTimeout       errorType = "timeout"
	ErrorCanceled      errorType = "canceled"
	ErrorExec          errorType = "execution"
	ErrorBadData       errorType = "bad_data"
	ErrorInternal      errorType = "internal"
	ErrorUnavailable   errorType = "unavailable"
	ErrorNotFound      errorType = "not_found"
	ErrorNotAcceptable errorType = "not_acceptable"
)

// Custom error codes.
const (
	// Non-standard status code (originally introduced by nginx) for the case when a client closes
	// the connection while the server is still processing the request.
	statusClientClosedConnection = 499
)

// Custom errors.
var (
	ErrNoUser            = errors.New("no user identified")
	ErrNoPrivs           = errors.New("current user does not have admin privileges")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidQueryField = errors.New("invalid query fields")
	ErrMissingData       = errors.New("missing data in the request")
	ErrNoAuth            = errors.New("user do not have permissions to view metrics of this job/pod/vm")
	ErrNoAccess          = errors.New("user do not have permissions to access this resource")
	ErrInvalidClusterID  = errors.New("invalid ceems cluster id")
	ErrUnavailable       = errors.New("tsdb/pyroscope unavailable")
)

// ErrorResponse returns error response for by setting errorString and errorType in response.
func ErrorResponse[T any](w http.ResponseWriter, apiErr *APIError, logger *slog.Logger, data []T) {
	var code int

	switch apiErr.Typ { //nolint:exhaustive
	case ErrorBadData, errorType(ErrNoUser.Error()), errorType(ErrInvalidRequest.Error()), errorType(ErrInvalidQueryField.Error()), errorType(ErrMissingData.Error()):
		code = http.StatusBadRequest
	case ErrorUnauthorized, errorType(ErrNoPrivs.Error()):
		code = http.StatusUnauthorized
	case ErrorForbidden, errorType(ErrNoAuth.Error()), errorType(ErrNoAccess.Error()):
		code = http.StatusForbidden
	case ErrorExec:
		code = http.StatusUnprocessableEntity
	case ErrorCanceled:
		code = statusClientClosedConnection
	case ErrorTimeout, ErrorUnavailable, errorType(ErrUnavailable.Error()):
		code = http.StatusServiceUnavailable
	case ErrorInternal:
		code = http.StatusInternalServerError
	case ErrorNotFound:
		code = http.StatusNotFound
	case ErrorNotAcceptable:
		code = http.StatusNotAcceptable
	default:
		code = http.StatusInternalServerError
	}

	w.WriteHeader(code)

	response := Response[T]{
		Status:    "error",
		ErrorType: apiErr.Typ,
		Error:     apiErr.Err.Error(),
		Data:      data,
	}

	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		logger.Error("Failed to encode response", "err", err)
		w.Write([]byte("KO"))
	}
}
