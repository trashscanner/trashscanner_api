package errlocal

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseError_Error(t *testing.T) {
	err := &BaseError{
		Msg: "test error",
		Sys: "test_system",
		DetailsMap: map[string]any{
			"key1": "value1",
			"key2": 42,
		},
	}

	errStr := err.Error()

	assert.Contains(t, errStr, "message: test error")
	assert.Contains(t, errStr, "system: test_system")
	assert.Contains(t, errStr, "details:")
	assert.Contains(t, errStr, "key1: value1")
	assert.Contains(t, errStr, "key2: 42")
}

func TestBaseError_Error_EmptyMessage(t *testing.T) {
	err := &BaseError{
		Msg: "",
		Sys: "test_system",
		DetailsMap: map[string]any{
			"key": "value",
		},
	}

	errStr := err.Error()

	assert.NotContains(t, errStr, "message:")
	assert.Contains(t, errStr, "system: test_system")
	assert.Contains(t, errStr, "details:")
}

func TestBaseError_Error_EmptySystem(t *testing.T) {
	err := &BaseError{
		Msg:        "test error",
		Sys:        "",
		DetailsMap: map[string]any{},
	}

	errStr := err.Error()

	assert.Contains(t, errStr, "message: test error")
	assert.NotContains(t, errStr, "Sys:")
	assert.NotContains(t, errStr, "details:")
}

func TestBaseError_Error_EmptyDetails(t *testing.T) {
	err := &BaseError{
		Msg:        "test error",
		Sys:        "test_system",
		DetailsMap: nil,
	}

	errStr := err.Error()

	assert.Contains(t, errStr, "message: test error")
	assert.Contains(t, errStr, "system: test_system")
	assert.NotContains(t, errStr, "details:")
}

func TestBaseError_Message(t *testing.T) {
	err := &BaseError{
		Msg: "test message",
		Sys: "system",
	}

	assert.Equal(t, "test message", err.Message())
}

func TestBaseError_System(t *testing.T) {
	err := &BaseError{
		Msg: "message",
		Sys: "test_system",
	}

	assert.Equal(t, "test_system", err.System())
}

func TestBaseError_Details(t *testing.T) {
	details := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	err := &BaseError{
		Msg:        "message",
		Sys:        "system",
		DetailsMap: details,
	}

	result := err.Details()

	assert.Equal(t, details, result)
	assert.Len(t, result, 2)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, 123, result["key2"])
}

func TestBaseError_Code(t *testing.T) {
	err := &BaseError{
		Msg: "message",
	}

	assert.Equal(t, 500, err.Code())
}

func TestBaseError_Base(t *testing.T) {
	err := &BaseError{
		Msg: "message",
		Sys: "system",
	}

	base := err.Base()

	assert.Equal(t, err, base)
	assert.Equal(t, "message", base.Msg)
	assert.Equal(t, "system", base.Sys)
}

func TestNewErrInternal(t *testing.T) {
	details := map[string]any{"error": "database connection failed"}

	err := NewErrInternal("internal error", "database", details)

	assert.NotNil(t, err)
	assert.Equal(t, "internal error", err.Message())
	assert.Equal(t, "database", err.System())
	assert.Equal(t, details, err.Details())
	assert.Equal(t, http.StatusInternalServerError, err.Code())
}

func TestErrInternal_Code(t *testing.T) {
	err := &ErrInternal{
		BaseError: BaseError{
			Msg: "error",
		},
	}

	assert.Equal(t, http.StatusInternalServerError, err.Code())
	assert.Equal(t, 500, err.Code())
}

func TestErrInternal_AsLocalError(t *testing.T) {
	var localErr LocalError = NewErrInternal("test", "system", nil)

	assert.NotNil(t, localErr)
	assert.Equal(t, "test", localErr.Message())
	assert.Equal(t, 500, localErr.Code())
}

func TestNewErrNotFound(t *testing.T) {
	details := map[string]any{"id": 123}

	err := NewErrNotFound("user not found", "user_service", details)

	assert.NotNil(t, err)
	assert.Equal(t, "user not found", err.Message())
	assert.Equal(t, "user_service", err.System())
	assert.Equal(t, details, err.Details())
	assert.Equal(t, http.StatusNotFound, err.Code())
}

func TestErrNotFound_Code(t *testing.T) {
	err := &ErrNotFound{
		BaseError: BaseError{
			Msg: "not found",
		},
	}

	assert.Equal(t, http.StatusNotFound, err.Code())
	assert.Equal(t, 404, err.Code())
}

func TestErrNotFound_AsLocalError(t *testing.T) {
	var localErr LocalError = NewErrNotFound("resource not found", "api", nil)

	assert.NotNil(t, localErr)
	assert.Equal(t, "resource not found", localErr.Message())
	assert.Equal(t, 404, localErr.Code())
}

func TestLocalError_Interface(t *testing.T) {
	testCases := []struct {
		name         string
		err          LocalError
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "ErrInternal",
			err:          NewErrInternal("internal", "sys", nil),
			expectedCode: 500,
			expectedMsg:  "internal",
		},
		{
			name:         "ErrNotFound",
			err:          NewErrNotFound("not found", "sys", nil),
			expectedCode: 404,
			expectedMsg:  "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedCode, tc.err.Code())
			assert.Equal(t, tc.expectedMsg, tc.err.Message())
			assert.NotNil(t, tc.err.Base())
		})
	}
}

func TestBaseError_Error_AllFieldsEmpty(t *testing.T) {
	err := &BaseError{}

	errStr := err.Error()

	assert.Equal(t, "", errStr)
}

func TestBaseError_Error_ComplexDetails(t *testing.T) {
	err := &BaseError{
		Msg: "complex error",
		Sys: "system",
		DetailsMap: map[string]any{
			"string": "text",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"nil":    nil,
			"array":  []int{1, 2, 3},
			"map":    map[string]string{"nested": "value"},
		},
	}

	errStr := err.Error()

	assert.Contains(t, errStr, "message: complex error")
	assert.Contains(t, errStr, "system: system")
	assert.Contains(t, errStr, "details:")
	assert.Contains(t, errStr, "string: text")
	assert.Contains(t, errStr, "int: 42")
	assert.Contains(t, errStr, "float: 3.14")
	assert.Contains(t, errStr, "bool: true")
}

func TestNewErrInternal_NilDetails(t *testing.T) {
	err := NewErrInternal("error", "system", nil)

	assert.NotNil(t, err)
	assert.Nil(t, err.Details())
	assert.NotContains(t, err.Error(), "details:")
}

func TestNewErrNotFound_NilDetails(t *testing.T) {
	err := NewErrNotFound("not found", "system", nil)

	assert.NotNil(t, err)
	assert.Nil(t, err.Details())
	assert.NotContains(t, err.Error(), "details:")
}

func TestBaseError_DetailsImmutability(t *testing.T) {
	originalDetails := map[string]any{
		"key": "value",
	}
	err := &BaseError{
		Msg:        "test",
		Sys:        "system",
		DetailsMap: originalDetails,
	}

	retrievedDetails := err.Details()
	retrievedDetails["new_key"] = "new_value"

	assert.Equal(t, "new_value", err.Details()["new_key"])
}
