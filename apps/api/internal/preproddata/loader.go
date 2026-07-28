package preproddata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"
)

var ErrReconciliationMismatch = errors.New(
	"preprod reconciliation does not match immutable intent",
)

const maxCommandCheckpoints int64 = 1024
const maxCommandPayloadBytes = 1 << 20

type AuthoritativeCommand struct {
	Family      string `json:"family"`
	OperationID string `json:"operationId"`
	Payload     []byte `json:"payload"`
}

type Reconciliation struct {
	ActualCounts        map[string]int64  `json:"actualCounts"`
	RelationshipDigests map[string]string `json:"relationshipDigests"`
}

// CommandBoundary is implemented only by server-owned application services.
// Loader packages stream deterministic commands through this boundary and
// never mutate domain tables, append-only histories, or users directly.
type CommandBoundary interface {
	Preflight(context.Context, TargetFingerprint, Operation) error
	Apply(context.Context, AuthoritativeCommand) error
	Reconcile(context.Context) (Reconciliation, error)
}

type CommandStream interface {
	Next(context.Context) (AuthoritativeCommand, error)
}

type ResumableCommandStream interface {
	CommandStream
	ResumeAfter(context.Context, int64, string) error
}

type sliceCommandStream struct {
	commands []AuthoritativeCommand
	index    int
}

func NewSliceCommandStream(
	commands ...AuthoritativeCommand,
) CommandStream {
	return &sliceCommandStream{
		commands: append([]AuthoritativeCommand(nil), commands...),
	}
}

func (stream *sliceCommandStream) Next(
	_ context.Context,
) (AuthoritativeCommand, error) {
	if stream.index >= len(stream.commands) {
		return AuthoritativeCommand{}, io.EOF
	}
	command := stream.commands[stream.index]
	stream.index++
	return command, nil
}

func (stream *sliceCommandStream) ResumeAfter(
	_ context.Context,
	appliedCommands int64,
	lastOperationID string,
) error {
	if appliedCommands < 0 ||
		appliedCommands > int64(len(stream.commands)) {
		return fmt.Errorf("resume position is outside the command stream")
	}
	if appliedCommands == 0 {
		if lastOperationID != "" {
			return fmt.Errorf("zero resume position has an operation ID")
		}
		stream.index = 0
		return nil
	}
	if stream.commands[appliedCommands-1].OperationID != lastOperationID {
		return fmt.Errorf("resume operation does not match the command stream")
	}
	stream.index = int(appliedCommands)
	return nil
}

type RunInput struct {
	Intent        IntentManifest
	Authorization OperationAuthorization
	ControlStore  *FileControlStore
	Boundary      CommandBoundary
	Commands      CommandStream
	Clock         func() time.Time
}

