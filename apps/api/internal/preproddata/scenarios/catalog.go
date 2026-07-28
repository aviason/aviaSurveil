package scenarios

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
)

type Catalog struct {
	Routes  []RouteCoverage  `json:"routes"`
	Actions []ActionCoverage `json:"actions"`
}

type RouteCoverage struct {
	AuditID    string `json:"auditId"`
	SurfaceID  string `json:"surfaceId"`
	Role       string `json:"role"`
	ScreenName string `json:"screenName"`
}

type ActionCoverage struct {
	ActionID   string `json:"actionId"`
	SurfaceID  string `json:"surfaceId"`
	ControlKey string `json:"controlKey"`
	Boundary   string `json:"boundary"`
}

func ParseCatalogs(routeSource, ledgerSource []byte) (Catalog, error) {
	var routes []struct {
		AuditID    string `json:"auditId"`
		Role       string `json:"role"`
		ScreenName string `json:"screenName"`
	}
	if err := decodeExactJSON(routeSource, &routes); err != nil {
		return Catalog{}, fmt.Errorf("decode route catalog: %w", err)
	}
	var ledger struct {
		ReactScope struct {
			SurfaceIDs []string `json:"reactParitySurfaceIds"`
		} `json:"reactScope"`
		ActionEvidence []struct {
			SurfaceID  string   `json:"surfaceId"`
			Scope      string   `json:"scope"`
			Profiles   []string `json:"profiles"`
			ControlKey string   `json:"controlKey"`
			Boundary   string   `json:"boundary"`
			Assertion  string   `json:"assertion"`
		} `json:"actionEvidence"`
	}
	if err := decodeExactJSON(ledgerSource, &ledger); err != nil {
		return Catalog{}, fmt.Errorf("decode behavior ledger: %w", err)
	}
	if len(routes) != 86 ||
		len(ledger.ReactScope.SurfaceIDs) != len(routes) {
		return Catalog{}, fmt.Errorf(
			"route and surface catalogs must contain the same exact 86 entries",
		)
	}

	catalog := Catalog{
		Routes: make([]RouteCoverage, len(routes)),
	}
	seenRoutes := make(map[string]bool, len(routes))
	seenSurfaces := make(map[string]bool, len(routes))
	for index, source := range routes {
		role, ok := canonicalRole(source.Role)
		surfaceID := strings.TrimSpace(ledger.ReactScope.SurfaceIDs[index])
		if !ok ||
			strings.TrimSpace(source.AuditID) == "" ||
			strings.TrimSpace(source.ScreenName) == "" ||
			surfaceID == "" ||
			seenRoutes[source.AuditID] ||
			seenSurfaces[surfaceID] {
			return Catalog{}, fmt.Errorf(
				"route catalog contains an incomplete or duplicate entry",
			)
		}
		seenRoutes[source.AuditID] = true
		seenSurfaces[surfaceID] = true
		catalog.Routes[index] = RouteCoverage{
			AuditID:    strings.TrimSpace(source.AuditID),
			SurfaceID:  surfaceID,
			Role:       role,
			ScreenName: strings.TrimSpace(source.ScreenName),
		}
	}

	executableAssertions := map[string]bool{
		"assertNativeFormControlOutcome": true,
		"assertAccessibleStateOutcome":   true,
		"assertControlledSurfaceOutcome": true,
		"assertDurableControlOutcome":    true,
		"suggestedFilename":              true,
	}
	for _, evidence := range ledger.ActionEvidence {
		if evidence.Scope != "route" ||
			!executableAssertions[evidence.Assertion] ||
			(len(evidence.Profiles) > 0 &&
				!slices.Contains(evidence.Profiles, "mock")) {
			continue
		}
		if !seenSurfaces[evidence.SurfaceID] ||
			strings.TrimSpace(evidence.ControlKey) == "" ||
			strings.TrimSpace(evidence.Boundary) == "" {
			return Catalog{}, fmt.Errorf(
				"executable action references an invalid route or boundary",
			)
		}
		catalog.Actions = append(catalog.Actions, ActionCoverage{
			ActionID:   evidence.SurfaceID + "|" + evidence.ControlKey,
			SurfaceID:  evidence.SurfaceID,
			ControlKey: evidence.ControlKey,
			Boundary:   evidence.Boundary,
		})
	}
	sort.Slice(catalog.Actions, func(left, right int) bool {
		return catalog.Actions[left].ActionID <
			catalog.Actions[right].ActionID
	})
	if len(catalog.Actions) != 306 {
		return Catalog{}, fmt.Errorf(
			"executable visible-action catalog must contain exactly 306 entries",
		)
	}
	for index := 1; index < len(catalog.Actions); index++ {
		if catalog.Actions[index-1].ActionID ==
			catalog.Actions[index].ActionID {
			return Catalog{}, fmt.Errorf(
				"executable visible-action catalog contains a duplicate",
			)
		}
	}
	return catalog, nil
}

func decodeExactJSON(source []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(source)))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("JSON contains trailing data")
	}
	return nil
}

func canonicalRole(source string) (string, bool) {
	role, ok := map[string]string{
		"Global":             "admin",
		"CAA Inspector":      "inspector",
		"Lead Inspector":     "leadInspector",
		"Department Manager": "manager",
		"General Manager":    "gm",
		"Finance":            "finance",
		"Executive Director": "executiveDirector",
		"Auditee":            "auditee",
		"Admin Preview":      "admin",
	}[strings.TrimSpace(source)]
	return role, ok
}
