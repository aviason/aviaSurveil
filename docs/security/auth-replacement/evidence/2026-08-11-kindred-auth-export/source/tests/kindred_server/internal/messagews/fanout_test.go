package messagews

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	apigwtypes "github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"

	"kindred_server/internal/message"
	"kindred_server/internal/messagefanout"
)

func TestRealtimeFanoutPostsToSubscribedConnections(t *testing.T) {
	repo := &fakeSubscriptionStore{subs: []Subscription{
		{ConnectionID: "conn-1", RequestID: "r1"},
		{ConnectionID: "conn-2", RequestID: "r1"},
	}}
	poster := &fakeConnectionPoster{}
	fanout := NewRealtimeFanout(repo, poster, nil)

	payload := messagefanout.MessageEvent{
		Type:            "message.created",
		ID:              "m1",
		RequestID:       "r1",
		SenderID:        "u1",
		ClientMessageID: "11111111-1111-4111-8111-111111111111",
		ProtocolVersion: 4,
		SessionID:       "session-1",
		RatchetHeader: message.RatchetHeader{
			DHPublicKey:         "dh",
			PreviousChainLength: 0,
			MessageNumber:       1,
			PQ:                  &message.PQRatchetHeader{Epoch: 1, Type: "ct1", MessageNumber: 0, Data: "pq"},
		},
		Nonce:            "nonce",
		Ciphertext:       "ct",
		CiphertextHash:   "cipher-hash",
		PlaintextHash:    "plain-hash",
		SenderSignature:  "sig",
		SenderKeyVersion: 1,
	}
	if err := fanout.PublishRealtime(context.Background(), payload); err != nil {
		t.Fatalf("PublishRealtime error = %v", err)
	}
	if len(poster.posts) != 2 {
		t.Fatalf("posts = %#v", poster.posts)
	}
	if poster.posts[0].connectionID != "conn-1" || poster.posts[1].connectionID != "conn-2" {
		t.Fatalf("posts = %#v", poster.posts)
	}
	var got messagefanout.MessageEvent
	if err := json.Unmarshal(poster.posts[0].data, &got); err != nil {
		t.Fatalf("message json: %v", err)
	}
	if !reflect.DeepEqual(got, payload) {
		t.Fatalf("payload = %#v", got)
	}
}

func TestRealtimeFanoutDeletesGoneConnections(t *testing.T) {
	repo := &fakeSubscriptionStore{subs: []Subscription{{ConnectionID: "conn-1", RequestID: "r1"}}}
	poster := &fakeConnectionPoster{err: &apigwtypes.GoneException{}}
	fanout := NewRealtimeFanout(repo, poster, nil)

	err := fanout.PublishRealtime(context.Background(), messagefanout.MessageEvent{Type: "message.created", RequestID: "r1"})
	if err != nil {
		t.Fatalf("PublishRealtime error = %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "conn-1" {
		t.Fatalf("deleted = %#v", repo.deleted)
	}
}

func TestRealtimeFanoutReturnsSubscriptionLoadError(t *testing.T) {
	want := errors.New("ddb down")
	fanout := NewRealtimeFanout(&fakeSubscriptionStore{err: want}, &fakeConnectionPoster{}, nil)

	if err := fanout.PublishRealtime(context.Background(), messagefanout.MessageEvent{RequestID: "r1"}); !errors.Is(err, want) {
		t.Fatalf("PublishRealtime error = %v, want %v", err, want)
	}
}

type fakeSubscriptionStore struct {
	subs    []Subscription
	deleted []string
	err     error
}

func (s *fakeSubscriptionStore) ListSubscriptionsByRequest(_ context.Context, _ string, _ int) ([]Subscription, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.subs, nil
}

func (s *fakeSubscriptionStore) DeleteConnection(_ context.Context, connectionID string) error {
	s.deleted = append(s.deleted, connectionID)
	return nil
}

type postedMessage struct {
	connectionID string
	data         []byte
}

type fakeConnectionPoster struct {
	posts []postedMessage
	err   error
}

func (p *fakeConnectionPoster) PostToConnection(_ context.Context, input *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
	p.posts = append(p.posts, postedMessage{connectionID: aws.ToString(input.ConnectionId), data: input.Data})
	if p.err != nil {
		return nil, p.err
	}
	return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
}
