package diagram_test

import (
	"errors"
	"testing"

	"github.com/iokdigital/go-mermaid"
)

func TestErrInvalidFormat(t *testing.T) {
	if !errors.Is(diagram.ErrInvalidFormat, diagram.ErrInvalidFormat) {
		t.Error("ErrInvalidFormat should match itself")
	}
}

func TestErrRendererNotAvailable(t *testing.T) {
	if !errors.Is(diagram.ErrRendererNotAvailable, diagram.ErrRendererNotAvailable) {
		t.Error("ErrRendererNotAvailable should match itself")
	}
}

func TestErrPNGSizeLimitExceeded(t *testing.T) {
	if !errors.Is(diagram.ErrPNGSizeLimitExceeded, diagram.ErrPNGSizeLimitExceeded) {
		t.Error("ErrPNGSizeLimitExceeded should match itself")
	}
}

func TestErrDuplicateNodeID(t *testing.T) {
	if !errors.Is(diagram.ErrDuplicateNodeID, diagram.ErrDuplicateNodeID) {
		t.Error("ErrDuplicateNodeID should match itself")
	}
}

func TestFallbackFormatError(t *testing.T) {
	err := &diagram.FallbackFormatError{
		Err:      diagram.ErrRendererNotAvailable,
		Fallback: diagram.FormatHTML,
	}

	if err.Error() != diagram.ErrRendererNotAvailable.Error() {
		t.Errorf("expected error message %q, got %q",
			diagram.ErrRendererNotAvailable.Error(), err.Error())
	}

	if !errors.Is(err, diagram.ErrRendererNotAvailable) {
		t.Error("FallbackFormatError should unwrap to ErrRendererNotAvailable")
	}

	if err.FallbackFormat() != diagram.FormatHTML {
		t.Errorf("expected fallback format %q, got %q",
			diagram.FormatHTML, err.FallbackFormat())
	}
}

func TestFallbackFormatErrorFromPNG(t *testing.T) {
	err := &diagram.FallbackFormatError{
		Err:      diagram.ErrPNGSizeLimitExceeded,
		Fallback: diagram.FormatHTML,
	}

	if err.FallbackFormat() != diagram.FormatHTML {
		t.Errorf("expected fallback %q, got %q", diagram.FormatHTML, err.FallbackFormat())
	}

	if !errors.Is(err, diagram.ErrPNGSizeLimitExceeded) {
		t.Error("should unwrap to ErrPNGSizeLimitExceeded")
	}
}

func TestErrorsDistinct(t *testing.T) {
	errs := []error{
		diagram.ErrInvalidFormat,
		diagram.ErrRendererNotAvailable,
		diagram.ErrPNGSizeLimitExceeded,
		diagram.ErrDuplicateNodeID,
	}

	for i := range errs {
		for j := range errs {
			if i != j && errors.Is(errs[i], errs[j]) {
				t.Errorf("errs[%d] should not match errs[%d]", i, j)
			}
		}
	}
}

func TestErrInvalidFormatMessage(t *testing.T) {
	if diagram.ErrInvalidFormat.Error() == "" {
		t.Error("ErrInvalidFormat should have non-empty message")
	}
}

func TestErrRendererNotAvailableMessage(t *testing.T) {
	if diagram.ErrRendererNotAvailable.Error() == "" {
		t.Error("ErrRendererNotAvailable should have non-empty message")
	}
}

func TestErrPNGSizeLimitExceededMessage(t *testing.T) {
	if diagram.ErrPNGSizeLimitExceeded.Error() == "" {
		t.Error("ErrPNGSizeLimitExceeded should have non-empty message")
	}
}

func TestErrDuplicateNodeIDMessage(t *testing.T) {
	if diagram.ErrDuplicateNodeID.Error() == "" {
		t.Error("ErrDuplicateNodeID should have non-empty message")
	}
}