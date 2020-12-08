package helpers

import (
	"errors"
	"fmt"
)

//region StringError
// StringError this type is just like `errors.New` but it may be declared as `const`
type StringError string

func (this StringError) Error() string { return string(this) }

const (
	ErrInvalidArgument   StringError = "One or more invalid argument passed to the function"
	ErrTooFewArguments   StringError = "Too few arguments passed to the function"
	ErrTooManyArguments  StringError = "Too many arguments passed to the function"
	ErrOutOfRange        StringError = "Argument is out of valid range"
	ErrOperationTimedOut StringError = "Operation timed out"
)

//endregion

//region ComponentError
// ComponentError this error indicate a situation when a subcomponent failed
type ComponentError struct {
	Component interface{}
	Failure   error
}

func (this ComponentError) Error() string {
	named, ok := this.Component.(Named)
	if ok {
		return fmt.Sprintf("Execution of `%s` failed: %v", named.GetName(), this.Failure)
	}
	return fmt.Sprintf("Sub component execution failed: %v", this.Failure)
}
func (this ComponentError) Is(err error) bool {
	return errors.Is(this.Failure, err)
}
func (this ComponentError) As(target interface{}) bool {
	return errors.As(this.Failure, target)
}
func (this ComponentError) Unwrap() error {
	return this.Failure
}

//endregion

//region AggregateError
type AggregateErrorBuilder struct {
	Errors AggregateError
}

func (this *AggregateErrorBuilder) AddError(err error) {
	if err != nil {
		this.Errors = append(this.Errors, err)
	}
}
func (this *AggregateErrorBuilder) GetError() error {
	if len(this.Errors) == 0 {
		return nil
	}
	if len(this.Errors) == 1 {
		return this.Errors[0]
	}
	return this.Errors
}

type AggregateError []error

func (this AggregateError) Error() string {
	if len(this) == 0 {
		return ""
	}
	if len(this) == 1 {
		return this[0].Error()
	}

	return "Multiple operations failed"
}
func (this AggregateError) Is(err error) bool {
	for i := 0; i < len(this); i++ {
		if errors.Is(this[i], err) {
			return true
		}
	}
	return false
}
func (this AggregateError) As(target interface{}) bool {
	for i := 0; i < len(this); i++ {
		if errors.As(this[i], target) {
			return true
		}
	}
	return false
}

//endregion
