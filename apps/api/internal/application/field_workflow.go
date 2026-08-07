package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklists"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings"
	"github.com/jackc/pgx/v5"
)

type UpsertChecklistResponseCommand struct {
	OperationID              string
	CorrelationID            string
	ResponseID               string
	InspectionID             string
	PackageID                string
	QuestionID               string
	ExpectedResponseRevision *int64
	Answer                   string
	CommentToAuditee         string
	InternalCAANote          string
}

type ChecklistResponseResult struct {
	ResponseID   string `json:"responseId"`
	InspectionID string `json:"inspectionId"`
	QuestionID   string `json:"questionId"`
	Answer       string `json:"answer"`
	Revision     int64  `json:"revision"`
}

func (service *Service) UpsertChecklistResponse(ctx context.Context, actor identity.Principal, command UpsertChecklistResponseCommand) (ChecklistResponseResult, error) {
	semantic := struct {
		InspectionID             string `json:"inspectionId"`
		PackageID                string `json:"packageId"`
		QuestionID               string `json:"questionId"`
		ExpectedResponseRevision *int64 `json:"expectedResponseRevision"`
		Answer                   string `json:"answer"`
		CommentToAuditee         string `json:"commentToAuditee"`
		InternalCAANote          string `json:"internalCaaNote"`
	}{
		InspectionID: command.InspectionID, PackageID: command.PackageID, QuestionID: command.QuestionID,
		ExpectedResponseRevision: command.ExpectedResponseRevision, Answer: strings.TrimSpace(command.Answer),
		CommentToAuditee: strings.TrimSpace(command.CommentToAuditee), InternalCAANote: strings.TrimSpace(command.InternalCAANote),
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "upsert_checklist_response", EntityID: command.ResponseID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[ChecklistResponseResult], error) {
		if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" {
			return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: Inspector role required", ErrForbidden)
		}
		if command.ResponseID == "" || command.InspectionID == "" || command.PackageID == "" || command.QuestionID == "" || !validChecklistAnswer(semantic.Answer) {
			return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: valid response, Audit, package, question, and answer are required", ErrInvalid)
		}

		var organizationID, checklistStatus string
		var packageExpiry *time.Time
		var packageRevoked *time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT inspection.organization_id, checklist.status, package.expires_at, package.revoked_at
			FROM inspections inspection
			JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
			JOIN inspection_packages package ON package.inspection_id = inspection.id AND package.id = $2
			JOIN inspection_question_assignments assignment
			  ON assignment.inspection_id = inspection.id AND assignment.question_id = $3 AND assignment.subject_id = $4
			WHERE inspection.id = $1
			FOR UPDATE OF checklist, package
		`, command.InspectionID, command.PackageID, command.QuestionID, actor.SubjectID).Scan(
			&organizationID, &checklistStatus, &packageExpiry, &packageRevoked,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: Inspector is not assigned to this Audit question", ErrForbidden)
			}
			return transition[ChecklistResponseResult]{}, err
		}
		if !checklists.CanEdit(checklists.Status(checklistStatus)) {
			return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: submitted checklist is read-only", ErrConflict)
		}
		now := service.clock().UTC()
		if packageRevoked != nil || (packageExpiry != nil && !now.Before(*packageExpiry)) {
			return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: inspection package is expired or withdrawn", ErrConflict)
		}

		var existingID, beforeAnswer string
		var existingRevision int64
		err := transaction.QueryRow(ctx, `
			SELECT id, response_value, revision FROM checklist_responses
			WHERE inspection_id = $1 AND question_id = $2 FOR UPDATE
		`, command.InspectionID, command.QuestionID).Scan(&existingID, &beforeAnswer, &existingRevision)
		var nextRevision int64
		if errors.Is(err, pgx.ErrNoRows) {
			if command.ExpectedResponseRevision != nil {
				return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: response does not exist at expected revision", ErrConflict)
			}
			nextRevision = 1
			if _, err := transaction.Exec(ctx, `
				INSERT INTO checklist_responses (
					id, inspection_id, package_id, question_id, assigned_inspector_subject_id,
					response_value, comment_to_auditee, internal_caa_note, revision, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), 1, $9)
			`, command.ResponseID, command.InspectionID, command.PackageID, command.QuestionID, actor.SubjectID,
				semantic.Answer, semantic.CommentToAuditee, semantic.InternalCAANote, now); err != nil {
				return transition[ChecklistResponseResult]{}, fmt.Errorf("create checklist response: %w", err)
			}
		} else if err != nil {
			return transition[ChecklistResponseResult]{}, err
		} else {
			if existingID != command.ResponseID || command.ExpectedResponseRevision == nil || existingRevision != *command.ExpectedResponseRevision {
				return transition[ChecklistResponseResult]{}, fmt.Errorf("%w: stale or mismatched checklist response", ErrConflict)
			}
			nextRevision = existingRevision + 1
			if _, err := transaction.Exec(ctx, `
				UPDATE checklist_responses
				SET response_value = $2, comment_to_auditee = NULLIF($3, ''), internal_caa_note = NULLIF($4, ''),
				    revision = $5, updated_at = $6
				WHERE id = $1
			`, existingID, semantic.Answer, semantic.CommentToAuditee, semantic.InternalCAANote, nextRevision, now); err != nil {
				return transition[ChecklistResponseResult]{}, fmt.Errorf("update checklist response: %w", err)
			}
		}

		response := ChecklistResponseResult{
			ResponseID: command.ResponseID, InspectionID: command.InspectionID,
			QuestionID: command.QuestionID, Answer: semantic.Answer, Revision: nextRevision,
		}
		return transition[ChecklistResponseResult]{
			Response: response, OrganizationID: organizationID, Action: "checklist_response.recorded",
			EntityType: "checklist_response", EntityID: command.ResponseID, EntityVersion: nextRevision,
			BeforeStatus: beforeAnswer, AfterStatus: semantic.Answer,
			SyncKind: "checklist_response", OutboxTopic: "checklist_response.recorded",
		}, nil
	})
}

type SubmitChecklistCommand struct {
	OperationID               string
	CorrelationID             string
	InspectionID              string
	ExpectedChecklistRevision int64
}

type ChecklistTransitionResult struct {
	InspectionID string            `json:"inspectionId"`
	Status       checklists.Status `json:"status"`
	Revision     int64             `json:"revision"`
}

func (service *Service) SubmitChecklist(ctx context.Context, actor identity.Principal, command SubmitChecklistCommand) (ChecklistTransitionResult, error) {
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "submit_checklist", EntityID: command.InspectionID,
		Semantic: struct {
			ExpectedRevision int64 `json:"expectedRevision"`
		}{command.ExpectedChecklistRevision},
	}, func(ctx context.Context, transaction pgx.Tx) (transition[ChecklistTransitionResult], error) {
		if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" {
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("%w: Inspector role required", ErrForbidden)
		}
		var status, organizationID string
		var revision int64
		if err := transaction.QueryRow(ctx, `
			SELECT checklist.status, checklist.revision, inspection.organization_id
			FROM inspection_checklists checklist
			JOIN inspections inspection ON inspection.id = checklist.inspection_id
			WHERE checklist.inspection_id = $1 FOR UPDATE OF checklist
		`, command.InspectionID).Scan(&status, &revision, &organizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[ChecklistTransitionResult]{}, ErrNotFound
			}
			return transition[ChecklistTransitionResult]{}, err
		}
		rows, err := transaction.Query(ctx, `
			SELECT DISTINCT subject_id FROM inspection_question_assignments WHERE inspection_id = $1 ORDER BY subject_id
		`, command.InspectionID)
		if err != nil {
			return transition[ChecklistTransitionResult]{}, err
		}
		assigned := []string{}
		for rows.Next() {
			var subjectID string
			if err := rows.Scan(&subjectID); err != nil {
				rows.Close()
				return transition[ChecklistTransitionResult]{}, err
			}
			assigned = append(assigned, subjectID)
		}
		rows.Close()
		decision, err := checklists.Submit(checklists.SubmitInput{
			Actor: actor, AssignedSubjectIDs: assigned, Status: checklists.Status(status),
			Revision: revision, ExpectedRevision: command.ExpectedChecklistRevision,
		})
		if err != nil {
			if !containsString(assigned, actor.SubjectID) {
				return transition[ChecklistTransitionResult]{}, fmt.Errorf("%w: %v", ErrForbidden, err)
			}
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_checklists SET status = $2, revision = $3, submitted_at = $4 WHERE inspection_id = $1
		`, command.InspectionID, string(decision.Status), decision.Revision, now); err != nil {
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("submit checklist: %w", err)
		}
		response := ChecklistTransitionResult{InspectionID: command.InspectionID, Status: decision.Status, Revision: decision.Revision}
		return transition[ChecklistTransitionResult]{
			Response: response, OrganizationID: organizationID, Action: "checklist.submitted",
			EntityType: "inspection_checklist", EntityID: command.InspectionID, EntityVersion: decision.Revision,
			BeforeStatus: status, AfterStatus: string(decision.Status),
			SyncKind: "inspection_checklist", OutboxTopic: "checklist.submitted",
		}, nil
	})
}

