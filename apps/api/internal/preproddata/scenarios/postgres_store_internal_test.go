package scenarios

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

func TestRelationshipDigestAccumulatorMatchesCanonicalDigestWithoutRetention(
	t *testing.T,
) {
	tuples := [][]string{
		{"synthetic-z", "organization-2", "3"},
		{"synthetic-a", "organization-1", "1"},
		{"synthetic-m", "organization-1", "2"},
	}
	accumulator := newRelationshipDigestAccumulator()
	for _, tuple := range [][]string{tuples[1], tuples[2], tuples[0]} {
		accumulator.Add(tuple)
	}
	if got, expected := accumulator.Digest(), relationshipDigest(tuples); got != expected {
		t.Fatalf("streaming digest = %s, expected %s", got, expected)
	}

	accumulatorType := reflect.TypeOf(*accumulator)
	for index := 0; index < accumulatorType.NumField(); index++ {
		field := accumulatorType.Field(index)
		if field.Type.Kind() == reflect.Slice ||
			field.Type.Kind() == reflect.Map {
			t.Fatalf(
				"relationship accumulator retains unbounded %s field %s",
				field.Type.Kind(),
				field.Name,
			)
		}
	}
}

func TestStressObjectPayloadSizesSumToTheExactFrozenEightGiB(
	t *testing.T,
) {
	profile, err := profiles.Lookup("stress", "1.0.0")
	if err != nil {
		t.Fatalf("lookup stress: %v", err)
	}
	count := profile.ExpectedCounts["objectVersions"]
	base := profile.ResourceEnvelope.ObjectBytes / count
	remainder := profile.ResourceEnvelope.ObjectBytes % count
	if got := objectPayloadSize(profile, 0); got != base+1 {
		t.Fatalf("first stress object size = %d, expected %d", got, base+1)
	}
	if got := objectPayloadSize(profile, remainder); got != base {
		t.Fatalf("post-remainder stress object size = %d, expected %d", got, base)
	}
	total := (base+1)*remainder + base*(count-remainder)
	if total != profile.ResourceEnvelope.ObjectBytes ||
		total != int64(8*1024*1024*1024) {
		t.Fatalf("stress object payload total = %d", total)
	}

	record := Record{
		RecordID:       "synthetic-objectversions-0001",
		OrganizationID: "AUDITEE-A",
	}
	content, _ := safeSyntheticObjectContent(
		record,
		"synthetic-objects-0001",
		objectPayloadSize(profile, 0),
	)
	if int64(len(content)) != base+1 ||
		!strings.Contains(string(content), `"padding":"SSSS`) {
		t.Fatalf("bounded stress object content size = %d", len(content))
	}

	smoke, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke: %v", err)
	}
	if got := objectPayloadSize(smoke, 0); got != 0 {
		t.Fatalf("smoke object payload target = %d, expected safe JSON size", got)
	}
}

func TestLocalStressQualificationRetainsAnExactBoundedObjectPayload(
	t *testing.T,
) {
	profile, err := profiles.Lookup("stress", "1.1.0")
	if err != nil {
		t.Fatalf("lookup local stress qualification: %v", err)
	}
	count := profile.ExpectedCounts["objectVersions"]
	base := profile.ResourceEnvelope.ObjectBytes / count
	remainder := profile.ResourceEnvelope.ObjectBytes % count
	if got := objectPayloadSize(profile, 0); got != base+1 {
		t.Fatalf(
			"first local stress object size = %d, expected %d",
			got,
			base+1,
		)
	}
	if got := objectPayloadSize(profile, remainder); got != base {
		t.Fatalf(
			"post-remainder local stress object size = %d, expected %d",
			got,
			base,
		)
	}
	total := (base+1)*remainder + base*(count-remainder)
	if total != int64(512*1024*1024) {
		t.Fatalf("local stress object payload total = %d", total)
	}
}

func TestAssignmentRelationshipsRemainUniqueAtEveryFrozenProfileScale(
	t *testing.T,
) {
	for _, item := range []struct {
		name    string
		version string
	}{
		{"smoke", "1.0.0"},
		{"acceptance", "1.0.0"},
		{"realistic", "1.0.0"},
		{"stress", "1.0.0"},
		{"realistic", "1.1.0"},
		{"stress", "1.1.0"},
	} {
		t.Run(item.name+"@"+item.version, func(t *testing.T) {
			profile, err := profiles.Lookup(item.name, item.version)
			if err != nil {
				t.Fatalf("lookup profile: %v", err)
			}
			generator, err := preproddata.NewGenerator(
				profile,
				[]byte("task-8-assignment-relationship-test"),
			)
			if err != nil {
				t.Fatalf("new generator: %v", err)
			}
			stream := &Stream{
				profile:   profile,
				generator: generator,
			}
			count := profile.ExpectedCounts["assignments"]
			relationships := make(map[string]struct{}, count)
			for index := int64(0); index < count; index++ {
				record := stream.record("assignments", index)
				key := strings.Join([]string{
					attrString(record, "auditId"),
					attrString(record, "questionId"),
					attrString(record, "membershipId"),
				}, "\x00")
				if _, exists := relationships[key]; exists {
					t.Fatalf(
						"duplicate assignment relationship at index %d",
						index,
					)
				}
				relationships[key] = struct{}{}
			}
			if int64(len(relationships)) != count {
				t.Fatalf(
					"unique assignment relationships = %d, expected %d",
					len(relationships),
					count,
				)
			}
		})
	}
}

func TestChecklistResponseRelationshipsRemainUniqueAtEveryFrozenProfileScale(
	t *testing.T,
) {
	for _, item := range []struct {
		name    string
		version string
	}{
		{"smoke", "1.0.0"},
		{"acceptance", "1.0.0"},
		{"realistic", "1.0.0"},
		{"stress", "1.0.0"},
		{"realistic", "1.1.0"},
		{"stress", "1.1.0"},
	} {
		t.Run(item.name+"@"+item.version, func(t *testing.T) {
			profile, err := profiles.Lookup(item.name, item.version)
			if err != nil {
				t.Fatalf("lookup profile: %v", err)
			}
			generator, err := preproddata.NewGenerator(
				profile,
				[]byte("task-8-checklist-response-relationship-test"),
			)
			if err != nil {
				t.Fatalf("new generator: %v", err)
			}
			stream := &Stream{
				profile:   profile,
				generator: generator,
			}
			count := profile.ExpectedCounts["checklistResponses"]
			relationships := make(map[string]struct{}, count)
			for index := int64(0); index < count; index++ {
				record := stream.record("checklistResponses", index)
				key := strings.Join([]string{
					attrString(record, "auditId"),
					attrString(record, "questionId"),
				}, "\x00")
				if _, exists := relationships[key]; exists {
					t.Fatalf(
						"duplicate checklist response relationship at index %d",
						index,
					)
				}
				relationships[key] = struct{}{}
			}
			if int64(len(relationships)) != count {
				t.Fatalf(
					"unique checklist response relationships = %d, expected %d",
					len(relationships),
					count,
				)
			}
		})
	}
}
