package errors

import (
	"errors"
	"fmt"
)

// Standard errors
var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")
	
	// ErrInvalidInput is returned when the input is invalid
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrPermissionDenied is returned when the entity/user lacks permission
	ErrPermissionDenied = errors.New("permission denied")
	
	// ErrEntityNotFound is returned when an entity is not found
	ErrEntityNotFound = errors.New("entity not found")
	
	// ErrLTMUnavailable is returned when the LTM storage is unavailable
	ErrLTMUnavailable = errors.New("long-term memory store unavailable")
	
	// ErrLuaExecution is returned when there's an error executing a Lua script
	ErrLuaExecution = errors.New("lua script execution error")
)

// Wrap wraps an error with additional context
func Wrap(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// Is reports whether any error in err's tree matches target.
// This is a convenience function that wraps errors.Is
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
// This is a convenience function that wraps errors.As
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
