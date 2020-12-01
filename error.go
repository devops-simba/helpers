package helpers

type StringError string

func (this StringError) Error() string { return string(this) }

const (
	ErrorInvalidArgument  StringError = "One or more invalid argument passed to the function"
	ErrorTooFewArguments  StringError = "Too few arguments passed to the function"
	ErrorTooManyArguments StringError = "Too many arguments passed to the function"
)
