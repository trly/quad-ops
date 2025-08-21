package systemd

import (
	"fmt"
)

// Error represents an error from systemd operations.
type Error struct {
	Operation string // The operation that failed (Start, Stop, Restart, etc.)
	UnitName  string // The name of the unit
	UnitType  string // The type of the unit
	Cause     error  // The underlying error
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("systemd %s failed for %s.%s: %v", e.Operation, e.UnitName, e.UnitType, e.Cause)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new Error with the given details.
func NewError(operation, unitName, unitType string, cause error) *Error {
	return &Error{
		Operation: operation,
		UnitName:  unitName,
		UnitType:  unitType,
		Cause:     cause,
	}
}

// ConnectionError represents an error connecting to systemd.
type ConnectionError struct {
	UserMode bool  // Whether this was a user or system connection attempt
	Cause    error // The underlying error
}

// Error implements the error interface.
func (e *ConnectionError) Error() string {
	mode := "system"
	if e.UserMode {
		mode = "user"
	}
	return fmt.Sprintf("failed to connect to systemd %s bus: %v", mode, e.Cause)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a new ConnectionError.
func NewConnectionError(userMode bool, cause error) *ConnectionError {
	return &ConnectionError{
		UserMode: userMode,
		Cause:    cause,
	}
}

// UnitNotFoundError represents an error when a unit cannot be found.
type UnitNotFoundError struct {
	UnitName string
	UnitType string
}

// Error implements the error interface.
func (e *UnitNotFoundError) Error() string {
	return fmt.Sprintf("unit not found: %s.%s", e.UnitName, e.UnitType)
}

// NewUnitNotFoundError creates a new UnitNotFoundError.
func NewUnitNotFoundError(unitName, unitType string) *UnitNotFoundError {
	return &UnitNotFoundError{
		UnitName: unitName,
		UnitType: unitType,
	}
}

// IsConnectionError checks if an error is a ConnectionError.
func IsConnectionError(err error) bool {
	_, ok := err.(*ConnectionError)
	return ok
}

// IsError checks if an error is a systemd Error.
func IsError(err error) bool {
	_, ok := err.(*Error)
	return ok
}

// IsUnitNotFoundError checks if an error is a UnitNotFoundError.
func IsUnitNotFoundError(err error) bool {
	_, ok := err.(*UnitNotFoundError)
	return ok
}
