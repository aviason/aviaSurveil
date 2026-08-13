import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const recoveryRoot = path.join(repositoryRoot, "deploy/recovery");
const localComposePath = path.join(repositoryRoot, "deploy/local/compose.yaml");
const recoveryComposePath = path.join(
  recoveryRoot,
  "compose.recovery.yaml",
);
const applicationConfigPath = path.join(
  recoveryRoot,
  "pgbackrest-application.conf",
);
const identityConfigPath = path.join(
  recoveryRoot,
  "pgbackrest-identity.conf",
);
const policyPath = path.join(recoveryRoot, "minio-backup-policy.json");

function readRequired(filePath) {
  assert.ok(
    existsSync(filePath),
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function readScript(name) {
  return readRequired(path.join(repositoryRoot, "scripts", name));
}

function composeConfig() {
  readRequired(recoveryComposePath);
  return JSON.parse(
    execFileSync(
      "docker",
      [
        "compose",
        "--file",
        localComposePath,
        "--file",
        recoveryComposePath,
        "--profile",
        "full",
        "--profile",
        "recovery",
        "config",
        "--format",
        "json",
      ],
      {
        cwd: repositoryRoot,
        encoding: "utf8",
        env: {
          ...process.env,
          AVIASURVEIL_LOCAL_STATE_DIR:
            "/private/tmp/aviasurveil360-backup-contract",
        },
      },
    ),
  );
}

function volumeSource(service, target) {
  return service.volumes.find((volume) => volume.target === target)?.source;
}

test("recovery profile isolates the immutable backup store from primary objects", () => {
  const compose = composeConfig();
  const primary = compose.services.minio;
  const backup = compose.services["backup-minio"];

  assert.ok(backup, "backup-minio service is required");
  assert.deepEqual(backup.profiles, ["recovery"]);
  assert.match(backup.image, /@sha256:[a-f0-9]{64}$/);
  assert.equal(backup.read_only, true);
  assert.match(String(backup.user), /^[1-9]\d*:[0-9]+$/);
  assert.equal(backup.ports, undefined);
  assert.ok(backup.healthcheck?.test);

  const primaryVolume = volumeSource(primary, "/data");
  const backupVolume = volumeSource(backup, "/data");
  assert.ok(primaryVolume);
  assert.ok(backupVolume);
  assert.notEqual(primaryVolume, backupVolume);
  assert.equal(backupVolume, "backup-object-store");

  assert.equal(compose.networks["recovery-backup"].internal, true);
  assert.ok(
    Object.hasOwn(backup.networks, "recovery-backup"),
    "backup MinIO must join only the internal recovery network",
  );
  assert.equal(primary.networks["recovery-backup"], undefined);
});

test("backup MinIO administration accepts only its reviewed self-signed TLS boundary", () => {
  const initializer = readRequired(
    path.join(recoveryRoot, "backup-minio-init.sh"),
  );
  const adminCommands = initializer
    .split("\n")
    .filter((line) => line.includes("mc admin "));
  assert.ok(adminCommands.length >= 6);
  for (const command of adminCommands) {
    assert.match(command, /mc admin --insecure /);
  }
});

test("object backup identity can apply retention only within its two backup buckets", () => {
  const initializer = readRequired(
    path.join(recoveryRoot, "backup-minio-init.sh"),
  );
  const policyMatch = initializer.match(
    /cat >\/tmp\/object-policy\.json <<'EOF'\n([\s\S]+?)\nEOF/,
  );
  assert.ok(policyMatch, "object backup IAM policy must be embedded");
  const policy = JSON.parse(policyMatch[1]);
  const objectStatement = policy.Statement.find(({ Resource }) =>
    Resource?.includes("arn:aws:s3:::application-object-backups/*"),
  );

  assert.ok(objectStatement);
  assert.ok(objectStatement.Action.includes("s3:PutObject"));
  assert.ok(objectStatement.Action.includes("s3:PutObjectRetention"));
  assert.deepEqual(objectStatement.Resource, [
    "arn:aws:s3:::application-object-backups/*",
    "arn:aws:s3:::recovery-catalog/*",
  ]);
  assert.doesNotMatch(JSON.stringify(policy), /"Action"\s*:\s*"\*"/);
  assert.doesNotMatch(JSON.stringify(policy), /"Resource"\s*:\s*"\*"/);
});

test("both PostgreSQL clusters use the reviewed pgBackRest runtime and separate stanzas", () => {
  const compose = composeConfig();
  const application = compose.services.postgres;
  const identity = compose.services["preprod-auth-postgres"];

  assert.equal(
    application.image,
    "aviasurveil360/postgres-recovery:local",
  );
  assert.equal(identity.image, application.image);
  assert.ok(Object.hasOwn(application.networks, "recovery-backup"));
  assert.ok(Object.hasOwn(identity.networks, "recovery-backup"));
  assert.match(application.command.join(" "), /stanza=application/);
  assert.match(identity.command.join(" "), /stanza=identity/);

  for (const [service, configName] of [
    [application, "pgbackrest_application_config"],
    [identity, "pgbackrest_identity_config"],
  ]) {
    assert.ok(
      service.configs.some(({ source }) => source === configName),
      `${configName} must be mounted`,
    );
    for (const secret of [
      "backup_pgbackrest_access_key",
      "backup_pgbackrest_secret_key",
      "backup_repository_cipher_passphrase",
    ]) {
      assert.ok(
        service.secrets.some(({ source }) => source === secret),
        `${secret} must be mounted`,
      );
    }
  }

  assert.equal(
    compose.configs.pgbackrest_application_config.file,
    applicationConfigPath,
  );
  assert.equal(
    compose.configs.pgbackrest_identity_config.file,
    identityConfigPath,
  );
  assert.equal(compose.configs.backup_policy.file, policyPath);
});

test("recovery runtime pins the pgBackRest and JSON tool package versions", () => {
  const dockerfile = readRequired(path.join(recoveryRoot, "Dockerfile"));
  assert.match(dockerfile, /pgbackrest=2\.55\.1-r0/);
  assert.match(dockerfile, /jq=1\.8\.1-r0/);
  for (const fixedPackage of [
    "libcrypto3=3.5.7-r0",
    "libssl3=3.5.7-r0",
    "libxml2=2.13.9-r1",
    "musl=1.2.5-r12",
    "musl-utils=1.2.5-r12",
    "zlib=1.3.2-r0",
  ]) {
    assert.match(dockerfile, new RegExp(fixedPackage.replaceAll(".", "\\.")));
  }
  assert.match(dockerfile, /rm -f \/usr\/local\/bin\/gosu/);
  assert.doesNotMatch(dockerfile, /apk\s+add[^\n]*\spgbackrest(?:\s|$)/);
});

test("recovery runtime participates in the digest-bound image evidence gate", () => {
  const buildScript = readScript("build-local-images.sh");
  const evidenceCheck = readScript("check-local-image-evidence.sh");
  const sbomScript = readScript("generate-image-sboms.sh");
  const scanScript = readScript("scan-local-images.sh");
  const harness = readScript("test-backup-profile.sh");

  assert.match(
    buildScript,
    /build_image postgres-recovery aviasurveil360\/postgres-recovery:local deploy\/recovery\/Dockerfile postgres-recovery/,
  );
  assert.match(buildScript, /POSTGRES_IMAGE=\$postgres_image/);
  assert.match(buildScript, /Built 8 local runtime images/);
  assert.match(sbomScript, /Generated 8 digest-bound CycloneDX SBOMs/);
  assert.match(
    scanScript,
    /All 8 local image digests passed the HIGH\/CRITICAL vulnerability gate/,
  );
  assert.match(
    evidenceCheck,
    /recovery\)\s+required_images="postgres-recovery"/,
  );
  assert.match(harness, /check-local-image-evidence\.sh" recovery/);
  assert.doesNotMatch(harness, /compose build postgres/);
});

test("pgBackRest stanzas use encrypted, retained, distinct S3 repositories without plaintext credentials", () => {
  const expected = [
    [
      applicationConfigPath,
      "application",
      "/var/lib/postgresql/data/pgdata",
      "application-database-backups",
      "aviasurveil360",
      "aviasurveil360",
    ],
    [
      identityConfigPath,
      "identity",
      "/var/lib/postgresql/data/pgdata",
      "identity-database-backups",
      "auth_preprod",
      "auth_local_preprod",
    ],
  ];

  for (const [configPath, stanza, pgPath, bucket, pgUser, pgDatabase] of expected) {
    const source = readRequired(configPath);
    assert.match(source, new RegExp(`^\\[${stanza}\\]$`, "m"));
    assert.match(source, new RegExp(`^pg1-path=${pgPath}$`, "m"));
    assert.match(source, new RegExp(`^pg1-user=${pgUser}$`, "m"));
    assert.match(source, new RegExp(`^pg1-database=${pgDatabase}$`, "m"));
    assert.match(source, /^repo1-type=s3$/m);
    assert.match(source, /^repo1-s3-endpoint=backup-minio$/m);
    assert.match(source, /^repo1-storage-port=9000$/m);
    assert.match(source, /^repo1-s3-uri-style=path$/m);
    assert.match(
      source,
      new RegExp(`^repo1-s3-bucket=${bucket}$`, "m"),
    );
    assert.match(source, /^repo1-cipher-type=aes-256-cbc$/m);
    assert.match(source, /^repo1-retention-full=2$/m);
    assert.match(source, /^repo1-retention-diff=4$/m);
    assert.doesNotMatch(source, /repo1-(?:s3-key|s3-key-secret|cipher-pass)=/);
    assert.doesNotMatch(source, /password\s*=/i);
  }

  assert.notEqual(
    readRequired(applicationConfigPath),
    readRequired(identityConfigPath),
  );
});

test("backup policy requires versioning, retention, exact manifests, and all recovery components", () => {
  const policy = JSON.parse(readRequired(policyPath));
  assert.equal(policy.schemaVersion, 1);
  assert.equal(policy.artifactStatus, "candidate-only");
  assert.equal(policy.failureDomain, "same-host-logically-isolated");
  assert.equal(policy.productionHostLossCovered, false);
  assert.deepEqual(policy.schedules, {
    full: "weekly",
    differential: "daily",
    incremental: "every-15-minutes",
  });
  assert.equal(policy.objectBackup.versioningRequired, true);
  assert.equal(policy.objectBackup.objectLock.enabled, true);
  assert.ok(policy.objectBackup.objectLock.retentionDays >= 7);
  assert.deepEqual(policy.objectBackup.manifestFields, [
    "sourceBucket",
    "key",
    "versionId",
    "etag",
    "sha256",
    "size",
    "metadata",
    "retentionUntil",
  ]);
  assert.deepEqual(policy.recoveryPoint.requiredComponents, [
    "applicationDatabase",
    "identityDatabase",
    "identityFingerprint",
    "applicationObjects",
    "configurationReferences",
  ]);
  assert.equal(policy.recoveryPoint.partialSuccessAllowed, false);
});

test("backup commands are locked, typed, secret-file based, and never mark partial success", () => {
  for (const [script, stanza] of [
    ["backup-postgres.sh", "application"],
    ["backup-identity-postgres.sh", "identity"],
  ]) {
    const source = readScript(script);
    assert.match(source, /acquire_recovery_lock/);
    assert.match(source, /full\|diff\|incr/);
    assert.match(
      source,
      new RegExp(`pgbackrest_command[\\s\\S]+?\\n\\s+${stanza}\\s+`),
    );
    assert.match(source, /stanza-create/);
    assert.match(source, /\bcheck\b/);
    assert.match(source, /\bbackup\b/);
    assert.doesNotMatch(source, /(?:PASSWORD|SECRET|CIPHER_PASS)=["'][^"$]/);
    assert.doesNotMatch(source, /status["']?\s*:\s*["']complete/);
  }

  const recoveryLibrary = readRequired(
    path.join(repositoryRoot, "scripts/lib/recovery-backup.sh"),
  );
  assert.match(recoveryLibrary, /pgbackrest-secret/);
  const secretWrapper = readRequired(
    path.join(recoveryRoot, "pgbackrest-secret"),
  );
  assert.match(secretWrapper, /PGBACKREST_REPO1_CIPHER_PASS/);
  assert.match(secretWrapper, /\/run\/secrets\//);
  assert.doesNotMatch(
    secretWrapper,
    /(?:PASSWORD|SECRET|CIPHER_PASS)=["'][^"$]/,
  );

  const objectBackup = readScript("backup-objects.sh");
  assert.match(objectBackup, /acquire_recovery_lock/);
  assert.match(objectBackup, /object-backup/);
  assert.match(objectBackup, /recovery-point/);
  assert.doesNotMatch(objectBackup, /mc\s+mirror/);
});

test("recovery shell library resolves its state directory under POSIX sh", () => {
  const libraryPath = path.join(
    repositoryRoot,
    "scripts/lib/recovery-backup.sh",
  );
  const expectedState = "/private/tmp/aviasurveil360-recovery-library-test";
  const output = execFileSync(
    "sh",
    [
      "-c",
      `. "${libraryPath}"; printf '%s' "$recovery_state_directory"`,
    ],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      env: {
        ...process.env,
        AVIASURVEIL_LOCAL_STATE_DIR: expectedState,
      },
    },
  );
  assert.equal(output, expectedState);
});

test("object backup enumerates exact versions and records checksum plus retention metadata", () => {
  const source = readRequired(
    path.join(repositoryRoot, "apps/api/cmd/object-backup/main.go"),
  );
  assert.match(source, /WithVersions:\s*true/);
  assert.match(source, /VersionID/);
  assert.match(source, /ETag/);
  assert.match(source, /sha256/);
  assert.match(source, /UserMetadata/);
  assert.match(source, /RetentionUntil/);
  assert.match(source, /sourceBucket/);
  assert.match(source, /applicationObjects/);
  assert.match(source, /candidate-only/);
});

test("application and identity fingerprints cover authoritative and MFA state", () => {
  const application = readRequired(
    path.join(repositoryRoot, "apps/api/cmd/recovery-fingerprint/main.go"),
  );
  for (const table of [
    "schema_migrations",
    "organizations",
    "inspections",
    "findings",
    "cap_revisions",
    "evidence_versions",
    "outbox_messages",
  ]) {
    assert.match(application, new RegExp(table));
  }
  assert.match(application, /sha256/);
  assert.match(application, /applicationDatabase/);

  const identity = readScript("identity-recovery-fingerprint.sh");
  for (const table of [
    "auth_identity.accounts",
    "auth_identity.identifiers",
    "auth_identity.application_authorities",
    "auth_identity.mfa_factors",
    "auth_identity.provider_sessions",
  ]) {
    assert.match(identity, new RegExp(table));
  }
  assert.match(identity, /mfaFactors/u);
  assert.match(identity, /adminReceipts/u);
  assert.match(identity, /sha256/);
});

test("catalog verification refuses incomplete, mutable, or unchecked recovery points", () => {
  const source = readScript("verify-backup-catalog.sh");
  for (const component of [
    "applicationDatabase",
    "identityDatabase",
    "identityFingerprint",
    "applicationObjects",
    "configurationReferences",
  ]) {
    assert.match(source, new RegExp(component));
  }
  assert.match(source, /sha256/);
  assert.match(source, /retentionUntil/);
  assert.match(source, /status.*complete/);
  assert.match(source, /partial.*refus/i);
  assert.match(source, /backup\.recovery_point\.age/);
  assert.doesNotMatch(source, /docker\s+(?:system\s+)?prune/);
});

test("no-argument catalog verification runs two isolated recovery points and cleans up", () => {
  const catalog = readScript("verify-backup-catalog.sh");
  const harness = readScript("test-backup-profile.sh");
  assert.match(catalog, /test-backup-profile\.sh/);
  assert.match(harness, /aviasurveil360-task-plan4-backup-/);
  assert.match(harness, /verify-backup-catalog\.sh.*--create/s);
  assert.match(harness, /\bfull\b/);
  assert.match(harness, /\bincr\b/);
  assert.match(harness, /controlled.*database/i);
  assert.match(harness, /controlled.*object/i);
  assert.match(harness, /applicationDatabase\.sha256/);
  assert.match(harness, /manifestSha256/);
  assert.match(harness, /assert_no_task_owned_residue/);
  assert.match(harness, /cleanup_status=\$\?/);
  assert.match(harness, /residue_status=0/);
  assert.doesNotMatch(harness, /^\s+status=0$/m);
  assert.doesNotMatch(harness, /docker\s+(?:system\s+)?prune/);
});
