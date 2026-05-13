package handler

import "net/http"

type HttpError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e HttpError) Error() string {
	return e.Message
}

func (e HttpError) WithCode(code string) HttpError {
	e.Code = code
	return e
}

func (e HttpError) WithMessage(message string, errors ...error) HttpError {
	if len(errors) > 0 {
		for _, err := range errors {
			message += ": " + err.Error() + "; "
		}
	}
	e.Message = message
	return e
}

var BadRequest = HttpError{
	StatusCode: http.StatusBadRequest,
	Code:       "BadRequest",
	Message:    "Bad request",
}

var InternalServerError = HttpError{
	StatusCode: http.StatusInternalServerError,
	Code:       "InternalServerError",
	Message:    "Internal server error",
}

var Forbidden = HttpError{
	StatusCode: http.StatusForbidden,
	Code:       "Forbidden",
	Message:    "Forbidden",
}

var Unauthorized = HttpError{
	StatusCode: http.StatusUnauthorized,
	Code:       "Unauthorized",
	Message:    "Unauthorized",
}

var NotFound = HttpError{
	StatusCode: http.StatusNotFound,
	Code:       "NotFound",
	Message:    "Not found",
}
