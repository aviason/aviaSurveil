package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"kindred_server/internal/auth"
)

type TokenVerifier interface {
	VerifyToken(context.Context, string) (auth.Claims, error)
}

type LocalHTTPHandler struct {
	handler  *Handler
	verifier TokenVerifier
}

type localGraphQLRequest struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
}

type localGraphQLResponse struct {
	Data   map[string]any      `json:"data,omitempty"`
	Errors []localGraphQLError `json:"errors,omitempty"`
}

type localGraphQLError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

func NewLocalHTTPHandler(handler *Handler, verifier TokenVerifier) http.Handler {
	return &LocalHTTPHandler{handler: handler, verifier: verifier}
}

func (h *LocalHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeLocalGraphQLErrors(w, http.StatusMethodNotAllowed, NewAppSyncError(http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed"))
		return
	}
	if h.handler == nil {
		writeLocalGraphQLErrors(w, http.StatusInternalServerError, NewAppSyncError(http.StatusInternalServerError, "internal_error", "GraphQL handler is not configured"))
		return
	}

	var req localGraphQLRequest
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		writeLocalGraphQLErrors(w, http.StatusBadRequest, NewAppSyncError(http.StatusBadRequest, "bad_request", "invalid GraphQL request body"))
		return
	}

	event, fieldName, appErr := h.resolverEvent(r, req)
	if appErr != nil {
		writeLocalGraphQLErrors(w, http.StatusOK, appErr)
		return
	}

	payload, err := h.handler.Handle(r.Context(), event)
	if err != nil {
		writeLocalGraphQLErrors(w, http.StatusOK, appSyncErrorForGraphQLResponse(err))
		return
	}

	writeLocalGraphQLResponse(w, http.StatusOK, localGraphQLResponse{
		Data: map[string]any{fieldName: payload},
	})
}

func (h *LocalHTTPHandler) resolverEvent(r *http.Request, req localGraphQLRequest) (ResolverEvent, string, *AppSyncError) {
	operation, field, appErr := selectLocalGraphQLOperation(req)
	if appErr != nil {
		return ResolverEvent{}, "", appErr
	}

	parentType, appErr := localGraphQLParentType(operation)
	if appErr != nil {
		return ResolverEvent{}, "", appErr
	}

	variables, appErr := variablesWithDefaults(operation, req.Variables)
	if appErr != nil {
		return ResolverEvent{}, "", appErr
	}

	arguments, appErr := localGraphQLArguments(field, variables)
	if appErr != nil {
		return ResolverEvent{}, "", appErr
	}

	identity, appErr := h.identityFromRequest(r)
	if appErr != nil {
		return ResolverEvent{}, "", appErr
	}

	event := ResolverEvent{
		Arguments: arguments,
		Identity:  identity,
		Request: ResolverRequest{
			Headers: requestHeaderMap(r),
		},
		Info: ResolverInfo{
			FieldName:        field.Name,
			ParentTypeName:   parentType,
			SelectionSetList: selectionSetList(field.SelectionSet),
			Variables:        variables,
		},
		TypeName:  parentType,
		FieldName: field.Name,
	}

	return event, field.Name, nil
}

func selectLocalGraphQLOperation(req localGraphQLRequest) (*ast.OperationDefinition, *ast.Field, *AppSyncError) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "GraphQL query is required")
	}
	doc, err := parser.ParseQuery(&ast.Source{Name: "local-http-graphql", Input: req.Query})
	if err != nil {
		return nil, nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "malformed GraphQL query")
	}

	operation, appErr := selectOperation(doc, strings.TrimSpace(req.OperationName))
	if appErr != nil {
		return nil, nil, appErr
	}
	field, appErr := selectRootField(operation)
	if appErr != nil {
		return nil, nil, appErr
	}
	return operation, field, nil
}

func selectOperation(doc *ast.QueryDocument, operationName string) (*ast.OperationDefinition, *AppSyncError) {
	if doc == nil || len(doc.Operations) == 0 {
		return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "GraphQL operation is required")
	}
	if operationName != "" {
		for _, operation := range doc.Operations {
			if operation.Name == operationName {
				return operation, nil
			}
		}
		return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "GraphQL operationName was not found")
	}
	if len(doc.Operations) != 1 {
		return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "GraphQL operationName is required when multiple operations are present")
	}
	return doc.Operations[0], nil
}

func selectRootField(operation *ast.OperationDefinition) (*ast.Field, *AppSyncError) {
	var fields []*ast.Field
	for _, selection := range operation.SelectionSet {
		field, ok := selection.(*ast.Field)
		if !ok {
			return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "local GraphQL requires a concrete root field")
		}
		if field.Name == "__typename" {
			continue
		}
		fields = append(fields, field)
	}
	if len(fields) != 1 {
		return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", "local GraphQL requires exactly one root field")
	}
	return fields[0], nil
}

