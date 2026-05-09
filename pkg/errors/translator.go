package errors

import "net/http"

func ToHTTPStatus(code string) int {
	switch code {
	case "bad_request", "invalid_argument":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
