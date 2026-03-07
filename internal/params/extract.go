package params

import (
	"errors"
	"fmt"
	"strings"
)

// RetryableError marks errors that allow the user to retry selection.
type RetryableError interface {
	error
	Retryable() bool
}

// ListExtractError reports that a selected row could not satisfy the requested
// 1-based field extraction.
type ListExtractError struct {
	FieldIndex int
	FieldCount int
}

func (e *ListExtractError) Error() string {
	return fmt.Sprintf("selected row does not contain field %d", e.FieldIndex)
}

// Retryable reports that users can choose a different row and try again.
func (e *ListExtractError) Retryable() bool {
	return true
}

// IsRetryable reports whether an error supports retrying the same selection flow.
func IsRetryable(err error) bool {
	var retryable RetryableError
	return errors.As(err, &retryable) && retryable.Retryable()
}

// ExtractListValue returns the full raw row unless both a literal delimiter and
// a 1-based field index are configured. When extraction is configured, the
// selected field is trimmed before returning.
func ExtractListValue(raw, delimiter string, fieldIndex int) (string, error) {
	if fieldIndex <= 0 || delimiter == "" {
		return raw, nil
	}

	parts := strings.Split(raw, delimiter)
	if fieldIndex > len(parts) {
		return "", &ListExtractError{FieldIndex: fieldIndex, FieldCount: len(parts)}
	}

	return strings.TrimSpace(parts[fieldIndex-1]), nil
}
