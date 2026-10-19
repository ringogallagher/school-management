package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// actorFromRequest builds the services.Actor for the authenticated user
func actorFromRequest(r *http.Request) (services.Actor, bool) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		return services.Actor{}, false
	}
	return services.Actor{UserID: claims.UserID, Role: models.UserRole(claims.Role)}, true
}

// urlID parses a numeric chi URL parameter
func urlID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}

// writeServiceError maps service errors to HTTP status codes:
// ErrForbidden -> 403, ErrNotFound -> 404, anything else -> 400 (validation / business rule)
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrForbidden):
		writeError(w, http.StatusForbidden, "you do not have access to this resource")
	case errors.Is(err, services.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func writeOK(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, &models.SuccessResponse{Code: http.StatusOK, Message: message, Data: data})
}

func writeCreated(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusCreated, &models.SuccessResponse{Code: http.StatusCreated, Message: message, Data: data})
}
