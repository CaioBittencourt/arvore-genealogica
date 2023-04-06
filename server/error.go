package server

import (
	"net/http"

	"github.com/CaioBittencourt/arvore-genealogica/errors"
)

type ErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorCode    string `json:"errorCode"`
	StatusCode   int    `json:"-"`
}

type ErrorResponseConfiguration struct {
	ErrorShouldBeVisible bool
	ErrorStatusCode      int
}

var errorResponseConfigurationByApplicationErrorCode = map[errors.ApplicationErrorCode]ErrorResponseConfiguration{
	errors.PersonNotFoundErrorCode:          {ErrorShouldBeVisible: true, ErrorStatusCode: 404},
	errors.PersonNotFoundInGraph:            {ErrorShouldBeVisible: true, ErrorStatusCode: 404},
	errors.InvalidPersonNameErrorCode:       {ErrorShouldBeVisible: true, ErrorStatusCode: 400},
	errors.TooManyParentsForPersonErrorCode: {ErrorShouldBeVisible: true, ErrorStatusCode: 400},
	errors.InvalidPersonGenderErrorCode:     {ErrorShouldBeVisible: true, ErrorStatusCode: 400},
}

func (er ErrorResponse) Error() string {
	return er.ErrorMessage
}

func BuildErrorResponseFromError(err error) ErrorResponse {
	internalServerError := ErrorResponse{ErrorMessage: "Internal Server Error", StatusCode: http.StatusInternalServerError}

	applicationError, isApplicationError := errors.CastToApplicationError(err)
	if !isApplicationError {
		return internalServerError
	}

	configForError, ok := errorResponseConfigurationByApplicationErrorCode[applicationError.Code]
	if !ok {
		return internalServerError
	}

	if !configForError.ErrorShouldBeVisible {
		return internalServerError
	}

	return ErrorResponse{ErrorMessage: applicationError.Messsage, ErrorCode: string(applicationError.Code), StatusCode: configForError.ErrorStatusCode}
}
