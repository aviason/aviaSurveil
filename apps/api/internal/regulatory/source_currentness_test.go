package regulatory

import "testing"

func TestSourceCurrentnessActivationRejectsIncompleteAndSelfTransitions(t *testing.T) {
	valid := SourceCurrentnessActivationCommand{
		OperationID:              "SOURCE-CURRENTNESS-TEST-VALID",
		IdempotencyKey:           "SOURCE-CURRENTNESS-TEST-VALID",
		CurrentSourceSnapshotID:  "SOURCE-V2",
		CurrentSourceHash:        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PreviousSourceSnapshotID: "SOURCE-V1",
		PreviousSourceHash:       "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Reason:                   "Exercise exact immutable source-currentness activation validation.",
	}
	if !validSourceCurrentnessActivation(valid) {
		t.Fatalf("complete predecessor/current activation was rejected: %+v", valid)
	}
	for name, mutate := range map[string]func(*SourceCurrentnessActivationCommand){
		"missing predecessor hash": func(command *SourceCurrentnessActivationCommand) {
			command.PreviousSourceHash = ""
		},
		"self transition": func(command *SourceCurrentnessActivationCommand) {
			command.PreviousSourceSnapshotID = command.CurrentSourceSnapshotID
		},
		"invalid current hash": func(command *SourceCurrentnessActivationCommand) {
			command.CurrentSourceHash = "not-a-sha256"
		},
	} {
		t.Run(name, func(t *testing.T) {
			command := valid
			mutate(&command)
			if validSourceCurrentnessActivation(command) {
				t.Fatalf("invalid source-currentness activation was accepted: %+v", command)
			}
		})
	}
}
