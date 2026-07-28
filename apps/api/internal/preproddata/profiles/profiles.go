package profiles

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"time"
)

var (
	ErrUnknownProfile  = errors.New("unknown preprod data profile")
	ErrProfileMutation = errors.New("preprod data profile differs from frozen catalog")
)

type ResourceEnvelope struct {
	SeedRequired         bool      `json:"seedRequired"`
	ClockOrigin          time.Time `json:"clockOrigin"`
	IdentityNamespace    string    `json:"identityNamespace"`
	CPUCores             int       `json:"cpuCores"`
	MemoryMiB            int64     `json:"memoryMiB"`
	DiskMiB              int64     `json:"diskMiB"`
	ObjectBytes          int64     `json:"objectBytes"`
	DurationSeconds      int64     `json:"durationSeconds"`
	QualificationSeconds int64     `json:"qualificationSeconds,omitempty"`
	CleanupSeconds       int64     `json:"cleanupSeconds"`
}

type Catalogs struct {
	RouteCount            int      `json:"routeCount"`
	VisibleActionCoverage string   `json:"visibleActionCoverage"`
	Roles                 []string `json:"roles"`
	LifecycleScenarios    []string `json:"lifecycleScenarios"`
}

type Profile struct {
	Name                  string                      `json:"name"`
	Version               string                      `json:"version"`
	Status                string                      `json:"status"`
	ImplementationAllowed bool                        `json:"implementationAllowed"`
	ChangePolicy          string                      `json:"changePolicy"`
	Catalogs              Catalogs                    `json:"catalogs"`
	ResourceEnvelope      ResourceEnvelope            `json:"resourceEnvelope"`
	ExpectedCounts        map[string]int64            `json:"expectedCounts"`
	ExactDistributions    map[string]map[string]int64 `json:"exactDistributions"`
}

var clockOrigin = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

var roles = []string{
	"inspector", "leadInspector", "manager", "finance", "gm",
	"executiveDirector", "auditee", "admin",
}

var lifecycleScenarios = []string{
	"planned", "active", "overdue", "returned", "rejected", "corrected",
	"superseded", "reopened", "partially-closed", "not-closed",
	"authorized-closed", "verified-closed",
}

var catalog = frozenCatalog()

