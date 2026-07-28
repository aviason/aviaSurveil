package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"
)

type MailpitInvitationEndpointConfig struct {
	Keycloak   *KeycloakEndpoint
	BaseURL    string
	HTTPClient *http.Client
}

type MailpitInvitationEndpoint struct {
	keycloak   *KeycloakEndpoint
	baseURL    *url.URL
	httpClient *http.Client
}

type scenarioMailpitMessages struct {
	Messages []scenarioMailpitMessage `json:"messages"`
}

type scenarioMailpitMessage struct {
	ID string                   `json:"ID"`
	To []scenarioMailpitAddress `json:"To"`
}

type scenarioMailpitAddress struct {
	Address string `json:"Address"`
}

func NewMailpitInvitationEndpoint(
	config MailpitInvitationEndpointConfig,
) (*MailpitInvitationEndpoint, error) {
	baseURL, err := url.Parse(strings.TrimRight(
		strings.TrimSpace(config.BaseURL),
		"/",
	))
	if err != nil ||
		(baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.Host == "" ||
		config.Keycloak == nil {
		return nil, fmt.Errorf(
			"valid Mailpit HTTP(S) URL and Keycloak endpoint are required",
		)
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &MailpitInvitationEndpoint{
		keycloak:   config.Keycloak,
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

func (endpoint *MailpitInvitationEndpoint) Preflight(
	ctx context.Context,
) error {
	messages, err := endpoint.messages(ctx)
	if err != nil {
		return err
	}
	if len(messages) != 0 {
		return fmt.Errorf(
			"connected-scenario Mailpit target retains %d messages",
			len(messages),
		)
	}
	return nil
}

func (endpoint *MailpitInvitationEndpoint) ResumePreflight(
	ctx context.Context,
) error {
	messages, err := endpoint.messages(ctx)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if strings.TrimSpace(message.ID) == "" || len(message.To) == 0 {
			return fmt.Errorf(
				"connected-scenario Mailpit target has an invalid resumable message",
			)
		}
		for _, recipient := range message.To {
			if !strings.HasSuffix(
				strings.ToLower(strings.TrimSpace(recipient.Address)),
				"@synthetic.invalid",
			) {
				return fmt.Errorf(
					"connected-scenario Mailpit target has a non-synthetic recipient",
				)
			}
		}
	}
	return nil
}

func (endpoint *MailpitInvitationEndpoint) EnsureInvitationDelivery(
	ctx context.Context,
	delivery InvitationDelivery,
) error {
	if err := validateInvitationDelivery(delivery); err != nil {
		return err
	}
	messages, err := endpoint.messages(ctx)
	if err != nil {
		return err
	}
	matches := recipientMatches(messages, delivery.Email)
	if matches == 1 {
		return nil
	}
	if matches > 1 {
		return fmt.Errorf(
			"Mailpit contains duplicate synthetic invitation recipient %s",
			delivery.Email,
		)
	}
	if err := endpoint.keycloak.issueActionsEmail(
		ctx,
		delivery.SubjectID,
		delivery.RequiredActions,
	); err != nil {
		return err
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		messages, err = endpoint.messages(ctx)
		if err == nil && recipientMatches(messages, delivery.Email) == 1 {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return fmt.Errorf(
				"Mailpit omitted synthetic invitation recipient %s",
				delivery.Email,
			)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (endpoint *MailpitInvitationEndpoint) ReconcileInvitationDeliveries(
	ctx context.Context,
	expected []InvitationDelivery,
) error {
	messages, err := endpoint.messages(ctx)
	if err != nil {
		return err
	}
	if len(messages) != len(expected) {
		return fmt.Errorf(
			"Mailpit message count = %d, expected %d",
			len(messages),
			len(expected),
		)
	}
	recipients := make([]string, 0, len(messages))
	for _, message := range messages {
		if strings.TrimSpace(message.ID) == "" || len(message.To) != 1 {
			return fmt.Errorf("Mailpit message has invalid recipient metadata")
		}
		recipients = append(
			recipients,
			strings.ToLower(strings.TrimSpace(message.To[0].Address)),
		)
	}
	sort.Strings(recipients)
	expectedRecipients := make([]string, len(expected))
	for index, delivery := range expected {
		if err := validateInvitationDelivery(delivery); err != nil {
			return err
		}
		expectedRecipients[index] = delivery.Email
	}
	sort.Strings(expectedRecipients)
	if !sameStrings(recipients, expectedRecipients) {
		return fmt.Errorf("Mailpit synthetic invitation recipients differ")
	}
	return nil
}

func (endpoint *MailpitInvitationEndpoint) messages(
	ctx context.Context,
) ([]scenarioMailpitMessage, error) {
	requestURL := *endpoint.baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") +
		"/api/v1/messages"
	query := requestURL.Query()
	query.Set("limit", "10000")
	requestURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := endpoint.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Mailpit messages status %d",
			response.StatusCode,
		)
	}
	var payload scenarioMailpitMessages
	if err := decodeScenarioJSON(response.Body, &payload); err != nil {
		return nil, err
	}
	return payload.Messages, nil
}

func validateInvitationDelivery(delivery InvitationDelivery) error {
	email := strings.ToLower(strings.TrimSpace(delivery.Email))
	address, err := mail.ParseAddress(email)
	if err != nil ||
		address.Address != email ||
		!strings.HasSuffix(email, "@synthetic.invalid") ||
		!validProviderSubjectID(delivery.SubjectID) ||
		!syntheticSubjectPattern.MatchString(delivery.InvitationID) ||
		!syntheticSubjectPattern.MatchString(delivery.DeliveryID) ||
		!sameStrings(
			delivery.RequiredActions,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		) {
		return fmt.Errorf("invalid synthetic invitation delivery")
	}
	return nil
}

func recipientMatches(
	messages []scenarioMailpitMessage,
	email string,
) int {
	email = strings.ToLower(strings.TrimSpace(email))
	var matches int
	for _, message := range messages {
		for _, recipient := range message.To {
			if strings.ToLower(strings.TrimSpace(recipient.Address)) == email {
				matches++
			}
		}
	}
	return matches
}
