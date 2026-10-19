package handlers

import (
	"net/http"
	"strconv"
)

// parsePagination extracts limit/offset query parameters with sane defaults,
// falling back silently on invalid or missing values.
func parsePagination(r *http.Request) (limit int, offset int) {
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
