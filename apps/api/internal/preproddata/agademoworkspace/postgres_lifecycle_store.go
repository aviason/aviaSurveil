package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
)

func (store *PostgresStore) AppendLifecycleEvent(ctx context.Context, event LifecycleEvent) (LifecycleEvent, error) {
	if store == nil || store.pool == nil || !store.command {
		return LifecycleEvent{}, fmt.Errorf("workspace lifecycle requires command store")
	}
	if event.LifecycleID == "" || event.EventID == "" || event.OperationID == "" || event.CommandKey == "" || event.EventType == "" || event.ActorSubjectID == "" || len(event.Payload) == 0 || event.Sequence < 1 || !validDigest(event.EventDigest) {
		return LifecycleEvent{}, ErrWorkspaceAppendOnly
	}
	var aggregate struct {
		GenerationID     string `json:"generationId"`
		RecommendationID string `json:"recommendationId"`
		Revision         int    `json:"revision"`
		State            string `json:"state"`
		Digest           string `json:"digest"`
	}
	if err := json.Unmarshal(event.Payload, &aggregate); err != nil || aggregate.GenerationID == "" || aggregate.RecommendationID == "" || aggregate.Revision != event.Sequence || aggregate.State == "" || !validDigest(aggregate.Digest) {
		return LifecycleEvent{}, ErrWorkspaceAppendOnly
	}
	existing, err := store.GetLifecycleEvents(ctx, aggregate.GenerationID, event.LifecycleID)
	if err != nil {
		return LifecycleEvent{}, err
	}
	lastSequence := 0
	previousDigest := ""
	if len(existing) > 0 {
		last := existing[len(existing)-1]
		lastSequence = last.Sequence
		previousDigest = last.EventDigest
	}
	if event.Sequence != lastSequence+1 || event.PreviousDigest != previousDigest {
		return LifecycleEvent{}, ErrWorkspaceCAS
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_LIFECYCLE", "lifecycleId": event.LifecycleID, "generationId": aggregate.GenerationID,
		"recommendationId": aggregate.RecommendationID, "revision": aggregate.Revision, "aggregateDigest": aggregate.Digest,
		"state": aggregate.State, "sequence": event.Sequence, "eventId": event.EventID, "operationId": event.OperationID,
		"commandKey": event.CommandKey, "eventType": event.EventType, "payload": event.Payload,
		"actorSubjectId": event.ActorSubjectID, "createdAt": event.CreatedAt, "previousDigest": event.PreviousDigest,
		"eventDigest": event.EventDigest,
	}); err != nil {
		return LifecycleEvent{}, err
	}
	return event, nil
}

func (store *PostgresStore) GetLifecycleEvents(ctx context.Context, generationID, lifecycleID string) ([]LifecycleEvent, error) {
	if store == nil || store.pool == nil {
		return nil, fmt.Errorf("workspace lifecycle requires store")
	}
	var events []LifecycleEvent
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_LIFECYCLE_EVENTS", "generationId": generationID, "lifecycleId": lifecycleID}, &events); err != nil {
		return nil, err
	}
	return events, nil
}
