package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/communications"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/notifications"
	"github.com/go-chi/chi/v5"
)

func (api *CanonicalAPI) listCommunications(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	organizationID := ""
	if value := optionalQuery(request, "organizationId"); value != nil {
		organizationID = *value
	}
	items, err := api.communications.ListCommunications(
		request.Context(),
		actor,
		organizationID,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := generated.ListCommunicationsOutput{
		Items: make([]generated.CommunicationView, 0, len(items)),
	}
	for _, item := range items {
		output.Items = append(output.Items, communicationView(item))
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) sendCommunication(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SendCommunicationInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.ExpectedRevision != nil {
		api.respond(
			writer,
			nil,
			fmt.Errorf("%w: a new immutable Communication requires null expectedRevision", application.ErrInvalid),
		)
		return
	}
	if !validOptionalRevisionCommandHeaders(
		request,
		input.IdempotencyKey,
		input.ExpectedRevision,
	) {
		api.respond(
			writer,
			nil,
			fmt.Errorf("%w: Communication command headers are invalid", application.ErrInvalid),
		)
		return
	}
	organizationID := ""
	if input.OrganizationId != nil {
		organizationID = *input.OrganizationId
	}
	message, err := api.communications.SendCommunication(
		request.Context(),
		actor,
		application.SendCommunicationCommand{
			OperationID:    input.OperationId,
			CorrelationID:  input.OperationId,
			IdempotencyKey: input.IdempotencyKey,
			OrganizationID: organizationID,
			Subject:        input.Subject,
			Body:           input.Body,
			Audience:       communications.Audience(input.Audience),
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(message.Revision))
	writeJSON(writer, http.StatusCreated, communicationView(message))
}

func communicationView(
	message communications.Message,
) generated.CommunicationView {
	var organizationID *string
	if message.OrganizationID != "" {
		value := message.OrganizationID
		organizationID = &value
	}
	return generated.CommunicationView{
		Id:             message.ID,
		OrganizationId: organizationID,
		Subject:        message.Subject,
		Body:           message.Body,
		Audience:       string(message.Audience),
		Direction:      string(message.Direction),
		Revision:       message.Revision,
		CreatedAt:      message.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (api *CanonicalAPI) listCalendarItems(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	organizationID := ""
	if value := optionalQuery(request, "organizationId"); value != nil {
		organizationID = *value
	}
	items, err := api.communications.ListCalendarItems(
		request.Context(),
		actor,
		organizationID,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := generated.ListCalendarItemsOutput{
		Items: make([]generated.CalendarItemView, 0, len(items)),
	}
	for _, item := range items {
		output.Items = append(output.Items, calendarItemView(item))
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) getCalendarItem(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	item, err := api.communications.GetCalendarItem(
		request.Context(),
		actor,
		chi.URLParam(request, "calendarItemId"),
	)
	api.respond(writer, calendarItemView(item), err)
}

func calendarItemView(
	item communications.CalendarItem,
) generated.CalendarItemView {
	organizationName := item.OrganizationName
	nextAction := item.NextAction
	return generated.CalendarItemView{
		Id:               item.ID,
		AuditId:          item.AuditID,
		OrganizationId:   item.OrganizationID,
		OrganizationName: &organizationName,
		Title:            item.Title,
		NextAction:       &nextAction,
		ScheduledDate:    item.ScheduledDate,
		DueState:         generated.DueState(item.DueState),
	}
}

func (api *CanonicalAPI) listNotifications(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	page, err := api.communications.ListNotifications(
		request.Context(),
		actor,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := generated.ListNotificationsOutput{
		Items: make([]generated.NotificationView, 0, len(page.Items)),
	}
	for _, item := range page.Items {
		output.Items = append(output.Items, notificationView(item))
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) markNotificationRead(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.MarkNotificationReadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	notificationID := chi.URLParam(request, "notificationId")
	if notificationID != input.NotificationId {
		api.respond(
			writer,
			nil,
			fmt.Errorf("%w: Notification path and body must match", application.ErrInvalid),
		)
		return
	}
	if input.ExpectedRevision == nil {
		api.respond(
			writer,
			nil,
			fmt.Errorf("%w: Notification expectedRevision is required", application.ErrInvalid),
		)
		return
	}
	if !validRevisionCommandHeaders(
		request,
		input.IdempotencyKey,
		input.ExpectedRevision,
	) {
		api.respond(
			writer,
			nil,
			fmt.Errorf("%w: Notification command headers are invalid", application.ErrInvalid),
		)
		return
	}
	item, err := api.communications.MarkNotificationRead(
		request.Context(),
		actor,
		application.MarkNotificationReadCommand{
			OperationID:      input.OperationId,
			CorrelationID:    input.OperationId,
			IdempotencyKey:   input.IdempotencyKey,
			NotificationID:   notificationID,
			ExpectedRevision: *input.ExpectedRevision,
		},
	)
	if err == nil {
		writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	}
	api.respond(writer, notificationView(item), err)
}

func notificationView(
	item notifications.Notification,
) generated.NotificationView {
	var readAt *string
	if item.ReadAt != nil {
		value := item.ReadAt.UTC().Format(time.RFC3339Nano)
		readAt = &value
	}
	var emailAcceptedAt *string
	if item.EmailAcceptedAt != nil {
		value := item.EmailAcceptedAt.UTC().Format(time.RFC3339Nano)
		emailAcceptedAt = &value
	}
	var emailNextAttemptAt *string
	if item.EmailNextAttemptAt != nil {
		value := item.EmailNextAttemptAt.UTC().Format(time.RFC3339Nano)
		emailNextAttemptAt = &value
	}
	return generated.NotificationView{
		Id:                    item.ID,
		SubjectId:             item.RecipientSubjectID,
		Title:                 item.Title,
		Body:                  item.Body,
		ReadAt:                readAt,
		EmailDeliveryStatus:   string(item.EmailDeliveryStatus),
		EmailDeliveryAttempts: int64(item.EmailDeliveryAttempts),
		EmailAcceptedAt:       emailAcceptedAt,
		EmailNextAttemptAt:    emailNextAttemptAt,
		Revision:              item.Revision,
	}
}
