package httpapi

import (
	"net/http"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
)

func (api *CanonicalAPI) startInspection(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.StartInspectionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	auditID := decodedCanonicalPathParam(request, "auditId")
	if auditID == "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	result, err := api.application.StartInspection(request.Context(), actor, application.StartInspectionCommand{
		OperationID:  input.OperationId,
		InspectionID: auditID, ExpectedInspectionRevision: input.ExpectedInspectionRevision,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, generated.StartInspectionOutput{
		InspectionId: result.InspectionID, AssignmentId: result.AssignmentID,
		InspectionStatus: result.InspectionStatus, AssignmentStatus: string(result.AssignmentStatus),
		InspectionRevision: result.InspectionRevision, ChecklistRevision: result.ChecklistRevision,
		StartedAt: result.StartedAt.UTC().Format(time.RFC3339Nano),
	}, nil)
}
