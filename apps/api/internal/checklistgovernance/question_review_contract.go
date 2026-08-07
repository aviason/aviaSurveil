package checklistgovernance

import (
	"errors"
	"fmt"
)

// QuestionReviewMode is a server-selected authority boundary.  The retained
// queue/dossier UI may render both modes, but exercise commands can never
// reach governed technical approval or publication aggregates.
type QuestionReviewMode string

const (
	QuestionReviewModeGoverned QuestionReviewMode = "GOVERNED_OPERATIONAL"
	QuestionReviewModeExercise QuestionReviewMode = "PREPROD_EXERCISE"
)

// QuestionReviewAction is intentionally a finite command vocabulary.  The
// technical-approval and publication actions are separate command boundaries
// even though this compact contract lets the RED tests exercise the mode
// guard without a database.
type QuestionReviewAction string

const (
	QuestionReviewActionRetain             QuestionReviewAction = "RETAIN"
	QuestionReviewActionInclude            QuestionReviewAction = "INCLUDE"
	QuestionReviewActionExclude            QuestionReviewAction = "EXCLUDE"
	QuestionReviewActionDefer              QuestionReviewAction = "DEFER"
	QuestionReviewActionDomainReclassified QuestionReviewAction = "DOMAIN_RECLASSIFIED"
	QuestionReviewActionTopicReclassified  QuestionReviewAction = "TOPIC_RECLASSIFIED"
	QuestionReviewActionTechnicalApprove   QuestionReviewAction = "TECHNICAL_APPROVE"
	QuestionReviewActionPublish            QuestionReviewAction = "PUBLISH"
)

type QuestionReviewCommand struct {
	Action QuestionReviewAction
}

var (
	ErrInvalidQuestionReviewMode = errors.New("invalid question review mode")
	ErrExercisePublication       = errors.New("preprod exercise review cannot invoke technical approval or publication")
	ErrTechnicalApprovalRequired = errors.New("technical approval is required before publication")
	ErrUnsupportedReviewAction   = errors.New("unsupported question review action")
)

// QuestionReview is the authority-aware command boundary behind the
// canonical Find -> Compare -> Decide surface.  Durable implementations
// should persist the same state through append-only review events.
type QuestionReview struct {
	mode              QuestionReviewMode
	technicalApproved bool
}

func NewQuestionReview(mode QuestionReviewMode) (*QuestionReview, error) {
	if mode != QuestionReviewModeGoverned && mode != QuestionReviewModeExercise {
		return nil, fmt.Errorf("%w: %q", ErrInvalidQuestionReviewMode, mode)
	}
	return &QuestionReview{mode: mode}, nil
}

func (review *QuestionReview) Mode() QuestionReviewMode {
	if review == nil {
		return ""
	}
	return review.mode
}

// Execute is a compact command dispatcher used by the contract tests.  The
// production HTTP handlers should call ExecuteExerciseDisposition,
// ExecuteTechnicalApproval, or ExecutePublication explicitly rather than
// accepting an arbitrary client action.
func (review *QuestionReview) Execute(command QuestionReviewCommand) error {
	if review == nil {
		return ErrInvalidQuestionReviewMode
	}
	switch command.Action {
	case QuestionReviewActionRetain, QuestionReviewActionInclude,
		QuestionReviewActionExclude, QuestionReviewActionDefer,
		QuestionReviewActionDomainReclassified, QuestionReviewActionTopicReclassified:
		return nil
	case QuestionReviewActionTechnicalApprove:
		return review.ExecuteTechnicalApproval()
	case QuestionReviewActionPublish:
		return review.ExecutePublication()
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedReviewAction, command.Action)
	}
}

func (review *QuestionReview) ExecuteExerciseDisposition(action QuestionReviewAction) error {
	if review == nil || review.mode != QuestionReviewModeExercise {
		return ErrInvalidQuestionReviewMode
	}
	switch action {
	case QuestionReviewActionRetain, QuestionReviewActionInclude,
		QuestionReviewActionExclude, QuestionReviewActionDefer,
		QuestionReviewActionDomainReclassified, QuestionReviewActionTopicReclassified:
		return nil
	default:
		return ErrExercisePublication
	}
}

func (review *QuestionReview) ExecuteTechnicalApproval() error {
	if review == nil || review.mode != QuestionReviewModeGoverned {
		return ErrExercisePublication
	}
	review.technicalApproved = true
	return nil
}

func (review *QuestionReview) ExecutePublication() error {
	if review == nil || review.mode != QuestionReviewModeGoverned {
		return ErrExercisePublication
	}
	if !review.technicalApproved {
		return ErrTechnicalApprovalRequired
	}
	return nil
}

func (review *QuestionReview) CanTechnicalApprove() bool {
	return review != nil && review.mode == QuestionReviewModeGoverned
}

func (review *QuestionReview) CanPublish() bool {
	return review != nil && review.mode == QuestionReviewModeGoverned && review.technicalApproved
}
