package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aviason/aviaSurveil/internal/configuration"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/jackc/pgx/v5"
)

type CreateAdminQuestionCommand struct {
	OperationID         string
	IdempotencyKey      string
	ExpectedRevision    *int64
	Prompt              string
	ConfiguredReference string
	ExpectedEvidence    string
}

type CreateAdminTemplateDraftCommand struct {
	OperationID      string
	IdempotencyKey   string
	TemplateID       string
	ExpectedRevision int64
	ChangeReason     string
}

type AddAdminTemplateDraftQuestionCommand struct {
	OperationID      string
	IdempotencyKey   string
	TemplateID       string
	DraftVersionID   string
	QuestionID       string
	ExpectedRevision int64
}

type MoveAdminTemplateDraftQuestionCommand struct {
	OperationID      string
	IdempotencyKey   string
	TemplateID       string
	DraftVersionID   string
	QuestionID       string
	Direction        string
	ExpectedRevision int64
}

func (service *Service) CreateAdminQuestion(
	ctx context.Context,
	actor identity.Principal,
	command CreateAdminQuestionCommand,
) (configuration.AdminQuestion, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return configuration.AdminQuestion{}, fmt.Errorf(
			"%w: Admin Preview authority is required",
			ErrForbidden,
		)
	}
	prompt := strings.TrimSpace(command.Prompt)
	configuredReference := strings.TrimSpace(command.ConfiguredReference)
	expectedEvidence := strings.TrimSpace(command.ExpectedEvidence)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.ExpectedRevision != nil || prompt == "" ||
		len([]rune(prompt)) > 500 || configuredReference == "" || expectedEvidence == "" {
		return configuration.AdminQuestion{}, ErrInvalid
	}
	semantic := struct {
		IdempotencyKey      string `json:"idempotencyKey"`
		ExpectedRevision    *int64 `json:"expectedRevision"`
		Prompt              string `json:"prompt"`
		ConfiguredReference string `json:"configuredReference"`
		ExpectedEvidence    string `json:"expectedEvidence"`
	}{
		IdempotencyKey: command.IdempotencyKey, ExpectedRevision: command.ExpectedRevision,
		Prompt: prompt, ConfiguredReference: configuredReference,
		ExpectedEvidence: expectedEvidence,
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_admin_question",
		EntityID: "QUESTION_BANK", Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (
		transition[configuration.AdminQuestion],
		error,
	) {
		if _, err := transaction.Exec(ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			"admin-question-bank-allocation",
		); err != nil {
			return transition[configuration.AdminQuestion]{}, err
		}
		var questionCount int
		if err := transaction.QueryRow(ctx,
			"SELECT count(DISTINCT question_id) FROM question_versions",
		).Scan(&questionCount); err != nil {
			return transition[configuration.AdminQuestion]{}, err
		}
		questionID := fmt.Sprintf("Q-ADMIN-2026-%03d", questionCount+1)
		questionVersionID := "QV-" + questionID + "-V1"
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO question_versions (
				id, question_id, version, prompt, configured_reference,
				expected_evidence, created_by_subject_id, created_at
			) VALUES ($1, $2, 1, $3, $4, $5, $6, $7)
		`, questionVersionID, questionID, prompt, configuredReference,
			expectedEvidence, actor.SubjectID, now); err != nil {
			return transition[configuration.AdminQuestion]{}, err
		}
		question := configuration.AdminQuestion{
			ID: questionID, Prompt: prompt, ConfiguredReference: configuredReference,
			ExpectedEvidence: expectedEvidence, Revision: 1,
		}
		return transition[configuration.AdminQuestion]{
			Response: question, OrganizationID: actor.OrganizationID,
			Action: "admin.question_created", EntityType: "checklist_question",
			EntityID: questionID, EntityVersion: 1, AfterStatus: "DRAFT",
			Reason:   "Created Admin Preview Question Bank record.",
			SyncKind: "checklist_question", OutboxTopic: "admin.question_created",
		}, nil
	})
}

func (service *Service) CreateAdminTemplateDraft(
	ctx context.Context,
	actor identity.Principal,
	command CreateAdminTemplateDraftCommand,
) (configuration.AdminTemplateVersion, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return configuration.AdminTemplateVersion{}, fmt.Errorf(
			"%w: Admin Preview authority is required",
			ErrForbidden,
		)
	}
	changeReason := strings.TrimSpace(command.ChangeReason)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.TemplateID == "" || command.ExpectedRevision <= 0 || changeReason == "" {
		return configuration.AdminTemplateVersion{}, ErrInvalid
	}
	semantic := struct {
		IdempotencyKey   string `json:"idempotencyKey"`
		TemplateID       string `json:"templateId"`
		ExpectedRevision int64  `json:"expectedRevision"`
		ChangeReason     string `json:"changeReason"`
	}{
		command.IdempotencyKey, command.TemplateID,
		command.ExpectedRevision, changeReason,
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_admin_template_draft",
		EntityID: command.TemplateID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (
		transition[configuration.AdminTemplateVersion],
		error,
	) {
		var publishedVersionID string
		var masterRevision int64
		if err := transaction.QueryRow(ctx, `
			SELECT published_template_version_id, revision
			FROM template_masters
			WHERE id = $1 AND tombstoned_at IS NULL
			FOR UPDATE
		`, command.TemplateID).Scan(&publishedVersionID, &masterRevision); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[configuration.AdminTemplateVersion]{}, ErrNotFound
			}
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if masterRevision != command.ExpectedRevision {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		var activeDraft bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM template_draft_versions
				WHERE template_id = $1 AND status = 'DRAFT'
			)
		`, command.TemplateID).Scan(&activeDraft); err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if activeDraft {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		var publishedVersion int64
		var questionVersionIDs, questionIDs []string
		if err := transaction.QueryRow(ctx, `
			SELECT published.version::bigint,
				COALESCE(
					array_agg(link.question_version_id ORDER BY link.position)
						FILTER (WHERE link.question_version_id IS NOT NULL),
					ARRAY[]::text[]
				),
				COALESCE(
					array_agg(question.question_id ORDER BY link.position)
						FILTER (WHERE question.question_id IS NOT NULL),
					ARRAY[]::text[]
				)
			FROM checklist_template_versions published
			LEFT JOIN template_version_questions link
			  ON link.template_version_id = published.id
			LEFT JOIN question_versions question ON question.id = link.question_version_id
			WHERE published.id = $1
			GROUP BY published.id, published.version
		`, publishedVersionID).Scan(
			&publishedVersion, &questionVersionIDs, &questionIDs,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[configuration.AdminTemplateVersion]{}, ErrNotFound
			}
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		nextVersion := publishedVersion + 1
		draftID := strings.TrimSuffix(
			publishedVersionID,
			fmt.Sprintf("-%d", publishedVersion),
		) + fmt.Sprintf("-DRAFT-%d", nextVersion)
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO template_draft_versions (
				id, template_id, version, status, owner_role, creator_subject_id,
				change_reason, question_version_ids, revision, created_at, updated_at
			) VALUES ($1, $2, $3, 'DRAFT', 'Admin Preview', $4, $5, $6, 1, $7, $7)
		`, draftID, command.TemplateID, nextVersion, actor.SubjectID,
			changeReason, questionVersionIDs, now); err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		tag, err := transaction.Exec(ctx, `
			UPDATE template_masters
			SET revision = revision + 1, updated_at = $3
			WHERE id = $1 AND revision = $2
		`, command.TemplateID, masterRevision, now)
		if err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if tag.RowsAffected() != 1 {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		draft := configuration.AdminTemplateVersion{
			ID: draftID, TemplateID: command.TemplateID, Version: nextVersion,
			Status: "DRAFT", Owner: "Admin Preview", CreatorSubjectID: actor.SubjectID,
			ChangeReason: changeReason, QuestionIDs: questionIDs, Revision: 1,
			CreatedAt: now,
		}
		return transition[configuration.AdminTemplateVersion]{
			Response: draft, OrganizationID: actor.OrganizationID,
			Action: "admin.template_draft_created", EntityType: "checklist_template_version",
			EntityID: draft.ID, EntityVersion: draft.Revision,
			AfterStatus: "DRAFT", Reason: changeReason,
			SyncKind:    "checklist_template_version",
			OutboxTopic: "admin.template_draft_created",
		}, nil
	})
}

func (service *Service) AddAdminTemplateDraftQuestion(
	ctx context.Context,
	actor identity.Principal,
	command AddAdminTemplateDraftQuestionCommand,
) (configuration.AdminTemplateVersion, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return configuration.AdminTemplateVersion{}, fmt.Errorf(
			"%w: Admin Preview authority is required",
			ErrForbidden,
		)
	}
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.TemplateID == "" || command.DraftVersionID == "" ||
		command.QuestionID == "" || command.ExpectedRevision <= 0 {
		return configuration.AdminTemplateVersion{}, ErrInvalid
	}
	semantic := struct {
		IdempotencyKey   string `json:"idempotencyKey"`
		TemplateID       string `json:"templateId"`
		DraftVersionID   string `json:"draftVersionId"`
		QuestionID       string `json:"questionId"`
		ExpectedRevision int64  `json:"expectedRevision"`
	}{
		command.IdempotencyKey, command.TemplateID, command.DraftVersionID,
		command.QuestionID, command.ExpectedRevision,
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "add_admin_template_draft_question",
		EntityID: command.DraftVersionID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (
		transition[configuration.AdminTemplateVersion],
		error,
	) {
		draft, questionVersionIDs, err := getAdminTemplateDraftForUpdate(
			ctx, transaction, command.TemplateID, command.DraftVersionID,
		)
		if err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if draft.Revision != command.ExpectedRevision {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		var questionVersionID string
		if err := transaction.QueryRow(ctx, `
			SELECT id
			FROM question_versions
			WHERE question_id = $1
			ORDER BY version DESC
			LIMIT 1
		`, command.QuestionID).Scan(&questionVersionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[configuration.AdminTemplateVersion]{}, ErrNotFound
			}
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		var alreadyIncluded bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM question_versions
				WHERE id = ANY($1::text[]) AND question_id = $2
			)
		`, questionVersionIDs, command.QuestionID).Scan(&alreadyIncluded); err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if alreadyIncluded {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		questionVersionIDs = append(questionVersionIDs, questionVersionID)
		now := service.clock().UTC()
		tag, err := transaction.Exec(ctx, `
			UPDATE template_draft_versions
			SET question_version_ids = $3, revision = revision + 1, updated_at = $4
			WHERE id = $1 AND revision = $2
		`, draft.ID, draft.Revision, questionVersionIDs, now)
		if err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if tag.RowsAffected() != 1 {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		draft.QuestionIDs = append(draft.QuestionIDs, command.QuestionID)
		draft.Revision++
		return transition[configuration.AdminTemplateVersion]{
			Response: draft, OrganizationID: actor.OrganizationID,
			Action: "admin.template_question_added", EntityType: "checklist_template_version",
			EntityID: draft.ID, EntityVersion: draft.Revision,
			BeforeStatus: "DRAFT", AfterStatus: "DRAFT",
			Reason:      "Added exact Question " + command.QuestionID + ".",
			SyncKind:    "checklist_template_version",
			OutboxTopic: "admin.template_question_added",
		}, nil
	})
}