func localGraphQLParentType(operation *ast.OperationDefinition) (string, *AppSyncError) {
	switch operation.Operation {
	case ast.Query:
		return "Query", nil
	case ast.Mutation:
		return "Mutation", nil
	case ast.Subscription:
		return "", NewAppSyncError(http.StatusBadRequest, "unsupported_operation", "local HTTP GraphQL does not support subscriptions")
	default:
		return "", NewAppSyncError(http.StatusBadRequest, "bad_request", "unsupported GraphQL operation")
	}
}

func variablesWithDefaults(operation *ast.OperationDefinition, in map[string]any) (map[string]any, *AppSyncError) {
	variables := make(map[string]any, len(in))
	for name, value := range in {
		variables[name] = value
	}
	for _, definition := range operation.VariableDefinitions {
		if _, ok := variables[definition.Variable]; ok || definition.DefaultValue == nil {
			continue
		}
		value, err := definition.DefaultValue.Value(variables)
		if err != nil {
			return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid default value for GraphQL variable %s", definition.Variable))
		}
		variables[definition.Variable] = value
	}
	return variables, nil
}

func localGraphQLArguments(field *ast.Field, variables map[string]any) (map[string]json.RawMessage, *AppSyncError) {
	args := make(map[string]json.RawMessage, len(field.Arguments))
	for _, arg := range field.Arguments {
		value, err := arg.Value.Value(variables)
		if err != nil {
			return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid GraphQL argument %s", arg.Name))
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, NewAppSyncError(http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid GraphQL argument %s", arg.Name))
		}
		args[arg.Name] = raw
	}
	return args, nil
}

func (h *LocalHTTPHandler) identityFromRequest(r *http.Request) (*AppSyncIdentity, *AppSyncError) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return nil, nil
	}
	if h.verifier == nil {
		return nil, NewAppSyncError(http.StatusUnauthorized, "unauthorized", "invalid or missing token")
	}
	claims, err := h.verifier.VerifyToken(r.Context(), token)
	if err != nil {
		return nil, appSyncErrorForGraphQLResponse(err)
	}
	return identityFromClaims(claims, sourceIPs(r)), nil
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func identityFromClaims(claims auth.Claims, sourceIP []string) *AppSyncIdentity {
	claimMap := map[string]any{
		"sub":   claims.UserID,
		"email": claims.Email,
		"tv":    claims.TokenVersion,
	}
	if claims.DeviceID != "" {
		claimMap["did"] = claims.DeviceID
	}
	return &AppSyncIdentity{
		Sub:                 claims.UserID,
		Username:            claims.UserID,
		Claims:              claimMap,
		SourceIP:            sourceIP,
		DefaultAuthStrategy: "ALLOW",
	}
}

func requestHeaderMap(r *http.Request) map[string]string {
	headers := make(map[string]string, len(r.Header))
	for name, values := range r.Header {
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}
	return headers
}

func sourceIPs(r *http.Request) []string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		return []string{xff}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && strings.TrimSpace(host) != "" {
		return []string{host}
	}
	if trimmed := strings.TrimSpace(r.RemoteAddr); trimmed != "" {
		return []string{trimmed}
	}
	return nil
}

func selectionSetList(selectionSet ast.SelectionSet) []string {
	fields := make([]string, 0, len(selectionSet))
	for _, selection := range selectionSet {
		field, ok := selection.(*ast.Field)
		if !ok || field.Name == "__typename" {
			continue
		}
		if field.Alias != "" {
			fields = append(fields, field.Alias)
			continue
		}
		fields = append(fields, field.Name)
	}
	return fields
}

func appSyncErrorForGraphQLResponse(err error) *AppSyncError {
	var appErr *AppSyncError
	if errors.As(err, &appErr) {
		return appErr
	}
	return AppSyncErrorFromError(err)
}

func writeLocalGraphQLErrors(w http.ResponseWriter, status int, errs ...*AppSyncError) {
	response := localGraphQLResponse{Errors: make([]localGraphQLError, 0, len(errs))}
	for _, err := range errs {
		response.Errors = append(response.Errors, localGraphQLError{
			Message:    err.ErrorMessage,
			Extensions: appSyncErrorExtensions(err),
		})
	}
	writeLocalGraphQLResponse(w, status, response)
}

func writeLocalGraphQLResponse(w http.ResponseWriter, status int, response localGraphQLResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func appSyncErrorExtensions(err *AppSyncError) map[string]any {
	code := err.ErrorType
	status := http.StatusInternalServerError
	if err.ErrorInfo != nil {
		if err.ErrorInfo.Code != "" {
			code = err.ErrorInfo.Code
		}
		if err.ErrorInfo.Status != 0 {
			status = err.ErrorInfo.Status
		}
	}
	return map[string]any{
		"code":   code,
		"status": status,
	}
}
