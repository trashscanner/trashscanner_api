package utils

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetQueryParam_String(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns value when present",
			queryString:  "name=John",
			key:          "name",
			defaultValue: "default",
			expected:     "John",
		},
		{
			name:         "returns default when missing",
			queryString:  "",
			key:          "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "returns default when empty",
			queryString:  "name=",
			key:          "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "handles special characters",
			queryString:  "query=hello%20world",
			key:          "query",
			defaultValue: "",
			expected:     "hello world",
		},
		{
			name:         "handles numeric string",
			queryString:  "code=12345",
			key:          "code",
			defaultValue: "000",
			expected:     "12345",
		},
		{
			name:         "retrieves from multiple parameters",
			queryString:  "name=Alice&age=30&city=Moscow",
			key:          "city",
			defaultValue: "Unknown",
			expected:     "Moscow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					RawQuery: tt.queryString,
				},
			}

			result := GetQueryParam(req, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetQueryParam_Int(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "returns value when present",
			queryString:  "limit=100",
			key:          "limit",
			defaultValue: 10,
			expected:     100,
		},
		{
			name:         "returns default when missing",
			queryString:  "",
			key:          "limit",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "returns default when invalid",
			queryString:  "limit=invalid",
			key:          "limit",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "handles zero value",
			queryString:  "offset=0",
			key:          "offset",
			defaultValue: 20,
			expected:     0,
		},
		{
			name:         "handles negative value",
			queryString:  "delta=-5",
			key:          "delta",
			defaultValue: 0,
			expected:     0,
		},
		{
			name:         "handles large numbers",
			queryString:  "max=999999",
			key:          "max",
			defaultValue: 100,
			expected:     999999,
		},
		{
			name:         "returns default for empty string",
			queryString:  "count=",
			key:          "count",
			defaultValue: 42,
			expected:     42,
		},
		{
			name:         "retrieves from multiple parameters",
			queryString:  "name=Alice&age=30&city=Moscow",
			key:          "age",
			defaultValue: 0,
			expected:     30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					RawQuery: tt.queryString,
				},
			}

			result := GetQueryParam(req, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
