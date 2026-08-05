package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
	"github.com/go-chi/chi/v5"
)

const workspaceAPIBase = "/v1/preprod/aga-demo-workspace"

var workspaceRevisionETag = regexp.MustCompile(`^"rev-([1-9][0-9]*)"$`)

// NewAGADemoWorkspaceHandler contains the supplemental workspace API only. It
// is intentionally mounted separately from the accepted five-route candidate
// demo handler so the old surface cannot inherit workspace commands.
func NewAGADemoWorkspaceHandler(service *workspace.Service) http.Handler {
	api := &agaDemoWorkspaceAPI{service: service}
	router := chi.NewRouter()
	router.Get(workspaceAPIBase+"/capability", api.capability)
	router.Post(workspaceAPIBase+"/classification/query", api.query(workspace.FamilyClassificationQuery))
	router.Post(workspaceAPIBase+"/classification/commands", api.command(workspace.FamilyClassificationCommand))
	router.Post(workspaceAPIBase+"/recommendations/commands", api.command(workspace.FamilyRecommendationCommand))
	router.Post(workspaceAPIBase+"/lifecycle/query", api.query(workspace.FamilyLifecycleQuery))
	router.Post(workspaceAPIBase+"/lifecycle/commands", api.command(workspace.FamilyLifecycleCommand))
	router.Post(workspaceAPIBase+"/admin/commands", api.command(workspace.FamilyAdminCommand))
	return router
}

// ProtectAGADemoWorkspace is a mutation-aware neutral boundary. It applies
// privacy headers before authentication, validates CSRF before body parsing,
// and establishes broad workspace authority before the handler can inspect a
// request body or query a domain object.
func ProtectAGADemoWorkspace(boundary *AuthBoundary, service *workspace.Service, next http.Handler) http.Handler {
	if boundary == nil || service == nil || next == nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { agaDemoNotFound(writer) })
	}
	return WithAGACandidateDemoNoStore(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		sink := &discardResponseWriter{header: make(http.Header)}
		principal, ok := boundary.authenticate(sink, request)
		if !ok {
			agaDemoNotFound(writer)
			return
		}
		if isMutation(request.Method) && !boundary.validateCSRF(sink, request, principal.SessionID) {
			agaDemoNotFound(writer)
			return
		}
		if !service.HasBroadAuthority(request.Context(), principal) {
			agaDemoNotFound(writer)
			return
		}
		ctx := contextWithPrincipal(request, principal)
		next.ServeHTTP(writer, request.WithContext(ctx))
	}))
}

type agaDemoWorkspaceAPI struct {
	service *workspace.Service
}

func (api *agaDemoWorkspaceAPI) capability(writer http.ResponseWriter, request *http.Request) {
	principal, ok := PrincipalFromContext(request.Context())
	if !ok || api.service == nil {
		agaDemoNotFound(writer)
		return
	}
	capability, err := api.service.Capability(request.Context(), principal)
	if err != nil || !capability.Available {
		agaDemoNotFound(writer)
		return
	}
	agaDemoJSON(writer, capability)
}

func (api *agaDemoWorkspaceAPI) query(family workspace.OperationFamily) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := PrincipalFromContext(request.Context())
		if !ok || api.service == nil {
			agaDemoNotFound(writer)
			return
		}
		var body workspace.QueryRequest
		if err := decodeWorkspaceJSON(request, &body); err != nil {
			writeProblem(writer, http.StatusBadRequest, "Invalid workspace query", "the closed query body is invalid", "AGA_WORKSPACE_QUERY_INVALID")
			return
		}
		if !queryBelongsToFamily(body.OperationID, family) {
			agaDemoNotFound(writer)
			return
		}
		output, err := api.service.Query(request.Context(), principal, body)
		if err != nil {
			api.writeServiceError(writer, err)
			return
		}
		agaDemoJSON(writer, output)
	}
}

func (api *agaDemoWorkspaceAPI) command(family workspace.OperationFamily) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := PrincipalFromContext(request.Context())
		if !ok || api.service == nil {
			agaDemoNotFound(writer)
			return
		}
		var body workspace.CommandEnvelope
		if err := decodeWorkspaceJSON(request, &body); err != nil {
			writeProblem(writer, http.StatusBadRequest, "Invalid workspace command", "the closed command body is invalid", "AGA_WORKSPACE_COMMAND_INVALID")
			return
		}
		if err := validateWorkspaceHeaders(request, family, body); err != nil {
			if errors.Is(err, workspace.ErrNeutralDenied) {
				agaDemoNotFound(writer)
				return
			}
			writeProblem(writer, http.StatusPreconditionFailed, "Workspace command conflict", "the command header and body compare-and-swap values do not match", "AGA_WORKSPACE_CAS_MISMATCH")
			return
		}
		output, err := api.service.Command(request.Context(), principal, family, body)
		if err != nil {
			api.writeServiceError(writer, err)
			return
		}
		agaDemoJSON(writer, output)
	}
}

