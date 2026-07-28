package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

func TestRunConfigurationCarriesAuthorizationByFileOnly(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "loader-config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "environment": "local-preprod",
  "profile": "smoke",
  "profileVersion": "1.0.0",
  "seedFile": "/run/secrets/preprod_seed",
  "authorizationFile": "/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory": "/var/lib/aviasurveil360-preprod-control",
  "intentFile": "/var/lib/aviasurveil360-preprod-control/intents/run-task6.json"
}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	configuration, err := loadRunConfiguration(configPath)
	if err != nil {
		t.Fatalf("load run configuration: %v", err)
	}
	if configuration.AuthorizationFile !=
		"/run/secrets/preprod_loader_authorization" {
		t.Fatalf("authorization file = %q", configuration.AuthorizationFile)
	}
	if configuration.ControlStoreDirectory !=
		"/var/lib/aviasurveil360-preprod-control" {
		t.Fatalf("control store = %q", configuration.ControlStoreDirectory)
	}
}

func TestRunConfigurationRejectsInlineAuthorizationAndWrongEnvironment(
	t *testing.T,
) {
	for name, body := range map[string]string{
		"inline token": `{
  "environment":"local-preprod",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/control",
  "intentFile":"/control/intent.json",
  "authorizationToken":"forbidden"
}`,
		"wrong environment": `{
  "environment":"production",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/control",
  "intentFile":"/control/intent.json"
}`,
	} {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "loader-config.json")
			if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			if _, err := loadRunConfiguration(configPath); err == nil {
				t.Fatalf("unsafe configuration was accepted")
			}
		})
	}
}

func TestReadIntentRejectsTrailingContent(t *testing.T) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	const runID = "run-task6-trailing-intent"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID:          runID,
		Profile:        profile,
		SeedHash:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CodeDigest:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ContractDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Target: preproddata.TargetFingerprint{
			Environment:              "local-preprod",
			DatabaseName:             "aviasurveil360_local_preprod",
			DatabaseOwner:            "aviasurveil360_preprod_loader",
			PostgresSystemIdentifier: "7421987349021349876",
			PostgresHost:             "preprod-postgres",
			PostgresPort:             5432,
			ComposeProject:           "aviasurveil360-local-preprod",
			KeycloakRealm:            "aviasurveil360-local-preprod",
			KeycloakDatabase:         "keycloak_local_preprod",
			KeycloakServiceClientID:  "aviasurveil360-local-preprod-lifecycle",
			MailpitNamespace:         "aviasurveil360-local-preprod",
			ObjectBucket:             "aviasurveil360-local-preprod",
			ObjectPrefix:             "runs/" + runID + "/",
			LoaderQueueNamespace:     "aviasurveil360-local-preprod",
			ProfileName:              "smoke",
			ProfileVersion:           "1.0.0",
			RunID:                    runID,
		},
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("encode intent: %v", err)
	}
	intentPath := filepath.Join(t.TempDir(), "intent.json")
	encoded = append(encoded, []byte("\n{\"unexpected\":true}")...)
	if err := os.WriteFile(intentPath, encoded, 0o600); err != nil {
		t.Fatalf("write intent: %v", err)
	}

	if _, err := readIntent(intentPath); err == nil {
		t.Fatalf("intent reader accepted trailing JSON content")
	}
}
