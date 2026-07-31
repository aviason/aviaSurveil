package identity

import (
	"context"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// ResolveEffectiveDepartmentAssignments reads only active manager assignments
// whose effective period includes the supplied instant. Missing assignments are
// deliberately represented as an empty result, never as organization authority.
func ResolveEffectiveDepartmentAssignments(ctx context.Context, pool *database.Pool, subjectID string, at time.Time) ([]DepartmentAssignment, error) {
	rows, err := pool.Query(ctx, `
		SELECT membership.department_id, membership.organizational_unit_id, membership.effective_from, membership.effective_to
		FROM (
			SELECT DISTINCT ON (root_id) * FROM caa_department_memberships
			WHERE effective_from <= $2::date
			ORDER BY root_id, effective_from DESC, id DESC
		) membership
			JOIN LATERAL (
				SELECT status FROM caa_department_status_facts
				WHERE department_id = membership.department_id AND effective_from <= $2::date
				ORDER BY effective_from DESC, id DESC LIMIT 1
			) department_status ON department_status.status = 'ACTIVE'
			JOIN LATERAL (
				SELECT status FROM caa_organizational_unit_status_facts
				WHERE organizational_unit_id = membership.organizational_unit_id AND effective_from <= $2::date
				ORDER BY effective_from DESC, id DESC LIMIT 1
			) unit_status ON unit_status.status = 'ACTIVE'
			JOIN caa_organizational_units unit ON unit.id = membership.organizational_unit_id AND unit.department_id = membership.department_id
		WHERE membership.subject_id = $1
		  AND membership_role = 'DEPARTMENT_MANAGER'
		  AND membership.status = 'ACTIVE'
		  AND (membership.effective_to IS NULL OR membership.effective_to > $2::date)
		ORDER BY membership.department_id, membership.organizational_unit_id, membership.effective_from
	`, strings.TrimSpace(subjectID), at.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := []DepartmentAssignment{}
	for rows.Next() {
		var assignment DepartmentAssignment
		var effectiveTo *time.Time
		if err := rows.Scan(&assignment.DepartmentID, &assignment.OrganizationalUnitID, &assignment.EffectiveFrom, &effectiveTo); err != nil {
			return nil, err
		}
		assignment.EffectiveTo = effectiveTo
		assignments = append(assignments, assignment)
	}
	return assignments, rows.Err()
}

func CanTechnicallyReview(principal Principal, departmentID string) bool {
	if !principal.HasRole(RoleDepartmentManager) || strings.TrimSpace(departmentID) == "" {
		return false
	}
	for _, assignment := range principal.DepartmentAssignments {
		if assignment.DepartmentID == departmentID {
			return true
		}
	}
	return false
}

func CanTechnicallyReviewUnit(principal Principal, departmentID, organizationalUnitID string) bool {
	if !CanTechnicallyReview(principal, departmentID) || strings.TrimSpace(organizationalUnitID) == "" {
		return false
	}
	for _, assignment := range principal.DepartmentAssignments {
		if assignment.DepartmentID == departmentID && assignment.OrganizationalUnitID == organizationalUnitID {
			return true
		}
	}
	return false
}

func CanPublishChecklist(principal Principal, departmentID string) bool {
	return CanTechnicallyReview(principal, departmentID)
}