func (service *Service) MoveAdminTemplateDraftQuestion(
	ctx context.Context,
	actor identity.Principal,
	command MoveAdminTemplateDraftQuestionCommand,
) (configuration.AdminTemplateVersion, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return configuration.AdminTemplateVersion{}, fmt.Errorf(
			"%w: Admin Preview authority is required",
			ErrForbidden,
		)
	}
	direction := strings.ToUpper(strings.TrimSpace(command.Direction))
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.TemplateID == "" || command.DraftVersionID == "" ||
		command.QuestionID == "" || command.ExpectedRevision <= 0 ||
		(direction != "UP" && direction != "DOWN") {
		return configuration.AdminTemplateVersion{}, ErrInvalid
	}
	semantic := struct {
		IdempotencyKey   string `json:"idempotencyKey"`
		TemplateID       string `json:"templateId"`
		DraftVersionID   string `json:"draftVersionId"`
		QuestionID       string `json:"questionId"`
		Direction        string `json:"direction"`
		ExpectedRevision int64  `json:"expectedRevision"`
	}{
		command.IdempotencyKey, command.TemplateID, command.DraftVersionID,
		command.QuestionID, direction, command.ExpectedRevision,
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "move_admin_template_draft_question",
		EntityID: command.DraftVersionID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (
		transition[configuration.AdminTemplateVersion],
		error,
	) {
		draft, questionVersionIDs, err := getAdminTemplateDraftForUpdate(
			ctx, transaction, command.TemplateID, command.DraftVersionID,
		)
		if err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if draft.Revision != command.ExpectedRevision {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		index := slices.Index(draft.QuestionIDs, command.QuestionID)
		if index < 0 {
			return transition[configuration.AdminTemplateVersion]{}, ErrNotFound
		}
		target := index + 1
		if direction == "UP" {
			target = index - 1
		}
		if target < 0 || target >= len(questionVersionIDs) {
			return transition[configuration.AdminTemplateVersion]{}, ErrInvalid
		}
		questionVersionIDs[index], questionVersionIDs[target] =
			questionVersionIDs[target], questionVersionIDs[index]
		draft.QuestionIDs[index], draft.QuestionIDs[target] =
			draft.QuestionIDs[target], draft.QuestionIDs[index]
		now := service.clock().UTC()
		tag, err := transaction.Exec(ctx, `
			UPDATE template_draft_versions
			SET question_version_ids = $3, revision = revision + 1, updated_at = $4
			WHERE id = $1 AND revision = $2
		`, draft.ID, draft.Revision, questionVersionIDs, now)
		if err != nil {
			return transition[configuration.AdminTemplateVersion]{}, err
		}
		if tag.RowsAffected() != 1 {
			return transition[configuration.AdminTemplateVersion]{}, ErrConflict
		}
		draft.Revision++
		return transition[configuration.AdminTemplateVersion]{
			Response: draft, OrganizationID: actor.OrganizationID,
			Action:     "admin.template_question_reordered",
			EntityType: "checklist_template_version",
			EntityID:   draft.ID, EntityVersion: draft.Revision,
			BeforeStatus: "DRAFT", AfterStatus: "DRAFT",
			Reason:      "Moved exact Question " + command.QuestionID + " " + direction + ".",
			SyncKind:    "checklist_template_version",
			OutboxTopic: "admin.template_question_reordered",
		}, nil
	})
}

