package compose

import (
	"errors"
	"fmt"
)

// fileNotFoundError indicates that a compose file could not be found.
type fileNotFoundError struct {
	path  string
	cause error
}

func (e *fileNotFoundError) Error() string {
	return fmt.Sprintf("compose file not found: %s", e.path)
}

func (e *fileNotFoundError) Unwrap() error {
	return e.cause
}

// IsFileNotFoundError checks if an error is a fileNotFoundError.
func IsFileNotFoundError(err error) bool {
	var ferr *fileNotFoundError
	return errors.As(err, &ferr)
}

// invalidYAMLError indicates that the compose file contains invalid YAML.
type invalidYAMLError struct {
	cause error
}

func (e *invalidYAMLError) Error() string {
	return fmt.Sprintf("invalid YAML in compose file: %v", e.cause)
}

func (e *invalidYAMLError) Unwrap() error {
	return e.cause
}

// IsInvalidYAMLError checks if an error is an invalidYAMLError.
func IsInvalidYAMLError(err error) bool {
	var yerr *invalidYAMLError
	return errors.As(err, &yerr)
}

// validationError indicates that the compose project failed validation.
type validationError struct {
	message string
	cause   error
}

func (e *validationError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("validation failed: %s (%v)", e.message, e.cause)
	}
	return fmt.Sprintf("validation failed: %s", e.message)
}

func (e *validationError) Unwrap() error {
	return e.cause
}

// IsValidationError checks if an error is a validationError.
func IsValidationError(err error) bool {
	var verr *validationError
	return errors.As(err, &verr)
}

// pathError indicates an error related to file paths or path resolution.
type pathError struct {
	path  string
	cause error
}

func (e *pathError) Error() string {
	return fmt.Sprintf("path error: %s (%v)", e.path, e.cause)
}

func (e *pathError) Unwrap() error {
	return e.cause
}

// IsPathError checks if an error is a pathError.
func IsPathError(err error) bool {
	var perr *pathError
	return errors.As(err, &perr)
}

// loaderError indicates an error during loading/parsing of the compose file.
type loaderError struct {
	cause error
}

func (e *loaderError) Error() string {
	return fmt.Sprintf("failed to load compose file: %v", e.cause)
}

func (e *loaderError) Unwrap() error {
	return e.cause
}

// IsLoaderError checks if an error is a loaderError.
func IsLoaderError(err error) bool {
	var lerr *loaderError
	return errors.As(err, &lerr)
}

// quadletCompatibilityError indicates that the compose project cannot be converted to podman-systemd quadlets.
type quadletCompatibilityError struct {
	message string
	cause   error
}

func (e *quadletCompatibilityError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("quadlet compatibility error: %s (%v)", e.message, e.cause)
	}
	return fmt.Sprintf("quadlet compatibility error: %s", e.message)
}

func (e *quadletCompatibilityError) Unwrap() error {
	return e.cause
}

// IsQuadletCompatibilityError checks if an error is a quadletCompatibilityError.
func IsQuadletCompatibilityError(err error) bool {
	var qerr *quadletCompatibilityError
	return errors.As(err, &qerr)
}