func (api *agaDemoWorkspaceAPI) writeServiceError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workspace.ErrNeutralDenied), errors.Is(err, workspace.ErrRecommendationFactsUnavailable), errors.Is(err, workspace.ErrRecommendationAmbiguous), errors.Is(err, workspace.ErrLifecycleNotFound), errors.Is(err, workspace.ErrLifecycleRecommendationStale), errors.Is(err, workspace.ErrLifecycleBindingMismatch), errors.Is(err, preprod.ErrWorkspaceGeneration), errors.Is(err, preprod.ErrWorkspaceNotSealed):
		agaDemoNotFound(writer)
	case errors.Is(err, workspace.ErrMalformedCommand):
		writeProblem(writer, http.StatusBadRequest, "Invalid workspace command", "the command envelope is incomplete", "AGA_WORKSPACE_COMMAND_INVALID")
	case errors.Is(err, preprod.ErrWorkspaceCAS), errors.Is(err, workspace.ErrLifecycleConflict):
		writeProblem(writer, http.StatusPreconditionFailed, "Workspace command conflict", "the current workspace revision does not match the command", "AGA_WORKSPACE_CAS_CONFLICT")
	case errors.Is(err, preprod.ErrWorkspaceIdempotency):
		writeProblem(writer, http.StatusConflict, "Workspace command conflict", "the idempotency binding does not match the command", "AGA_WORKSPACE_IDEMPOTENCY_CONFLICT")
	case errors.Is(err, workspace.ErrCapabilityUnavailable):
		writeProblem(writer, http.StatusServiceUnavailable, "Workspace capability unavailable", "the synthetic lifecycle capability is not wired in this artifact", "AGA_WORKSPACE_CAPABILITY_UNAVAILABLE")
	default:
		// Domain errors are controlled conflicts after exact authorization. Do
		// not expose object identity or SQL details in a transport response.
		writeProblem(writer, http.StatusConflict, "Workspace command rejected", "the command could not be applied to the current synthetic workspace", "AGA_WORKSPACE_COMMAND_REJECTED")
	}
}

func contextWithPrincipal(request *http.Request, principal identity.Principal) context.Context {
	return context.WithValue(request.Context(), principalContextKey{}, principal)
}

func decodeWorkspaceJSON(request *http.Request, destination any) error {
	if request.Body == nil {
		return workspace.ErrMalformedCommand
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("workspace body has trailing data")
	}
	return nil
}

func validateWorkspaceHeaders(request *http.Request, family workspace.OperationFamily, body workspace.CommandEnvelope) error {
	if strings.TrimSpace(request.Header.Get("Idempotency-Key")) == "" || request.Header.Get("Idempotency-Key") != body.IdempotencyKey {
		return workspace.ErrNeutralDenied
	}
	if family == workspace.FamilyClassificationQuery || family == workspace.FamilyLifecycleQuery {
		return workspace.ErrNeutralDenied
	}
	match := workspaceRevisionETag.FindStringSubmatch(request.Header.Get("If-Match"))
	if len(match) != 2 {
		return workspace.ErrNeutralDenied
	}
	revision, err := strconv.Atoi(match[1])
	if err != nil {
		return workspace.ErrNeutralDenied
	}
	switch family {
	case workspace.FamilyClassificationCommand:
		if revision != body.ExpectedDraftRevision {
			return workspace.ErrNeutralDenied
		}
	case workspace.FamilyRecommendationCommand:
		expected := body.ExpectedDraftRevision
		if body.OperationID == workspace.OperationCreateInspection {
			expected = body.ExpectedRecommendationRevision
		}
		if revision != expected {
			return workspace.ErrNeutralDenied
		}
	case workspace.FamilyLifecycleCommand:
		if revision != body.ExpectedLifecycleRevision || strings.TrimSpace(body.ExpectedLifecycleDigest) == "" {
			return workspace.ErrNeutralDenied
		}
	case workspace.FamilyAdminCommand:
		if body.OperationID != workspace.OperationResetGeneration || revision != body.ExpectedGenerationRevision || strings.TrimSpace(body.ExpectedGenerationID) == "" || strings.TrimSpace(body.ExpectedGenerationSealDigest) == "" {
			return workspace.ErrNeutralDenied
		}
	default:
		return workspace.ErrNeutralDenied
	}
	return nil
}

func queryBelongsToFamily(operation string, family workspace.OperationFamily) bool {
	if family == workspace.FamilyClassificationQuery {
		switch operation {
		case workspace.OperationGetSummary, workspace.OperationGetTaxonomy, workspace.OperationGetProviderConfiguration, workspace.OperationSearchItems, workspace.OperationGetDraft, workspace.OperationGetHistory:
			return true
		}
	}
	if family == workspace.FamilyLifecycleQuery {
		switch operation {
		case workspace.OperationGetInspection, workspace.OperationGetFinding, workspace.OperationGetCAPEvidence, workspace.OperationGetRoleHistory:
			return true
		}
	}
	return false
}
