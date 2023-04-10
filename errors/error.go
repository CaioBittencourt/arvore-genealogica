package errors

type ApplicationErrorCode string

const (
	PersonNotFoundErrorCode          ApplicationErrorCode = "PERSON_NOT_FOUND"
	PersonNotFoundInGraph            ApplicationErrorCode = "PERSON_NOT_FOUND_IN_GRAPH"
	InvalidPersonNameErrorCode       ApplicationErrorCode = "INVALID_PERSON_NAME"
	TooManyParentsForPersonErrorCode ApplicationErrorCode = "TOO_MANY_PARENTS_FOR_PERSON"
	InvalidPersonGenderErrorCode     ApplicationErrorCode = "INVALID_PERSON_GENDER"
	ChildrenAlreadyHasTwoParents     ApplicationErrorCode = "CHILDREN_ALREADY_HAVE_TWO_PARENTS"
)

type ApplicationError struct {
	Messsage string
	Code     ApplicationErrorCode
}

func (ae ApplicationError) Error() string {
	return ae.Messsage
}

func NewApplicationError(message string, code ApplicationErrorCode) ApplicationError {
	return ApplicationError{Messsage: message, Code: code}
}

func CastToApplicationError(err error) (*ApplicationError, bool) {
	e, ok := err.(*ApplicationError)
	if ok {
		return e, true
	}

	e2, ok := err.(ApplicationError)
	if ok {
		return &e2, true
	}

	return nil, false
}

func ErrorHasCode(err error, code ApplicationErrorCode) bool {
	applicationError, ok := CastToApplicationError(err)
	if !ok {
		return false
	}

	return applicationError.Code == code
}