func Run(ctx context.Context, input RunInput) (ResultManifest, error) {
	if input.ControlStore == nil ||
		input.Boundary == nil ||
		input.Commands == nil {
		return ResultManifest{}, fmt.Errorf(
			"control store, command boundary, and command stream are required",
		)
	}
	if input.Clock == nil {
		input.Clock = time.Now
	}
	now := input.Clock().UTC()

	// The canonical intent is durable before any target preflight or data write.
	if err := input.ControlStore.AppendIntent(input.Intent); err != nil {
		return ResultManifest{}, fmt.Errorf("append intent: %w", err)
	}
	if err := input.Authorization.Validate(input.Intent, now); err != nil {
		return ResultManifest{}, err
	}
	existing, err := input.ControlStore.SuccessfulResult(
		input.Intent.RunID,
		input.Intent.IntentDigest,
	)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNoSuccessfulResult) {
		return ResultManifest{}, fmt.Errorf("read completed run: %w", err)
	}
	if err := input.Boundary.Preflight(
		ctx,
		input.Intent.Target,
		input.Authorization.Operation,
	); err != nil {
		return recordFailure(
			input,
			now,
			nil,
			Reconciliation{},
			errors.New("TARGET_PREFLIGHT_FAILED"),
		)
	}

	checkpoints, err := input.ControlStore.RunCheckpoints(
		input.Intent.RunID,
		input.Intent.IntentDigest,
	)
	if errors.Is(err, ErrNoCheckpoints) {
		checkpoints = nil
	} else if err != nil {
		return ResultManifest{}, fmt.Errorf("read run checkpoints: %w", err)
	}
	checkpointNames := make([]string, 0, len(checkpoints)+2)
	for _, checkpoint := range checkpoints {
		checkpointNames = append(checkpointNames, checkpoint.Name)
	}
	var checkpointSequence int64
	var appliedCommands int64
	var lastOperationID string
	if len(checkpoints) > 0 {
		latest := checkpoints[len(checkpoints)-1]
		checkpointSequence = latest.Sequence
		appliedCommands = latest.AppliedCommands
		lastOperationID = latest.LastOperationID
	}
	switch input.Authorization.Operation {
	case LoadEmptyTarget:
		if len(checkpoints) != 0 {
			return ResultManifest{}, errors.New(
				"LOAD_EMPTY_TARGET_REQUIRES_NEW_RUN_ID",
			)
		}
	case ResumeRun:
		if len(checkpoints) == 0 {
			return ResultManifest{}, errors.New(
				"RESUME_RUN_REQUIRES_DURABLE_CHECKPOINT",
			)
		}
		resumable, ok := input.Commands.(ResumableCommandStream)
		if !ok {
			return ResultManifest{}, errors.New(
				"RESUME_RUN_REQUIRES_RESUMABLE_COMMAND_STREAM",
			)
		}
		if err := resumable.ResumeAfter(
			ctx,
			appliedCommands,
			lastOperationID,
		); err != nil {
			return ResultManifest{}, errors.New(
				"RESUME_RUN_STREAM_POSITION_MISMATCH",
			)
		}
	case DropRecreateTarget:
		if len(checkpoints) != 0 {
			return ResultManifest{}, errors.New(
				"DROP_RECREATE_TARGET_REQUIRES_NEW_RUN_ID",
			)
		}
	}
	if err := input.ControlStore.ConsumeAuthorization(
		input.Authorization,
		now,
	); err != nil {
		return ResultManifest{}, err
	}

	checkpointSequence++
	authorizationCheckpointName := fmt.Sprintf(
		"%s-authorization-consumed-%012d",
		strings.ToLower(string(input.Authorization.Operation)),
		checkpointSequence,
	)
	checkpointNames = append(
		checkpointNames,
		authorizationCheckpointName,
	)
	if err := input.ControlStore.AppendCheckpoint(Checkpoint{
		SchemaVersion: "preprod-run-checkpoint/v1",
		RunID:         input.Intent.RunID, IntentDigest: input.Intent.IntentDigest,
		Sequence: checkpointSequence, Name: authorizationCheckpointName,
		AppliedCommands: appliedCommands, LastOperationID: lastOperationID,
		RecordedAt: now,
	}); err != nil {
		return ResultManifest{}, err
	}

	maxCommands, checkpointInterval := commandBounds(input.Intent)
	if appliedCommands > maxCommands {
		return ResultManifest{}, errors.New(
			"DURABLE_CHECKPOINT_EXCEEDS_IMMUTABLE_INTENT_BOUND",
		)
	}
	lastCheckpointCount := appliedCommands
	for {
		command, err := input.Commands.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return recordFailure(
				input,
				input.Clock().UTC(),
				checkpointNames,
				Reconciliation{},
				errors.New("COMMAND_STREAM_FAILED"),
			)
		}
		if appliedCommands >= maxCommands {
			return recordFailure(
				input,
				input.Clock().UTC(),
				checkpointNames,
				Reconciliation{},
				fmt.Errorf(
					"authoritative command stream exceeds immutable intent bound",
				),
			)
		}
		if err := validateAuthoritativeCommand(command); err != nil {
			return recordFailure(
				input,
				input.Clock().UTC(),
				checkpointNames,
				Reconciliation{},
				err,
			)
		}
		if err := input.Boundary.Apply(ctx, command); err != nil {
			commandErr := fmt.Errorf(
				"AUTHORITATIVE_COMMAND_FAILED family=%s",
				command.Family,
			)
			return recordFailure(
				input,
				input.Clock().UTC(),
				checkpointNames,
				Reconciliation{},
				commandErr,
			)
		}
		appliedCommands++
		lastOperationID = command.OperationID
		if appliedCommands%checkpointInterval == 0 {
			checkpointSequence++
			name := commandCheckpointName(
				appliedCommands,
				lastOperationID,
			)
			checkpointNames = append(checkpointNames, name)
			if err := input.ControlStore.AppendCheckpoint(Checkpoint{
				SchemaVersion: "preprod-run-checkpoint/v1",
				RunID:         input.Intent.RunID,
				IntentDigest:  input.Intent.IntentDigest,
				Sequence:      checkpointSequence, Name: name,
				AppliedCommands: appliedCommands,
				LastOperationID: lastOperationID,
				RecordedAt:      input.Clock().UTC(),
			}); err != nil {
				return ResultManifest{}, err
			}
			lastCheckpointCount = appliedCommands
		}
	}
	if appliedCommands > lastCheckpointCount {
		checkpointSequence++
		name := commandCheckpointName(appliedCommands, lastOperationID)
		checkpointNames = append(checkpointNames, name)
		if err := input.ControlStore.AppendCheckpoint(Checkpoint{
			SchemaVersion:   "preprod-run-checkpoint/v1",
			RunID:           input.Intent.RunID,
			IntentDigest:    input.Intent.IntentDigest,
			Sequence:        checkpointSequence,
			Name:            name,
			AppliedCommands: appliedCommands,
			LastOperationID: lastOperationID,
			RecordedAt:      input.Clock().UTC(),
		}); err != nil {
			return ResultManifest{}, err
		}
	}
	reconciliation, err := input.Boundary.Reconcile(ctx)
	if err != nil {
		return recordFailure(
			input,
			input.Clock().UTC(),
			checkpointNames,
			Reconciliation{},
			errors.New("TARGET_RECONCILIATION_FAILED"),
		)
	}
	if err := validateReconciliation(input.Intent, reconciliation); err != nil {
		return recordFailure(
			input,
			input.Clock().UTC(),
			checkpointNames,
			reconciliation,
			err,
		)
	}
	result, err := BuildResult(ResultInput{
		RunID: input.Intent.RunID, IntentDigest: input.Intent.IntentDigest,
		ActualCounts:        reconciliation.ActualCounts,
		RelationshipDigests: reconciliation.RelationshipDigests,
		Checkpoints:         checkpointNames, CompletedAt: input.Clock().UTC(),
	})
	if err != nil {
		return ResultManifest{}, err
	}
	if err := input.ControlStore.AppendResult(result); err != nil {
		return ResultManifest{}, err
	}
	return result, nil
}

