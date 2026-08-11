package response

import (
	"encoding/json"
	"net/http"

	apperrors "kindred_server/internal/platform/errors"
)

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, err error) {
	appErr := apperrors.ToAppError(err)
	JSON(w, appErr.Status, ErrorBody{Error: ErrorDetail{Code: appErr.Code, Message: appErr.Message}})
}
