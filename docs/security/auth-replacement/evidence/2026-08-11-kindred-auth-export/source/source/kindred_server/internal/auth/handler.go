package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/internal/platform/router"
	"kindred_server/pkg/response"
)

type Handler struct {
	service *Service
	userID  func(*http.Request) string
}

func NewHandler(service *Service, userID func(*http.Request) string) *Handler {
	return &Handler{service: service, userID: userID}
}

// RegisterRoutes wires auth endpoints. The `limiter` middleware (may be nil)
// is applied to sensitive endpoints vulnerable to enumeration or brute force.
func RegisterRoutes(r *router.Router, h *Handler, limiter func(http.Handler) http.Handler, auth func(http.Handler) http.Handler) {
	wrap := func(next http.Handler) http.Handler {
		if limiter == nil {
			return next
		}
		return limiter(next)
	}
	r.Handle(http.MethodPost, "/auth/register", wrap(http.HandlerFunc(h.Register)))
	r.Handle(http.MethodPost, "/auth/login", wrap(http.HandlerFunc(h.Login)))
	r.Handle(http.MethodPost, "/auth/reactivate", wrap(http.HandlerFunc(h.Reactivate)))
	r.Handle(http.MethodPost, "/auth/refresh", wrap(http.HandlerFunc(h.Refresh)))
	r.Handle(http.MethodPost, "/auth/logout", wrap(http.HandlerFunc(h.Logout)))
	r.Handle(http.MethodPost, "/auth/verify-email", wrap(http.HandlerFunc(h.VerifyEmail)))
	r.Handle(http.MethodPost, "/auth/resend-verification", wrap(http.HandlerFunc(h.ResendVerification)))
	r.Handle(http.MethodPost, "/auth/password/forgot", wrap(http.HandlerFunc(h.RequestPasswordReset)))
	r.Handle(http.MethodPost, "/auth/password/reset", wrap(http.HandlerFunc(h.ResetPassword)))
	if auth != nil {
		r.Handle(http.MethodPost, "/me/phone/verification/start", auth(http.HandlerFunc(h.StartPhoneVerification)))
		r.Handle(http.MethodPost, "/me/phone/verification/verify", auth(http.HandlerFunc(h.VerifyPhone)))
		r.Handle(http.MethodPost, "/me/phone/change/start", auth(http.HandlerFunc(h.StartPhoneChange)))
		r.Handle(http.MethodPost, "/me/phone/change/verify", auth(http.HandlerFunc(h.VerifyPhoneChange)))
		r.Handle(http.MethodPost, "/auth/password/change", auth(http.HandlerFunc(h.ChangePassword)))
		r.Handle(http.MethodPost, "/me/deactivation", auth(http.HandlerFunc(h.Deactivate)))
		r.Handle(http.MethodDelete, "/me", auth(http.HandlerFunc(h.DeleteAccount)))
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if err := h.service.Logout(r.Context(), req); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.Register(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.Login(r.Context(), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *Handler) Reactivate(w http.ResponseWriter, r *http.Request) {
	var req ReactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.Reactivate(r.Context(), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func writeAuthError(w http.ResponseWriter, err error) {
	var lockedErr *AccountLockedError
	if errors.As(err, &lockedErr) {
		seconds := int(lockedErr.RetryAfter.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}
	response.Error(w, err)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.VerifyEmail(r.Context(), req); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.ResendVerification(r.Context(), req.Email); err != nil {
		response.Error(w, err)
		return
	}
	// Always 202 to avoid leaking whether the email exists.
	response.JSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.RequestPasswordReset(r.Context(), req.Email); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"reset": true})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.ChangePassword(r.Context(), h.userID(r), req); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) StartPhoneVerification(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.StartPhoneVerification(r.Context(), h.userID(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, out)
}

func (h *Handler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	var req VerifyPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.VerifyPhone(r.Context(), h.userID(r), req); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (h *Handler) StartPhoneChange(w http.ResponseWriter, r *http.Request) {
	var req StartPhoneChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.StartPhoneChange(r.Context(), h.userID(r), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, out)
}

func (h *Handler) VerifyPhoneChange(w http.ResponseWriter, r *http.Request) {
	var req VerifyPhoneChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.VerifyPhoneChange(r.Context(), h.userID(r), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	var req PasswordStepUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.service.Deactivate(r.Context(), h.userID(r), req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req PasswordStepUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperrors.BadRequest("invalid JSON body"))
		return
	}
	out, err := h.service.DeleteAccount(r.Context(), h.userID(r), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, out)
}