func commandBounds(intent IntentManifest) (int64, int64) {
	var maximum int64
	for _, count := range intent.ExpectedCounts {
		maximum += count
	}
	interval := (maximum + maxCommandCheckpoints - 2) /
		(maxCommandCheckpoints - 1)
	if interval < 1 {
		interval = 1
	}
	return maximum, interval
}

func commandCheckpointName(
	appliedCommands int64,
	operationID string,
) string {
	return fmt.Sprintf(
		"authoritative-commands-applied-%012d-through-%s",
		appliedCommands,
		operationID,
	)
}

func recordFailure(
	input RunInput,
	completedAt time.Time,
	checkpoints []string,
	reconciliation Reconciliation,
	failure error,
) (ResultManifest, error) {
	result, buildErr := BuildResult(ResultInput{
		RunID: input.Intent.RunID, IntentDigest: input.Intent.IntentDigest,
		ActualCounts:        reconciliation.ActualCounts,
		RelationshipDigests: reconciliation.RelationshipDigests,
		Checkpoints:         checkpoints, Failures: []string{failure.Error()},
		CompletedAt: completedAt,
	})
	if buildErr != nil {
		return ResultManifest{}, errors.Join(
			failure,
			fmt.Errorf("build failed run result: %w", buildErr),
		)
	}
	if err := input.ControlStore.AppendResult(result); err != nil {
		return result, errors.Join(
			failure,
			fmt.Errorf("append failed run result: %w", err),
		)
	}
	return result, failure
}

func validateReconciliation(
	intent IntentManifest,
	reconciliation Reconciliation,
) error {
	if !maps.Equal(intent.ExpectedCounts, reconciliation.ActualCounts) {
		return ErrReconciliationMismatch
	}
	if len(reconciliation.RelationshipDigests) != len(intent.ExpectedCounts) {
		return fmt.Errorf(
			"%w: relationship digest families differ",
			ErrReconciliationMismatch,
		)
	}
	for family := range intent.ExpectedCounts {
		if !digestPattern.MatchString(
			reconciliation.RelationshipDigests[family],
		) {
			return fmt.Errorf(
				"%w: missing relationship digest for %s",
				ErrReconciliationMismatch,
				family,
			)
		}
	}
	return nil
}

func validateAuthoritativeCommand(command AuthoritativeCommand) error {
	if strings.TrimSpace(command.Family) == "" ||
		strings.TrimSpace(command.OperationID) == "" ||
		len(command.Payload) == 0 ||
		len(command.Payload) > maxCommandPayloadBytes {
		return fmt.Errorf("authoritative command is incomplete")
	}
	var payload any
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return fmt.Errorf("authoritative command payload must be valid JSON")
	}
	if _, ok := payload.(map[string]any); !ok {
		return fmt.Errorf("authoritative command payload must be a JSON object")
	}
	if containsForbiddenSecretField(payload) {
		return fmt.Errorf(
			"authoritative command contains forbidden secret field",
		)
	}
	return nil
}

func containsForbiddenSecretField(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.Map(func(character rune) rune {
				switch character {
				case '-', '_', '.', ' ':
					return -1
				default:
					return character
				}
			}, strings.ToLower(key))
			switch normalized {
			case "password", "totpsecret", "recoverycode",
				"provideractiontoken", "accesstoken", "refreshtoken",
				"privatekey", "apikey", "clientsecret":
				return true
			}
			if containsForbiddenSecretField(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsForbiddenSecretField(nested) {
				return true
			}
		}
	}
	return false
}
