//go:build preproddemo

package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// NewAGACandidateDemoHandler exposes only the five reviewed read operations.
// The preproddemo-tagged runtime uses this handler instead of the canonical
// product API so no governed-domain mutation route is reachable.
func NewAGACandidateDemoHandler(service *aga.Service) http.Handler {
	api := &candidateDemoAPI{service: service}
	router := chi.NewRouter()
	router.Get("/v1/admin/governed-checklist/aga-candidate-demo/capability", api.getAGACandidateDemoCapability)
	router.Get("/v1/admin/governed-checklist/aga-candidate-demo/summary", api.getAGACandidateDemoSummary)
	router.Get("/v1/admin/governed-checklist/aga-candidate-demo/forms", api.listAGACandidateDemoForms)
	router.Get("/v1/admin/governed-checklist/aga-candidate-demo/forms/{formCode}", api.getAGACandidateDemoForm)
	router.Get("/v1/admin/governed-checklist/aga-candidate-demo/questions", api.listAGACandidateDemoQuestions)
	return router
}

// WithAGACandidateDemoNoStore applies the privacy headers before the OIDC
// boundary, so anonymous and stale-session denials cannot be cached either.
func WithAGACandidateDemoNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "private, no-store")
		writer.Header().Set("Pragma", "no-cache")
		writer.Header().Set("Vary", "Cookie")
		next.ServeHTTP(writer, request)
	})
}

// ProtectAGACandidateDemo binds the neutral existence-hiding denial to the
// ordinary OIDC/session boundary used by the tagged runtime.
func ProtectAGACandidateDemo(boundary *AuthBoundary, next http.Handler) http.Handler {
	if boundary == nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { agaDemoNotFound(writer) })
	}
	return WithAGACandidateDemoNoStore(boundary.ProtectReadOnlyNeutral(next, agaDemoNotFound))
}

type candidateDemoAPI struct {
	service *aga.Service
}

func (api *candidateDemoAPI) getAGACandidateDemoCapability(writer http.ResponseWriter, request *http.Request) {
	actor, ok := agaDemoActor(request)
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	output, err := api.service.Capability(request.Context(), actor)
	if err != nil || !output.Available {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, output)
}

func (api *candidateDemoAPI) getAGACandidateDemoSummary(writer http.ResponseWriter, request *http.Request) {
	actor, ok := agaDemoActor(request)
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	output, err := api.service.Summary(request.Context(), actor)
	if err != nil {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, output)
}

func (api *candidateDemoAPI) listAGACandidateDemoForms(writer http.ResponseWriter, request *http.Request) {
	actor, ok := agaDemoActor(request)
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	limit := 50
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			agaDemoNotFound(writer)
			return
		}
		limit = parsed
	}
	output, err := api.service.Forms(request.Context(), actor, request.URL.Query().Get("cursor"), limit)
	if err != nil {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, output)
}

func (api *candidateDemoAPI) getAGACandidateDemoForm(writer http.ResponseWriter, request *http.Request) {
	actor, ok := agaDemoActor(request)
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	output, err := api.service.Form(request.Context(), actor, chi.URLParam(request, "formCode"))
	if err != nil {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, output)
}

func (api *candidateDemoAPI) listAGACandidateDemoQuestions(writer http.ResponseWriter, request *http.Request) {
	actor, ok := agaDemoActor(request)
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	limit := 50
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			agaDemoNotFound(writer)
			return
		}
		limit = parsed
	}
	output, err := api.service.Questions(request.Context(), actor, request.URL.Query().Get("cursor"), request.URL.Query().Get("formCode"), request.URL.Query().Get("sourceGapCategory"), request.URL.Query().Get("riskBand"), limit)
	if err != nil {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, output)
}

func agaDemoActor(request *http.Request) (actor identity.Principal, ok bool) {
	actor, ok = PrincipalFromContext(request.Context())
	return actor, ok && actor.OrganizationID == "CAA" && actor.HasRole(identity.RoleAdmin)
}

func agaDemoNotFound(writer http.ResponseWriter) {
	const body = `{"error":"not found"}`
	writer.Header().Set("Cache-Control", "private, no-store")
	writer.Header().Set("Pragma", "no-cache")
	writer.Header().Set("Vary", "Cookie")
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(http.StatusNotFound)
	_, _ = writer.Write([]byte(body))
}

func agaDemoJSON(writer http.ResponseWriter, output any) {
	writer.Header().Set("Cache-Control", "private, no-store")
	writer.Header().Set("Pragma", "no-cache")
	writer.Header().Set("Vary", "Cookie")
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(output)
}
