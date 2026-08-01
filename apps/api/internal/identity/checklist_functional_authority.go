package identity

import (
	"sort"
	"strings"
	"time"
)

// FunctionalPermission is an effective-dated capability assignment. These are
// deliberately not top-level application roles.
type FunctionalPermission string

const (
	FunctionalPermissionRegulatorySourceOwner FunctionalPermission = "REGULATORY_SOURCE_OWNER"
	FunctionalPermissionChecklistReviewer     FunctionalPermission = "CHECKLIST_REVIEWER"
)

type FunctionalAssignmentStatus string

const (
	FunctionalAssignmentActive  FunctionalAssignmentStatus = "ACTIVE"
	FunctionalAssignmentRevoked FunctionalAssignmentStatus = "REVOKED"
	FunctionalAssignmentExpired FunctionalAssignmentStatus = "EXPIRED"
)

type FunctionalScopeKind string

const (
	ScopeSourceIdentity              FunctionalScopeKind = "SOURCE_IDENTITY"
	ScopeReviewedSourceSet           FunctionalScopeKind = "REVIEWED_SOURCE_SET"
	ScopeDepartmentChecklists        FunctionalScopeKind = "DEPARTMENT_CHECKLISTS"
	ScopeProviderChecklists          FunctionalScopeKind = "PROVIDER_CHECKLISTS"
	ScopeReviewedSourceSetChecklists FunctionalScopeKind = "REVIEWED_SOURCE_SET_CHECKLISTS"
)

// FunctionalAssignmentScope contains only exact scope values. Empty values
// are never wildcards; ValidateFunctionalAssignmentScope rejects them.
type FunctionalAssignmentScope struct {
	Kind                     FunctionalScopeKind
	SourceVersionID          string
	SourceID                 string
	ChainRole                string
	ReviewedSourceSetID      string
	ReviewedSourceSetVersion int64
	ReviewedSourceSetDigest  string
	ProviderScopeID          string
	TargetID                 string
	InspectionType           string
	DepartmentID             string
	OrganizationalUnitID     string
	CandidateRootID          string
}

type FunctionalAssignment struct {
	AssignmentID  string
	RootID        string
	SupersedesID  string
	Version       int64
	SubjectID     string
	MembershipID  string
	Permission    FunctionalPermission
	Scope         FunctionalAssignmentScope
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Status        FunctionalAssignmentStatus
	CreatedAt     time.Time
}

type FunctionalAuthorizationQuery struct {
	SubjectID    string
	MembershipID string
	Permission   FunctionalPermission
	At           time.Time
	Scope        FunctionalAssignmentScope
}

// ResolveCurrentFunctionalAssignments selects the latest successor for every
// assignment root before applying subject, permission, scope, or effective-date
// filters. This prevents a stale/expired predecessor from becoming authority
// merely because the latest successor belongs to another scope or is future.
func ResolveCurrentFunctionalAssignments(assignments []FunctionalAssignment, at time.Time) []FunctionalAssignment {
	latest := make(map[string]FunctionalAssignment, len(assignments))
	for _, assignment := range assignments {
		if strings.TrimSpace(assignment.RootID) == "" || assignment.Version <= 0 {
			continue
		}
		current, exists := latest[assignment.RootID]
		if !exists || assignment.Version > current.Version || (assignment.Version == current.Version && assignment.CreatedAt.After(current.CreatedAt)) {
			latest[assignment.RootID] = assignment
		}
	}
	resolved := make([]FunctionalAssignment, 0, len(latest))
	for _, assignment := range latest {
		if assignment.Status != FunctionalAssignmentActive || assignment.EffectiveFrom.After(at) || (assignment.EffectiveTo != nil && !at.Before(*assignment.EffectiveTo)) {
			continue
		}
		if ValidateFunctionalAssignmentScope(assignment.Permission, assignment.Scope) != nil {
			continue
		}
		resolved = append(resolved, assignment)
	}
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].RootID != resolved[j].RootID {
			return resolved[i].RootID < resolved[j].RootID
		}
		return resolved[i].Version < resolved[j].Version
	})
	return resolved
}

func AuthorizeFunctionalAssignment(assignments []FunctionalAssignment, query FunctionalAuthorizationQuery) bool {
	if strings.TrimSpace(query.SubjectID) == "" || strings.TrimSpace(query.MembershipID) == "" || query.Permission == "" || strings.TrimSpace(query.Scope.DepartmentID) == "" || strings.TrimSpace(query.Scope.OrganizationalUnitID) == "" {
		return false
	}
	if ValidateFunctionalAssignmentScope(query.Permission, query.Scope) != nil {
		return false
	}
	for _, assignment := range ResolveCurrentFunctionalAssignments(assignments, query.At) {
		if assignment.SubjectID == query.SubjectID && assignment.MembershipID == query.MembershipID && assignment.Permission == query.Permission && assignment.Scope == query.Scope {
			return true
		}
	}
	return false
}

func ValidateFunctionalAssignmentScope(permission FunctionalPermission, scope FunctionalAssignmentScope) error {
	if strings.TrimSpace(scope.DepartmentID) == "" || strings.TrimSpace(scope.OrganizationalUnitID) == "" {
		return errInvalidFunctionalScope
	}
	switch permission {
	case FunctionalPermissionRegulatorySourceOwner:
		switch scope.Kind {
		case ScopeSourceIdentity:
			if strings.TrimSpace(scope.SourceVersionID) == "" || strings.TrimSpace(scope.SourceID) == "" || strings.TrimSpace(scope.ChainRole) == "" {
				return errInvalidFunctionalScope
			}
		case ScopeReviewedSourceSet:
			if strings.TrimSpace(scope.ReviewedSourceSetID) == "" || scope.ReviewedSourceSetVersion <= 0 || strings.TrimSpace(scope.ReviewedSourceSetDigest) == "" || strings.TrimSpace(scope.ProviderScopeID) == "" || strings.TrimSpace(scope.TargetID) == "" || strings.TrimSpace(scope.InspectionType) == "" {
				return errInvalidFunctionalScope
			}
		default:
			return errInvalidFunctionalScope
		}
	case FunctionalPermissionChecklistReviewer:
		switch scope.Kind {
		case ScopeDepartmentChecklists:
		case ScopeProviderChecklists:
			if strings.TrimSpace(scope.ProviderScopeID) == "" {
				return errInvalidFunctionalScope
			}
		case ScopeReviewedSourceSetChecklists:
			if strings.TrimSpace(scope.ReviewedSourceSetID) == "" || scope.ReviewedSourceSetVersion <= 0 || strings.TrimSpace(scope.ReviewedSourceSetDigest) == "" {
				return errInvalidFunctionalScope
			}
		default:
			return errInvalidFunctionalScope
		}
	default:
		return errInvalidFunctionalScope
	}
	return nil
}

var errInvalidFunctionalScope = &functionalScopeError{}

type functionalScopeError struct{}

func (*functionalScopeError) Error() string { return "invalid functional assignment scope" }
