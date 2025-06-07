package main

import (
	"fmt"
	"strings"
)

// ConfigError provides structured error information for config operations
type ConfigError struct {
	Op          string // operation being performed
	File        string // file involved
	Err         error  // underlying error
	Recoverable bool   // can operation be retried
	Context     map[string]string // additional context
}

func (e *ConfigError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.File, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ValidationError represents configuration validation failures
type ValidationError struct {
	Field   string
	Value   string
	Message string
	File    string
}

func (e *ValidationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("validation error in %s: %s (%s=%s)", e.File, e.Message, e.Field, e.Value)
	}
	return fmt.Sprintf("validation error: %s (%s=%s)", e.Message, e.Field, e.Value)
}

// OperationResult represents the result of a file operation
type OperationResult struct {
	File     string
	Success  bool
	Message  string
	Error    error
	Skipped  bool
	Backup   string // path to backup if created
}

// MultiError collects multiple errors from batch operations
type MultiError struct {
	Errors []error
	Op     string
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("%s: %v", e.Op, e.Errors[0])
	}
	
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("%s: multiple errors: %s", e.Op, strings.Join(messages, "; "))
}

func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Helper functions for creating specific error types
func NewConfigError(op, file string, err error) *ConfigError {
	return &ConfigError{
		Op:          op,
		File:        file,
		Err:         err,
		Recoverable: false,
		Context:     make(map[string]string),
	}
}

func NewRecoverableError(op, file string, err error) *ConfigError {
	return &ConfigError{
		Op:          op,
		File:        file,
		Err:         err,
		Recoverable: true,
		Context:     make(map[string]string),
	}
}

func NewValidationError(field, value, message, file string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		File:    file,
	}
}

// Error classification helpers
func IsRecoverable(err error) bool {
	if configErr, ok := err.(*ConfigError); ok {
		return configErr.Recoverable
	}
	return false
}

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}