func frozenCatalog() map[string]Profile {
	catalog := map[string]Profile{
		"smoke@1.0.0": profile(
			"smoke", "synthetic-smoke-v1", 2, 1024, 2048, 134217728, 120, 60,
			map[string]int64{
				"organizations": 3, "providerAccounts": 9,
				"desiredMembershipVersions": 18, "applicationProfiles": 9,
				"invitations": 6, "recoveryRequests": 2, "mfaEnrollments": 9,
				"sessions": 18, "offlineGrants": 4, "surveillancePlans": 4,
				"planningApprovals": 12, "audits": 2, "assignments": 3,
				"checklistTemplates": 4, "checklistTemplateVersions": 6,
				"checklistQuestions": 24, "inspectionPackages": 2,
				"checklistResponses": 24, "potentialFindings": 12, "findings": 8,
				"capRevisions": 12, "evidenceReferences": 8, "evidenceVersions": 16,
				"reviewDecisions": 16, "reportVersions": 6, "communications": 16,
				"notifications": 24, "auditEvents": 250, "outboxMessages": 80,
				"deliveryJobs": 48, "scannerJobs": 16, "renderJobs": 6,
				"objects": 22, "objectVersions": 24, "calendarRecords": 20,
				"syncChanges": 120, "routeDispositions": 86,
				"visibleActionDispositions": 306, "identityLifecycleCases": 18,
				"lifecycleScenarioCases": 12,
			},
			map[string]map[string]int64{
				"organizations": {"caa": 1, "auditee": 2},
				"providerAccounts": {
					"inspector": 1, "leadInspector": 1, "manager": 1, "finance": 1,
					"gm": 1, "executiveDirector": 1, "auditee": 2, "admin": 1,
				},
				"desiredMembershipVersions": {
					"requested": 2, "invited": 2, "active": 8, "suspended": 2,
					"deactivated": 2, "reactivation-pending": 2,
				},
				"invitations": {
					"issued": 1, "delivered": 1, "retryable-failure": 1,
					"expired": 1, "consumed": 1, "cancelled": 1,
				},
				"mfaEnrollments": {
					"enrolled": 5, "enrollment-required": 0, "reset-pending": 1,
					"unenrolled": 3,
				},
				"audits": {
					"planned": 1, "active": 1, "overdue": 0, "verified-closed": 0,
				},
				"potentialFindings": {
					"pending": 4, "returned": 2, "rejected": 2, "corrected": 2,
					"converted": 2,
				},
				"findings": {
					"open": 1, "overdue": 1, "reopened": 1, "partially-closed": 1,
					"not-closed": 1, "authorized-closed": 1, "verified-closed": 2,
				},
				"capRevisions": {
					"draft": 2, "submitted": 2, "returned": 2, "rejected": 2,
					"corrected": 2, "superseded": 1, "accepted": 1,
				},
				"evidenceVersions": {
					"uploaded": 4, "returned": 2, "rejected": 2, "corrected": 2,
					"superseded": 2, "accepted": 4,
				},
				"reportVersions": {
					"draft": 1, "returned": 1, "rejected": 1, "corrected": 1,
					"issued": 2,
				},
				"routeDispositions": {
					"authorized-data": 60, "intentional-empty": 10, "denied": 16,
				},
				"visibleActionDispositions": {
					"executable": 200, "disabled-by-role": 50, "disabled-by-state": 56,
				},
				"identityLifecycleCases": {
					"requested": 1, "invited": 1, "active": 5, "suspended": 1,
					"deactivated": 1, "reactivation-pending": 1, "role-changed": 1,
					"transferred": 1, "mfa-reset": 1, "forced-logout": 1,
					"invitation-expired": 1, "provider-unavailable": 1,
					"provider-drift": 1, "recovered": 1,
				},
				"lifecycleScenarioCases": equalDistribution(1),
			},
		),
		"acceptance@1.0.0": profile(
			"acceptance", "synthetic-acceptance-v1", 4, 4096, 20480,
			2147483648, 1200, 600,
			map[string]int64{
				"organizations": 25, "providerAccounts": 250,
				"desiredMembershipVersions": 350, "applicationProfiles": 250,
				"invitations": 100, "recoveryRequests": 25, "mfaEnrollments": 250,
				"sessions": 500, "offlineGrants": 125, "surveillancePlans": 1250,
				"planningApprovals": 4000, "audits": 1000, "assignments": 1500,
				"checklistTemplates": 50, "checklistTemplateVersions": 100,
				"checklistQuestions": 500, "inspectionPackages": 1000,
				"checklistResponses": 10000, "potentialFindings": 4500,
				"findings": 3000, "capRevisions": 4500,
				"evidenceReferences": 3000, "evidenceVersions": 6000,
				"reviewDecisions": 6000, "reportVersions": 2000,
				"communications": 8000, "notifications": 12000,
				"auditEvents": 100000, "outboxMessages": 30000,
				"deliveryJobs": 20000, "scannerJobs": 6000, "renderJobs": 2000,
				"objects": 8000, "objectVersions": 9000, "calendarRecords": 5000,
				"syncChanges": 50000, "routeDispositions": 86,
				"visibleActionDispositions": 306, "identityLifecycleCases": 250,
				"lifecycleScenarioCases": 1200,
			},
			map[string]map[string]int64{
				"organizations": {"caa": 1, "auditee": 24},
				"providerAccounts": {
					"inspector": 70, "leadInspector": 25, "manager": 25,
					"finance": 20, "gm": 20, "executiveDirector": 10,
					"auditee": 70, "admin": 10,
				},
				"desiredMembershipVersions": {
					"requested": 35, "invited": 35, "active": 200, "suspended": 30,
					"deactivated": 30, "reactivation-pending": 20,
				},
				"invitations": {
					"issued": 15, "delivered": 15, "retryable-failure": 10,
					"expired": 15, "consumed": 35, "cancelled": 10,
				},
				"mfaEnrollments": {
					"enrolled": 160, "enrollment-required": 0, "reset-pending": 20,
					"unenrolled": 70,
				},
				"audits": {
					"planned": 200, "active": 400, "overdue": 100,
					"verified-closed": 300,
				},
				"potentialFindings": {
					"pending": 900, "returned": 600, "rejected": 600,
					"corrected": 600, "converted": 1800,
				},
				"findings": {
					"open": 900, "overdue": 450, "reopened": 300,
					"partially-closed": 450, "not-closed": 300,
					"authorized-closed": 150, "verified-closed": 450,
				},
				"capRevisions": {
					"draft": 450, "submitted": 900, "returned": 675, "rejected": 450,
					"corrected": 675, "superseded": 450, "accepted": 900,
				},
				"evidenceVersions": {
					"uploaded": 1200, "returned": 900, "rejected": 600,
					"corrected": 900, "superseded": 600, "accepted": 1800,
				},
				"reportVersions": {
					"draft": 400, "returned": 300, "rejected": 200,
					"corrected": 300, "issued": 800,
				},
				"routeDispositions": {
					"authorized-data": 70, "intentional-empty": 8, "denied": 8,
				},
				"visibleActionDispositions": {
					"executable": 240, "disabled-by-role": 30, "disabled-by-state": 36,
				},
				"identityLifecycleCases": {
					"requested": 25, "invited": 25, "active": 100, "suspended": 20,
					"deactivated": 20, "reactivation-pending": 10, "role-changed": 10,
					"transferred": 10, "mfa-reset": 10, "forced-logout": 5,
					"invitation-expired": 5, "provider-unavailable": 3,
					"provider-drift": 3, "recovered": 4,
				},
				"lifecycleScenarioCases": equalDistribution(100),
			},
		),
		"realistic@1.0.0": profile(
			"realistic", "synthetic-realistic-v1", 8, 12288, 51200,
			21474836480, 7200, 2700,
			map[string]int64{
				"organizations": 100, "providerAccounts": 2000,
				"desiredMembershipVersions": 3000, "applicationProfiles": 2000,
				"invitations": 800, "recoveryRequests": 200, "mfaEnrollments": 2000,
				"sessions": 4000, "offlineGrants": 1000,
				"surveillancePlans": 25000, "planningApprovals": 80000,
				"audits": 20000, "assignments": 30000, "checklistTemplates": 200,
				"checklistTemplateVersions": 400, "checklistQuestions": 5000,
				"inspectionPackages": 20000, "checklistResponses": 250000,
				"potentialFindings": 90000, "findings": 60000,
				"capRevisions": 100000, "evidenceReferences": 100000,
				"evidenceVersions": 200000, "reviewDecisions": 200000,
				"reportVersions": 75000, "communications": 400000,
				"notifications": 600000, "auditEvents": 5000000,
				"outboxMessages": 1500000, "deliveryJobs": 1000000,
				"scannerJobs": 200000, "renderJobs": 75000, "objects": 275000,
				"objectVersions": 350000, "calendarRecords": 100000,
				"syncChanges": 2500000, "routeDispositions": 86,
				"visibleActionDispositions": 306, "identityLifecycleCases": 2000,
				"lifecycleScenarioCases": 24000,
			},
			scaleDistributions(99, 2000),
		),
		"stress@1.0.0": profile(
			"stress", "synthetic-stress-v1", 12, 12288, 65536,
			8589934592, 28800, 5400,
			map[string]int64{
				"organizations": 200, "providerAccounts": 4000,
				"desiredMembershipVersions": 6000, "applicationProfiles": 4000,
				"invitations": 1600, "recoveryRequests": 400,
				"mfaEnrollments": 4000, "sessions": 8000, "offlineGrants": 2000,
				"surveillancePlans": 50000, "planningApprovals": 160000,
				"audits": 40000, "assignments": 60000, "checklistTemplates": 400,
				"checklistTemplateVersions": 800, "checklistQuestions": 10000,
				"inspectionPackages": 40000, "checklistResponses": 500000,
				"potentialFindings": 180000, "findings": 120000,
				"capRevisions": 200000, "evidenceReferences": 200000,
				"evidenceVersions": 400000, "reviewDecisions": 400000,
				"reportVersions": 150000, "communications": 800000,
				"notifications": 1200000, "auditEvents": 10000000,
				"outboxMessages": 3000000, "deliveryJobs": 2000000,
				"scannerJobs": 400000, "renderJobs": 150000, "objects": 550000,
				"objectVersions": 700000, "calendarRecords": 200000,
				"syncChanges": 5000000, "routeDispositions": 86,
				"visibleActionDispositions": 306, "identityLifecycleCases": 4000,
				"lifecycleScenarioCases": 48000,
			},
			scaleDistributions(199, 4000),
		),
	}
	catalog["realistic@1.1.0"] = localQualificationProfile(
		catalog["acceptance@1.0.0"],
		"realistic",
		"synthetic-realistic-local-v1-1",
		2,
		8,
		8192,
		20480,
		2147483648,
		900,
		900,
		300,
	)
	catalog["stress@1.1.0"] = localQualificationProfile(
		catalog["acceptance@1.0.0"],
		"stress",
		"synthetic-stress-local-v1-1",
		4,
		12,
		12288,
		32768,
		536870912,
		1800,
		1800,
		300,
	)
	return catalog
}

