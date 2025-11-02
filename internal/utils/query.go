package utils

import (
	"net/http"
	"strconv"
)

func GetQueryParam[T string | int](r *http.Request, key string, defaultVal T) T {
	qVal := r.URL.Query().Get(key)
	if qVal == "" {
		return defaultVal
	}
	var result T
	switch any(result).(type) {
	case string:
		return any(qVal).(T)
	case int:
		intVal, err := strconv.Atoi(qVal)
		if err != nil || intVal < 0 {
			return defaultVal
		}
		result = any(intVal).(T)
	}

	return result
}
