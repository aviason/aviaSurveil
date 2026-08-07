package checklistgovernance

import "testing"

func TestExerciseQuestionReviewCannotPublish(t *testing.T) {
	review, err := NewQuestionReview(QuestionReviewModeExercise)
	if err != nil {
		t.Fatal(err)
	}
	if err := review.Execute(QuestionReviewCommand{Action: QuestionReviewActionPublish}); err == nil {
		t.Fatal("RED successor governance contract: exercise review must not invoke publication")
	}
}

func TestGovernedQuestionReviewUsesSeparateTechnicalAndPublicationCommands(t *testing.T) {
	review, err := NewQuestionReview(QuestionReviewModeGoverned)
	if err != nil {
		t.Fatal(err)
	}
	if err := review.Execute(QuestionReviewCommand{Action: QuestionReviewActionTechnicalApprove}); err != nil {
		t.Fatal(err)
	}
	if err := review.Execute(QuestionReviewCommand{Action: QuestionReviewActionPublish}); err != nil {
		t.Fatal(err)
	}
}