func profile(
	name, identityNamespace string,
	cpu int,
	memoryMiB, diskMiB, objectBytes, durationSeconds, cleanupSeconds int64,
	counts map[string]int64,
	distributions map[string]map[string]int64,
) Profile {
	return Profile{
		Name: name, Version: "1.0.0",
		Status:                "approved — owner decision recorded",
		ImplementationAllowed: false,
		ChangePolicy:          "new-version-required",
		Catalogs: Catalogs{
			RouteCount: 86, VisibleActionCoverage: "complete",
			Roles: slicesClone(roles), LifecycleScenarios: slicesClone(lifecycleScenarios),
		},
		ResourceEnvelope: ResourceEnvelope{
			SeedRequired: true, ClockOrigin: clockOrigin,
			IdentityNamespace: identityNamespace, CPUCores: cpu,
			MemoryMiB: memoryMiB, DiskMiB: diskMiB, ObjectBytes: objectBytes,
			DurationSeconds: durationSeconds, CleanupSeconds: cleanupSeconds,
		},
		ExpectedCounts: counts, ExactDistributions: distributions,
	}
}

func localQualificationProfile(
	source Profile,
	name, identityNamespace string,
	multiplier int64,
	cpu int,
	memoryMiB, diskMiB, objectBytes, durationSeconds,
	qualificationSeconds, cleanupSeconds int64,
) Profile {
	counts := make(map[string]int64, len(source.ExpectedCounts))
	distributions := make(
		map[string]map[string]int64,
		len(source.ExactDistributions),
	)
	preserved := map[string]bool{
		"routeDispositions":         true,
		"visibleActionDispositions": true,
	}
	for family, count := range source.ExpectedCounts {
		if preserved[family] {
			counts[family] = count
			continue
		}
		counts[family] = count * multiplier
	}
	for family, distribution := range source.ExactDistributions {
		scaled := make(map[string]int64, len(distribution))
		for state, count := range distribution {
			if preserved[family] {
				scaled[state] = count
				continue
			}
			scaled[state] = count * multiplier
		}
		distributions[family] = scaled
	}
	distributions["organizations"] = map[string]int64{
		"caa":     1,
		"auditee": counts["organizations"] - 1,
	}
	return Profile{
		Name: name, Version: "1.1.0",
		Status:                "approved — owner decision recorded",
		ImplementationAllowed: false,
		ChangePolicy:          "new-version-required",
		Catalogs: Catalogs{
			RouteCount: 86, VisibleActionCoverage: "complete",
			Roles: slicesClone(roles), LifecycleScenarios: slicesClone(lifecycleScenarios),
		},
		ResourceEnvelope: ResourceEnvelope{
			SeedRequired: true, ClockOrigin: clockOrigin,
			IdentityNamespace: identityNamespace, CPUCores: cpu,
			MemoryMiB: memoryMiB, DiskMiB: diskMiB, ObjectBytes: objectBytes,
			DurationSeconds:      durationSeconds,
			QualificationSeconds: qualificationSeconds,
			CleanupSeconds:       cleanupSeconds,
		},
		ExpectedCounts: counts, ExactDistributions: distributions,
	}
}