type ReopenChecklistCommand struct {
	OperationID               string
	CorrelationID             string
	InspectionID              string
	ExpectedChecklistRevision int64
	Reason                    string
}

func (service *Service) ReopenChecklist(ctx context.Context, actor identity.Principal, command ReopenChecklistCommand) (ChecklistTransitionResult, error) {
	semantic := struct {
		ExpectedRevision int64  `json:"expectedRevision"`
		Reason           string `json:"reason"`
	}{command.ExpectedChecklistRevision, strings.TrimSpace(command.Reason)}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "reopen_checklist", EntityID: command.InspectionID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[ChecklistTransitionResult], error) {
		if !actor.HasRole(identity.RoleLeadInspector, identity.RoleDepartmentManager) {
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("%w: Lead Inspector or Department Manager role required", ErrForbidden)
		}
		var status, organizationID string
		var revision int64
		if err := transaction.QueryRow(ctx, `
			SELECT checklist.status, checklist.revision, inspection.organization_id
			FROM inspection_checklists checklist
			JOIN inspections inspection ON inspection.id = checklist.inspection_id
			WHERE checklist.inspection_id = $1 FOR UPDATE OF checklist
		`, command.InspectionID).Scan(&status, &revision, &organizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[ChecklistTransitionResult]{}, ErrNotFound
			}
			return transition[ChecklistTransitionResult]{}, err
		}
		decision, err := checklists.Reopen(checklists.ReopenInput{
			Actor: actor, Status: checklists.Status(status), Revision: revision,
			ExpectedRevision: command.ExpectedChecklistRevision, Reason: semantic.Reason,
		})
		if err != nil {
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_checklists
			SET status = $2, revision = $3, reopened_at = $4, reopen_reason = $5
			WHERE inspection_id = $1
		`, command.InspectionID, string(decision.Status), decision.Revision, now, semantic.Reason); err != nil {
			return transition[ChecklistTransitionResult]{}, fmt.Errorf("reopen checklist: %w", err)
		}
		response := ChecklistTransitionResult{InspectionID: command.InspectionID, Status: decision.Status, Revision: decision.Revision}
		return transition[ChecklistTransitionResult]{
			Response: response, OrganizationID: organizationID, Action: "checklist.reopened",
			EntityType: "inspection_checklist", EntityID: command.InspectionID, EntityVersion: decision.Revision,
			BeforeStatus: status, AfterStatus: string(decision.Status), Reason: semantic.Reason,
			SyncKind: "inspection_checklist", OutboxTopic: "checklist.reopened",
		}, nil
	})
}

type CreatePotentialFindingCommand struct {
	OperationID                       string
	CorrelationID                     string
	InspectionID                      string
	QuestionID                        string
	ChecklistResponseID               string
	ExpectedChecklistResponseRevision int64
	Title                             string
	Description                       string
	CommentToAuditee                  string
	InternalCAANote                   string
	ExpectedEvidence                  string
	InspectionAttachmentIDs           []string
}

type PotentialFindingResult struct {
	ID           string                   `json:"id"`
	InspectionID string                   `json:"inspectionId"`
	QuestionID   string                   `json:"questionId"`
	Status       potentialfindings.Status `json:"status"`
	Revision     int64                    `json:"revision"`
	FindingID    string                   `json:"findingId,omitempty"`
}

func (service *Service) CreatePotentialFinding(ctx context.Context, actor identity.Principal, command CreatePotentialFindingCommand) (PotentialFindingResult, error) {
	attachmentIDs, err := normalizedUniqueIDs(command.InspectionAttachmentIDs)
	if err != nil {
		return PotentialFindingResult{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	semantic := struct {
		InspectionID            string   `json:"inspectionId"`
		QuestionID              string   `json:"questionId"`
		ResponseID              string   `json:"responseId"`
		ResponseRevision        int64    `json:"responseRevision"`
		Title                   string   `json:"title"`
		Description             string   `json:"description"`
		CommentToAuditee        string   `json:"commentToAuditee"`
		InternalCAANote         string   `json:"internalCaaNote"`
		ExpectedEvidence        string   `json:"expectedEvidence"`
		InspectionAttachmentIDs []string `json:"inspectionAttachmentIds"`
	}{command.InspectionID, command.QuestionID, command.ChecklistResponseID, command.ExpectedChecklistResponseRevision,
		strings.TrimSpace(command.Title), strings.TrimSpace(command.Description), strings.TrimSpace(command.CommentToAuditee),
		strings.TrimSpace(command.InternalCAANote), strings.TrimSpace(command.ExpectedEvidence), attachmentIDs}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "create_potential_finding", EntityID: command.ChecklistResponseID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[PotentialFindingResult], error) {
		if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: Inspector role required", ErrForbidden)
		}
		if semantic.Title == "" || semantic.Description == "" || semantic.CommentToAuditee == "" {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: title, description, and Comment to Auditee are required", ErrInvalid)
		}
		var responseInspectionID, responseQuestionID, assignedSubjectID, answer, organizationID string
		var responseRevision int64
		if err := transaction.QueryRow(ctx, `
			SELECT response.inspection_id, response.question_id, response.assigned_inspector_subject_id,
			       response.response_value, response.revision, inspection.organization_id
			FROM checklist_responses response
			JOIN inspections inspection ON inspection.id = response.inspection_id
			JOIN inspection_checklists checklist ON checklist.inspection_id = response.inspection_id AND checklist.status = 'IN_PROGRESS'
			JOIN inspection_question_assignments assignment
			  ON assignment.inspection_id = response.inspection_id
			 AND assignment.question_id = response.question_id
			 AND assignment.subject_id = $4
			WHERE response.id = $1 AND response.inspection_id = $2 AND response.question_id = $3
			FOR UPDATE OF response
		`, command.ChecklistResponseID, command.InspectionID, command.QuestionID, actor.SubjectID).Scan(
			&responseInspectionID, &responseQuestionID, &assignedSubjectID, &answer, &responseRevision, &organizationID,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[PotentialFindingResult]{}, fmt.Errorf("%w: response is outside the Inspector assignment", ErrForbidden)
			}
			return transition[PotentialFindingResult]{}, err
		}
		if responseRevision != command.ExpectedChecklistResponseRevision {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: stale checklist response revision", ErrConflict)
		}
		if answer != "NON_COMPLIANT" && answer != "OBSERVATION" {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: response does not support a Potential Finding", ErrConflict)
		}
		if err := inspections.ValidatePotentialFindingContext(inspections.PotentialFindingContext{
			AuditID: command.InspectionID, QuestionAuditID: responseInspectionID, ResponseAuditID: responseInspectionID,
			QuestionID: command.QuestionID, ResponseQuestionID: responseQuestionID, AssignedInspectorUserID: assignedSubjectID,
		}, actor); err != nil {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: %v", ErrForbidden, err)
		}
		if len(semantic.InspectionAttachmentIDs) > 0 {
			rows, err := transaction.Query(ctx, `
				SELECT id
				FROM inspection_attachments
				WHERE id = ANY($1)
				  AND inspection_id = $2
				  AND question_id = $3
				  AND checklist_response_id = $4
				  AND organization_id = $5
				  AND created_by_subject_id = $6
				  AND potential_finding_id IS NULL
				ORDER BY id
				FOR UPDATE
			`,
				semantic.InspectionAttachmentIDs,
				command.InspectionID,
				command.QuestionID,
				command.ChecklistResponseID,
				organizationID,
				actor.SubjectID,
			)
			if err != nil {
				return transition[PotentialFindingResult]{}, err
			}
			matched := 0
			for rows.Next() {
				matched++
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return transition[PotentialFindingResult]{}, err
			}
			rows.Close()
			if matched != len(semantic.InspectionAttachmentIDs) {
				return transition[PotentialFindingResult]{}, fmt.Errorf(
					"%w: Inspection Attachment is outside the exact response context",
					ErrForbidden,
				)
			}
		}
		potentialFindingID := service.idGenerator("potential-finding")
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO potential_findings (
				id, inspection_id, checklist_response_id, organization_id, status, finding_basis,
				expected_evidence, comment_to_auditee, internal_caa_note, revision, created_at, updated_at,
				question_id, title, description, created_by_subject_id
			) VALUES (
				$1, $2, $3, $4, 'PENDING_LEAD_REVIEW', $5, NULLIF($6, ''), $7, NULLIF($8, ''), 1, $9, $9,
				$10, $11, $12, $13
			)
		`, potentialFindingID, command.InspectionID, command.ChecklistResponseID, organizationID,
			semantic.Description, semantic.ExpectedEvidence, semantic.CommentToAuditee, semantic.InternalCAANote, now,
			command.QuestionID, semantic.Title, semantic.Description, actor.SubjectID); err != nil {
			return transition[PotentialFindingResult]{}, fmt.Errorf("create Potential Finding: %w", err)
		}
		if len(semantic.InspectionAttachmentIDs) > 0 {
			linkResult, err := transaction.Exec(ctx, `
				UPDATE inspection_attachments
				SET potential_finding_id = $1
				WHERE id = ANY($2) AND potential_finding_id IS NULL
			`, potentialFindingID, semantic.InspectionAttachmentIDs)
			if err != nil {
				return transition[PotentialFindingResult]{}, fmt.Errorf(
					"link Inspection Attachments: %w",
					err,
				)
			}
			if linkResult.RowsAffected() != int64(len(semantic.InspectionAttachmentIDs)) {
				return transition[PotentialFindingResult]{}, ErrConflict
			}
		}
		response := PotentialFindingResult{
			ID: potentialFindingID, InspectionID: command.InspectionID, QuestionID: command.QuestionID,
			Status: potentialfindings.StatusPendingLeadReview, Revision: 1,
		}
		return transition[PotentialFindingResult]{
			Response: response, OrganizationID: organizationID, Action: "potential_finding.created",
			EntityType: "potential_finding", EntityID: potentialFindingID, EntityVersion: 1,
			BeforeStatus: "", AfterStatus: string(potentialfindings.StatusPendingLeadReview),
			SyncKind: "potential_finding", OutboxTopic: "potential_finding.created",
		}, nil
	})
}

