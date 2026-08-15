package assignments

import (
	"context"
	"errors"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (service *Service) ListTeamMembers(
	ctx context.Context,
	actor identity.Principal,
	role *identity.Role,
	limit int32,
) ([]TeamMember, error) {
	if !CanViewTeamMembers(actor) {
		return nil, ErrForbidden
	}
	organizationFilter := actor.OrganizationID
	if actor.HasRole(identity.RoleAdmin) {
		organizationFilter = ""
	}
	if organizationFilter == "" && !actor.HasRole(identity.RoleAdmin) {
		return nil, ErrForbidden
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	roleFilter := ""
	if role != nil {
		roleFilter = string(*role)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT profile.subject_id, profile.display_name,
		       membership.roles[1], membership.organization_id, profile.revision
		FROM user_profiles profile
		JOIN identity_references identity ON identity.subject_id = profile.subject_id
		JOIN LATERAL (
			SELECT version.organization_id, version.roles
			FROM desired_membership_versions version
			JOIN desired_membership_sync sync
			  ON sync.membership_id = version.membership_id
			 AND sync.subject_id = version.subject_id
			 AND sync.desired_revision = version.revision
			WHERE version.subject_id = profile.subject_id
			  AND version.revision = (
			      SELECT latest.revision
			      FROM desired_membership_versions latest
			      WHERE latest.subject_id = profile.subject_id
			      ORDER BY latest.revision DESC
			      LIMIT 1
			  )
			  AND version.membership_state = 'ACTIVE'
			  AND version.effective_at <= now()
			  AND sync.observed_provider_enabled
			  AND sync.observed_organization_id = version.organization_id
			  AND sync.drift_state = 'IN_SYNC'
			  AND version.roles <@ sync.observed_roles
			  AND sync.observed_roles <@ version.roles
			LIMIT 1
		) membership ON true
		WHERE profile.tombstoned_at IS NULL
		  AND identity.tombstoned_at IS NULL
		  AND ($1 = '' OR $1 = ANY(membership.roles))
		  AND membership.roles[1] <> 'auditee'
		  AND ($3 = '' OR membership.organization_id = $3)
		ORDER BY profile.display_name, profile.subject_id
		LIMIT $2
	`, roleFilter, limit, organizationFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := []TeamMember{}
	for rows.Next() {
		var item TeamMember
		if err := rows.Scan(
			&item.SubjectID, &item.DisplayName, &item.Role,
			&item.OrganizationID, &item.Revision,
		); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func (service *Service) GetTeamMember(
	ctx context.Context,
	actor identity.Principal,
	subjectID string,
) (TeamMember, error) {
	if !CanViewTeamMembers(actor) {
		return TeamMember{}, ErrForbidden
	}
	organizationFilter := actor.OrganizationID
	if actor.HasRole(identity.RoleAdmin) {
		organizationFilter = ""
	}
	if organizationFilter == "" && !actor.HasRole(identity.RoleAdmin) {
		return TeamMember{}, ErrForbidden
	}
	return service.getTeamMember(ctx, subjectID, organizationFilter)
}

func (service *Service) getTeamMember(
	ctx context.Context,
	subjectID string,
	organizationFilter string,
) (TeamMember, error) {
	var output TeamMember
	if err := service.pool.QueryRow(ctx, `
		SELECT profile.subject_id, profile.display_name,
		       membership.roles[1], membership.organization_id, profile.revision
		FROM user_profiles profile
		JOIN identity_references identity ON identity.subject_id = profile.subject_id
		JOIN LATERAL (
			SELECT version.organization_id, version.roles
			FROM desired_membership_versions version
			JOIN desired_membership_sync sync
			  ON sync.membership_id = version.membership_id
			 AND sync.subject_id = version.subject_id
			 AND sync.desired_revision = version.revision
			WHERE version.subject_id = profile.subject_id
			  AND version.revision = (
			      SELECT latest.revision
			      FROM desired_membership_versions latest
			      WHERE latest.subject_id = profile.subject_id
			      ORDER BY latest.revision DESC
			      LIMIT 1
			  )
			  AND version.membership_state = 'ACTIVE'
			  AND version.effective_at <= now()
			  AND sync.observed_provider_enabled
			  AND sync.observed_organization_id = version.organization_id
			  AND sync.drift_state = 'IN_SYNC'
			  AND version.roles <@ sync.observed_roles
			  AND sync.observed_roles <@ version.roles
			LIMIT 1
		) membership ON true
		WHERE profile.subject_id = $1
		  AND profile.tombstoned_at IS NULL
		  AND identity.tombstoned_at IS NULL
		  AND membership.roles[1] <> 'auditee'
		  AND ($2 = '' OR membership.organization_id = $2)
	`, subjectID, organizationFilter).Scan(
		&output.SubjectID, &output.DisplayName, &output.Role,
		&output.OrganizationID, &output.Revision,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamMember{}, ErrNotFound
		}
		return TeamMember{}, err
	}
	if output.Role == identity.RoleAuditee || output.Role == "" {
		return TeamMember{}, ErrNotFound
	}
	return output, nil
}

func (service *Service) ListAuditTeams(
	ctx context.Context,
	actor identity.Principal,
	limit int32,
) ([]TeamAudit, error) {
	if !CanViewAuditTeams(actor) {
		return nil, ErrForbidden
	}
	organizationFilter := actor.OrganizationID
	if actor.HasRole(identity.RoleAdmin) {
		organizationFilter = ""
	}
	if organizationFilter == "" && !actor.HasRole(identity.RoleAdmin) {
		return nil, ErrForbidden
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := service.pool.Query(ctx, `
		SELECT inspection_id
		FROM audit_assignments
		WHERE tombstoned_at IS NULL
		  AND ($2 = '' OR organization_id = $2)
		ORDER BY scheduled_start_date, inspection_id
		LIMIT $1
	`, limit, organizationFilter)
	if err != nil {
		return nil, err
	}
	inspectionIDs := []string{}
	for rows.Next() {
		var inspectionID string
		if err := rows.Scan(&inspectionID); err != nil {
			rows.Close()
			return nil, err
		}
		inspectionIDs = append(inspectionIDs, inspectionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	output := make([]TeamAudit, 0, len(inspectionIDs))
	for _, inspectionID := range inspectionIDs {
		item, err := service.getAuditTeam(ctx, inspectionID, organizationFilter)
		if err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, nil
}

func (service *Service) GetAuditTeam(
	ctx context.Context,
	actor identity.Principal,
	inspectionID string,
) (TeamAudit, error) {
	if !CanViewAuditTeams(actor) {
		return TeamAudit{}, ErrForbidden
	}
	organizationFilter := actor.OrganizationID
	if actor.HasRole(identity.RoleAdmin) {
		organizationFilter = ""
	}
	if organizationFilter == "" && !actor.HasRole(identity.RoleAdmin) {
		return TeamAudit{}, ErrForbidden
	}
	return service.getAuditTeam(ctx, inspectionID, organizationFilter)
}

func (service *Service) getAuditTeam(
	ctx context.Context,
	inspectionID string,
	organizationFilter string,
) (TeamAudit, error) {
	var output TeamAudit
	var assignmentID, leadSubjectID string
	var startDate, endDate pgtype.Date
	if err := service.pool.QueryRow(ctx, `
		SELECT assignment.id, inspection.id, inspection.organization_id,
		       organization.legal_name, inspection.title, assignment.status,
		       assignment.scheduled_start_date, assignment.scheduled_end_date,
		       assignment.lead_subject_id, assignment.revision
		FROM audit_assignments assignment
		JOIN inspections inspection ON inspection.id = assignment.inspection_id
		JOIN organizations organization ON organization.id = inspection.organization_id
		WHERE inspection.id = $1
		  AND ($2 = '' OR inspection.organization_id = $2)
		  AND assignment.tombstoned_at IS NULL
		  AND inspection.tombstoned_at IS NULL
	`, inspectionID, organizationFilter).Scan(
		&assignmentID, &output.AuditID, &output.OrganizationID,
		&output.OrganizationName, &output.Title, &output.Status,
		&startDate, &endDate, &leadSubjectID, &output.Revision,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamAudit{}, ErrNotFound
		}
		return TeamAudit{}, err
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		output.ScheduledStartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		output.ScheduledEndDate = &value
	}
	lead, err := service.getTeamMember(ctx, leadSubjectID, organizationFilter)
	if err != nil {
		return TeamAudit{}, err
	}
	output.LeadInspector = lead

	memberRows, err := service.pool.Query(ctx, `
		SELECT member.subject_id
		FROM audit_team_members member
		WHERE member.assignment_id = $1 AND member.removed_at IS NULL
		ORDER BY CASE member.member_role WHEN 'LEAD_INSPECTOR' THEN 0 ELSE 1 END,
		         member.subject_id
	`, assignmentID)
	if err != nil {
		return TeamAudit{}, err
	}
	output.Members = []TeamMember{}
	for memberRows.Next() {
		var subjectID string
		if err := memberRows.Scan(&subjectID); err != nil {
			memberRows.Close()
			return TeamAudit{}, err
		}
		member, err := service.getTeamMember(ctx, subjectID, organizationFilter)
		if err != nil {
			memberRows.Close()
			return TeamAudit{}, err
		}
		output.Members = append(output.Members, member)
	}
	if err := memberRows.Err(); err != nil {
		memberRows.Close()
		return TeamAudit{}, err
	}
	memberRows.Close()

	assignmentRows, err := service.pool.Query(ctx, `
		SELECT question_id, subject_id
		FROM audit_question_assignments
		WHERE assignment_id = $1
		ORDER BY question_id, subject_id
	`, assignmentID)
	if err != nil {
		return TeamAudit{}, err
	}
	byQuestion := map[string][]string{}
	questionOrder := []string{}
	for assignmentRows.Next() {
		var questionID, subjectID string
		if err := assignmentRows.Scan(&questionID, &subjectID); err != nil {
			assignmentRows.Close()
			return TeamAudit{}, err
		}
		if _, exists := byQuestion[questionID]; !exists {
			questionOrder = append(questionOrder, questionID)
		}
		byQuestion[questionID] = append(byQuestion[questionID], subjectID)
	}
	if err := assignmentRows.Err(); err != nil {
		assignmentRows.Close()
		return TeamAudit{}, err
	}
	assignmentRows.Close()
	output.Assignments = make([]TeamQuestionAssignment, 0, len(questionOrder))
	for _, questionID := range questionOrder {
		output.Assignments = append(output.Assignments, TeamQuestionAssignment{
			QuestionID: questionID, AssignedMemberSubjectIDs: byQuestion[questionID],
		})
	}

	historyRows, err := service.pool.Query(ctx, `
		SELECT event_id, occurred_at, actor_subject_id, action,
		       COALESCE(reason, NULLIF(after_status, ''), action)
		FROM audit_events
		WHERE entity_id IN ($1, $2)
		ORDER BY occurred_at, event_id
	`, assignmentID, inspectionID)
	if err != nil {
		return TeamAudit{}, err
	}
	output.History = []TeamHistory{}
	for historyRows.Next() {
		var history TeamHistory
		if err := historyRows.Scan(
			&history.EventID, &history.OccurredAt, &history.ActorSubjectID,
			&history.Action, &history.Detail,
		); err != nil {
			historyRows.Close()
			return TeamAudit{}, err
		}
		output.History = append(output.History, history)
	}
	if err := historyRows.Err(); err != nil {
		historyRows.Close()
		return TeamAudit{}, err
	}
	historyRows.Close()
	return output, nil
}