func getAdminTemplateDraftForUpdate(
	ctx context.Context,
	transaction pgx.Tx,
	templateID string,
	draftID string,
) (configuration.AdminTemplateVersion, []string, error) {
	var draft configuration.AdminTemplateVersion
	var questionVersionIDs []string
	if err := transaction.QueryRow(ctx, `
		SELECT id, template_id, version::bigint, status, owner_role,
			creator_subject_id, change_reason, question_version_ids,
			revision, created_at
		FROM template_draft_versions
		WHERE id = $1 AND template_id = $2 AND status = 'DRAFT'
		FOR UPDATE
	`, draftID, templateID).Scan(
		&draft.ID, &draft.TemplateID, &draft.Version, &draft.Status, &draft.Owner,
		&draft.CreatorSubjectID, &draft.ChangeReason, &questionVersionIDs,
		&draft.Revision, &draft.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return configuration.AdminTemplateVersion{}, nil, ErrNotFound
		}
		return configuration.AdminTemplateVersion{}, nil, err
	}
	rows, err := transaction.Query(ctx, `
		SELECT question.question_id
		FROM unnest($1::text[]) WITH ORDINALITY AS selected(id, position)
		JOIN question_versions question ON question.id = selected.id
		ORDER BY selected.position
	`, questionVersionIDs)
	if err != nil {
		return configuration.AdminTemplateVersion{}, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			return configuration.AdminTemplateVersion{}, nil, err
		}
		draft.QuestionIDs = append(draft.QuestionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		return configuration.AdminTemplateVersion{}, nil, err
	}
	if len(draft.QuestionIDs) != len(questionVersionIDs) {
		return configuration.AdminTemplateVersion{}, nil, ErrInvalid
	}
	return draft, questionVersionIDs, nil
}
