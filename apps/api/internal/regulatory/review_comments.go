package regulatory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrReviewCommentIdempotencyConflict = errors.New("review comment idempotency conflict")

type ReviewComment struct {
	CommentID   string    `json:"commentId"`
	CandidateID string    `json:"candidateId"`
	Visibility  string    `json:"visibility"`
	Body        string    `json:"body"`
	AuthorID    string    `json:"authorId"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ReviewCommentLedger struct {
	mu       sync.RWMutex
	comments []ReviewComment
	byKey    map[string]ReviewComment
}

func NewReviewCommentLedger() *ReviewCommentLedger {
	return &ReviewCommentLedger{byKey: make(map[string]ReviewComment)}
}

func (ledger *ReviewCommentLedger) Append(candidateID, authorID, visibility, body, operationID, idempotencyKey string, at time.Time) (ReviewComment, error) {
	if ledger == nil || strings.TrimSpace(candidateID) == "" || strings.TrimSpace(authorID) == "" || strings.TrimSpace(body) == "" || strings.TrimSpace(operationID) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return ReviewComment{}, errors.New("review comment is incomplete")
	}
	if visibility != "INTERNAL_CAA" && visibility != "COMMENT_TO_AUDITEE" {
		return ReviewComment{}, errors.New("unsupported review comment visibility")
	}
	key := operationID + "\x00" + idempotencyKey
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.byKey[key]; ok {
		if existing.CandidateID != candidateID || existing.Body != body || existing.Visibility != visibility {
			return ReviewComment{}, ErrReviewCommentIdempotencyConflict
		}
		return existing, nil
	}
	sum := sha256.Sum256([]byte(candidateID + ":" + authorID + ":" + visibility + ":" + body + ":" + operationID))
	comment := ReviewComment{CommentID: "comment-" + hex.EncodeToString(sum[:8]), CandidateID: candidateID, Visibility: visibility, Body: body, AuthorID: authorID, CreatedAt: at.UTC()}
	ledger.comments = append(ledger.comments, comment)
	ledger.byKey[key] = comment
	return comment, nil
}

func (ledger *ReviewCommentLedger) List(candidateID string) []ReviewComment {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()
	output := make([]ReviewComment, 0)
	for _, comment := range ledger.comments {
		if comment.CandidateID == candidateID {
			output = append(output, comment)
		}
	}
	return output
}