func scaleDistributions(auditeeOrganizations, identityCases int64) map[string]map[string]int64 {
	scale := identityCases / 2000
	return map[string]map[string]int64{
		"organizations": {"caa": 1, "auditee": auditeeOrganizations},
		"providerAccounts": {
			"inspector": 600 * scale, "leadInspector": 200 * scale,
			"manager": 200 * scale, "finance": 150 * scale, "gm": 150 * scale,
			"executiveDirector": 100 * scale, "auditee": 500 * scale,
			"admin": 100 * scale,
		},
		"desiredMembershipVersions": {
			"requested": 300 * scale, "invited": 300 * scale,
			"active": 1800 * scale, "suspended": 200 * scale,
			"deactivated": 250 * scale, "reactivation-pending": 150 * scale,
		},
		"invitations": {
			"issued": 120 * scale, "delivered": 120 * scale,
			"retryable-failure": 80 * scale, "expired": 120 * scale,
			"consumed": 280 * scale, "cancelled": 80 * scale,
		},
		"mfaEnrollments": {
			"enrolled": 1300 * scale, "enrollment-required": 0,
			"reset-pending": 150 * scale, "unenrolled": 550 * scale,
		},
		"audits": {
			"planned": 4000 * scale, "active": 8000 * scale,
			"overdue": 2000 * scale, "verified-closed": 6000 * scale,
		},
		"potentialFindings": {
			"pending": 18000 * scale, "returned": 12000 * scale,
			"rejected": 12000 * scale, "corrected": 12000 * scale,
			"converted": 36000 * scale,
		},
		"findings": {
			"open": 18000 * scale, "overdue": 9000 * scale,
			"reopened": 6000 * scale, "partially-closed": 9000 * scale,
			"not-closed": 6000 * scale, "authorized-closed": 3000 * scale,
			"verified-closed": 9000 * scale,
		},
		"capRevisions": {
			"draft": 10000 * scale, "submitted": 20000 * scale,
			"returned": 15000 * scale, "rejected": 10000 * scale,
			"corrected": 15000 * scale, "superseded": 10000 * scale,
			"accepted": 20000 * scale,
		},
		"evidenceVersions": {
			"uploaded": 40000 * scale, "returned": 30000 * scale,
			"rejected": 20000 * scale, "corrected": 30000 * scale,
			"superseded": 20000 * scale, "accepted": 60000 * scale,
		},
		"reportVersions": {
			"draft": 15000 * scale, "returned": 11250 * scale,
			"rejected": 7500 * scale, "corrected": 11250 * scale,
			"issued": 30000 * scale,
		},
		"routeDispositions": {
			"authorized-data": 72, "intentional-empty": 6, "denied": 8,
		},
		"visibleActionDispositions": {
			"executable": 250, "disabled-by-role": 24, "disabled-by-state": 32,
		},
		"identityLifecycleCases": {
			"requested": 200 * scale, "invited": 200 * scale,
			"active": 800 * scale, "suspended": 160 * scale,
			"deactivated": 160 * scale, "reactivation-pending": 80 * scale,
			"role-changed": 80 * scale, "transferred": 80 * scale,
			"mfa-reset": 80 * scale, "forced-logout": 40 * scale,
			"invitation-expired": 40 * scale, "provider-unavailable": 24 * scale,
			"provider-drift": 24 * scale, "recovered": 32 * scale,
		},
		"lifecycleScenarioCases": equalDistribution(2000 * scale),
	}
}