func normalizedUniqueIDs(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("identifier is required")
		}
		if _, exists := seen[value]; exists {
			return nil, errors.New("duplicate identifier")
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized, nil
}

type DecidePotentialFindingCommand struct {
	OperationID        string
	CorrelationID      string
	PotentialFindingID string
	ExpectedRevision   int64
	Decision           potentialfindings.Decision
	Reason             string
}

func (service *Service) DecidePotentialFinding(ctx context.Context, actor identity.Principal, command DecidePotentialFindingCommand) (PotentialFindingResult, error) {
	semantic := struct {
		ExpectedRevision int64                      `json:"expectedRevision"`
		Decision         potentialfindings.Decision `json:"decision"`
		Reason           string                     `json:"reason"`
	}{command.ExpectedRevision, command.Decision, strings.TrimSpace(command.Reason)}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "decide_potential_finding", EntityID: command.PotentialFindingID, Semantic: semantic,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[PotentialFindingResult], error) {
		if !actor.HasRole(identity.RoleLeadInspector) {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: Lead Inspector role required", ErrForbidden)
		}
		if command.Decision == potentialfindings.DecisionConvert {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: conversion requires explicit severity command", ErrInvalid)
		}
		var status, inspectionID, questionID, organizationID string
		var revision int64
		if err := transaction.QueryRow(ctx, `
			SELECT status, revision, inspection_id, question_id, organization_id
			FROM potential_findings WHERE id = $1 FOR UPDATE
		`, command.PotentialFindingID).Scan(&status, &revision, &inspectionID, &questionID, &organizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[PotentialFindingResult]{}, ErrNotFound
			}
			return transition[PotentialFindingResult]{}, err
		}
		decision, err := potentialfindings.Decide(potentialfindings.DecideInput{
			Actor: actor, Status: potentialfindings.Status(status), Revision: revision,
			ExpectedRevision: command.ExpectedRevision, Decision: command.Decision, Reason: semantic.Reason,
		})
		if err != nil {
			return transition[PotentialFindingResult]{}, fmt.Errorf("%w: %v", ErrConflict, err)
		}
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			UPDATE potential_findings SET status = $2, revision = $3, updated_at = $4 WHERE id = $1
		`, command.PotentialFindingID, string(decision.Status), decision.Revision, now); err != nil {
			return transition[PotentialFindingResult]{}, fmt.Errorf("record Potential Finding decision: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO review_decisions (
				id, entity_type, entity_id, expected_revision, decision, reason,
				internal_caa_note, decided_by_subject_id, decided_at
			) VALUES ($1, 'potential_finding', $2, $3, $4, $5, $5, $6, $7)
		`, service.idGenerator("review-decision"), command.PotentialFindingID, command.ExpectedRevision,
			string(command.Decision), semantic.Reason, actor.SubjectID, now); err != nil {
			return transition[PotentialFindingResult]{}, fmt.Errorf("append Potential Finding decision: %w", err)
		}
		response := PotentialFindingResult{
			ID: command.PotentialFindingID, InspectionID: inspectionID, QuestionID: questionID,
			Status: decision.Status, Revision: decision.Revision,
		}
		return transition[PotentialFindingResult]{
			Response: response, OrganizationID: organizationID, Action: "potential_finding.decided",
			EntityType: "potential_finding", EntityID: command.PotentialFindingID, EntityVersion: decision.Revision,
			BeforeStatus: status, AfterStatus: string(decision.Status), Reason: semantic.Reason,
			SyncKind: "potential_finding", OutboxTopic: "potential_finding.decided",
		}, nil
	})
}

func validChecklistAnswer(answer string) bool {
	switch answer {
	case "COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED":
		return true
	default:
		return false
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