func equalDistribution(count int64) map[string]int64 {
	result := make(map[string]int64, len(lifecycleScenarios))
	for _, state := range lifecycleScenarios {
		result[state] = count
	}
	return result
}

func Lookup(name, version string) (Profile, error) {
	found, ok := catalog[name+"@"+version]
	if !ok {
		return Profile{}, fmt.Errorf("%w: %s@%s", ErrUnknownProfile, name, version)
	}
	found.Catalogs.Roles = slicesClone(found.Catalogs.Roles)
	found.Catalogs.LifecycleScenarios = slicesClone(found.Catalogs.LifecycleScenarios)
	found.ExpectedCounts = maps.Clone(found.ExpectedCounts)
	found.ExactDistributions = cloneDistributions(found.ExactDistributions)
	return found, nil
}

func ValidateFrozen(candidate Profile) error {
	frozen, err := Lookup(candidate.Name, candidate.Version)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(candidate, frozen) {
		return fmt.Errorf(
			"%w: %s@%s requires a new version",
			ErrProfileMutation,
			candidate.Name,
			candidate.Version,
		)
	}
	return nil
}

func cloneDistributions(
	source map[string]map[string]int64,
) map[string]map[string]int64 {
	output := make(map[string]map[string]int64, len(source))
	for family, distribution := range source {
		output[family] = maps.Clone(distribution)
	}
	return output
}

func slicesClone[T any](source []T) []T {
	return append([]T(nil), source...)
}
