import { createHash } from "node:crypto";
import { readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";

export const REQUIRED_SUBJECTS = Object.freeze([
  "cloudflared",
  "gateway",
  "api",
  "worker",
  "keycloak",
  "database-bootstrap",
  "migration",
]);

export const REQUIRED_BINDINGS = Object.freeze([
  "compose",
  "imageLock",
  "gatewayConfig",
  "systemdUnit",
  "tunnelSystemdUnit",
  "tunnelHealthSystemdUnit",
  "tunnelHealthTimer",
  "supervisor",
  "ipv6Preflight",
  "appEntrypoint",
  "databaseBootstrap",
  "keycloakEntrypoint",
  "keycloakRealm",
  "migrationSet",
  "decisionContract",
  "infrastructurePolicy",
  "rdsCABundle",
  "runtimeEnvironment",
  "gatewayStaticArtifact",
]);

const RUNTIME_IMAGE_ENV = Object.freeze({
  cloudflared: "AVIA_CLOUDFLARED_IMAGE",
  gateway: "AVIA_GATEWAY_IMAGE",
  api: "AVIA_API_IMAGE",
  worker: "AVIA_WORKER_IMAGE",
  keycloak: "AVIA_KEYCLOAK_IMAGE",
  "database-bootstrap": "AVIA_DATABASE_BOOTSTRAP_IMAGE",
  migration: "AVIA_MIGRATION_IMAGE",
});

const REQUIRED_RUNTIME_FILE_ENV = Object.freeze([
  "AVIA_CLOUDFLARE_TUNNEL_TOKEN_FILE",
  "AVIA_APP_DATABASE_PASSWORD_FILE",
  "AVIA_APP_MIGRATION_PASSWORD_FILE",
  "AVIA_DATABASE_BOOTSTRAP_PASSWORD_FILE",
  "AVIA_KEYCLOAK_DATABASE_PASSWORD_FILE",
  "AVIA_OIDC_CLIENT_SECRET_FILE",
  "AVIA_SESSION_ENCRYPTION_KEY_FILE",
  "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE",
  "AVIA_APP_SMTP_PASSWORD_FILE",
  "AVIA_KEYCLOAK_SMTP_PASSWORD_FILE",
  "AVIA_KEYCLOAK_REALM_FILE",
]);

const digestPattern = /^sha256:[0-9a-f]{64}$/u;
const imagePattern = /^.+@sha256:[0-9a-f]{64}$/u;

function sha256(value) {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

export function digestPath(targetPath) {
  const stats = statSync(targetPath);
  if (stats.isFile()) return sha256(readFileSync(targetPath));
  if (!stats.isDirectory()) throw new Error("binding target must be a regular file or directory");

  const entries = [];
  const walk = (directory) => {
    const children = readFileDirectory(directory);
    for (const name of children) {
      const child = path.join(directory, name);
      const childStats = statSync(child);
      if (childStats.isDirectory()) walk(child);
      else if (childStats.isFile()) {
        entries.push(`${path.relative(targetPath, child)}\0${sha256(readFileSync(child))}`);
      }
    }
  };
  walk(targetPath);
  return sha256(entries.sort().join("\n"));
}

function readFileDirectory(directory) {
  return readdirSync(directory).sort();
}

function arnBelongsTo(value, { service, region, accountId, resourcePrefix }) {
  if (typeof value !== "string") return false;
  const parts = value.split(":");
  return parts.length >= 6
    && parts[0] === "arn"
    && /^aws(?:-[a-z0-9]+)*$/u.test(parts[1])
    && parts[2] === service
    && parts[3] === region
    && parts[4] === accountId
    && parts.slice(5).join(":").startsWith(resourcePrefix);
}

export function validateReleaseManifest(manifest, options = {}) {
  const errors = [];
  const now = options.now ?? Date.now();
  const root = options.root;
  const add = (code) => errors.push(code);

  if (!manifest || typeof manifest !== "object" || Array.isArray(manifest)) return ["invalid-manifest"];
  if (manifest.schemaVersion !== 1) add("unsupported-schema-version");
  if (!/^[a-z0-9][a-z0-9._-]{7,80}$/u.test(manifest.releaseId ?? "")) add("invalid-release-id");
  if (manifest.artifactStatus !== "candidate-only" || manifest.releasePending !== true || manifest.productionReady !== false) add("invalid-evidence-labels");
  if (manifest.architecture !== "linux/arm64") add("arm64-required");
  if (!digestPattern.test(manifest.sourceTreeSha256 ?? "")) add("missing-source-tree-digest");

  const issued = Date.parse(manifest.issuedAt ?? "");
  const expires = Date.parse(manifest.expiresAt ?? "");
  if (!Number.isFinite(issued) || !Number.isFinite(expires) || issued > now || expires <= now || expires - issued > 31 * 86400000) add("stale-release-input");

  const subjects = manifest.subjects ?? {};
  if (Object.keys(subjects).length !== REQUIRED_SUBJECTS.length || REQUIRED_SUBJECTS.some((name) => !(name in subjects))) add("unresolved-runtime-subjects");
  for (const name of REQUIRED_SUBJECTS) {
    const subject = subjects[name] ?? {};
    if (!imagePattern.test(subject.image ?? "")) add(`mutable-image:${name}`);
    if (subject.platform !== "linux/arm64") add(`wrong-platform:${name}`);
    for (const field of ["configDigest", "sbomSha256", "provenanceSha256", "vulnerabilityEvidenceSha256"]) {
      if (!digestPattern.test(subject[field] ?? "")) add(`missing-${field}:${name}`);
    }
  }

  const bindings = manifest.bindings ?? {};
  if (Object.keys(bindings).length !== REQUIRED_BINDINGS.length || REQUIRED_BINDINGS.some((name) => !(name in bindings))) add("invalid-release-bindings");
  for (const [name, binding] of Object.entries(bindings)) {
    if (!binding || typeof binding !== "object" || path.isAbsolute(binding.path ?? "") || (binding.path ?? "").split(/[\\/]/u).includes("..") || !digestPattern.test(binding.sha256 ?? "")) {
      add(`invalid-binding:${name}`);
      continue;
    }
    if (root) {
      try {
        if (digestPath(path.join(root, binding.path)) !== binding.sha256) add(`changed-binding:${name}`);
      } catch {
        add(`missing-binding:${name}`);
      }
    }
  }

  if (manifest.database?.latestMigrationVersion !== 44 || manifest.database?.forwardOnly !== true || manifest.database?.runtimeRole !== "aviasurveil360_runtime" || manifest.database?.migrationRole !== "aviasurveil360_migrator" || manifest.database?.migrationRoleLockedAfterUse !== true) add("invalid-migration-contract");
  if (!digestPattern.test(manifest.rollback?.predecessorManifestSha256 ?? "") || !digestPattern.test(manifest.rollback?.compatibilityEvidenceSha256 ?? "") || manifest.rollback?.incompatibleAction !== "roll-forward-or-coordinated-restore") add("invalid-rollback-contract");

  const targets = manifest.targets ?? {};
  if (targets.awsProfile !== "avia") add("aws-operator-profile-must-be-avia");
  if (!/^\d{12}$/u.test(targets.awsAccountId ?? "") || !/^[a-z]{2}(?:-gov)?-[a-z]+-\d$/u.test(targets.region ?? "") || !/^i-[0-9a-f]{17}$/u.test(targets.instanceId ?? "") || !/^arn:aws[^:]*:rds:/u.test(targets.databaseArn ?? "") || !Array.isArray(targets.bucketArns) || targets.bucketArns.length < 4 || targets.bucketArns.some((value) => !/^arn:aws[^:]*:s3:::/u.test(value)) || !/^[0-9a-f]{32}$/u.test(targets.cloudflareAccountId ?? "") || !/^[0-9a-f]{32}$/u.test(targets.cloudflareZoneId ?? "") || !/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u.test(targets.cloudflareTunnelId ?? "") || !/^arn:aws[^:]*:ssm:/u.test(targets.connectorParameterArn ?? "") || !/^[a-z0-9][a-z0-9.-]+[a-z0-9]$/u.test(targets.hostname ?? "") || !Array.isArray(targets.terraformAddresses) || targets.terraformAddresses.length < 1 || targets.terraformAddresses.some((value) => !/^[a-z0-9_]+\.[a-z0-9_]+(?:\[[^\]]+\])?$/u.test(value)) || !["retain", "destroy-pending-separate-authorization"].includes(targets.retainOrDestroyDecision)) add("unresolved-remote-targets");
  if (!arnBelongsTo(targets.databaseArn, { service: "rds", region: targets.region, accountId: targets.awsAccountId, resourcePrefix: "db:" })) add("database-target-mismatch");
  if (!arnBelongsTo(targets.connectorParameterArn, { service: "ssm", region: targets.region, accountId: targets.awsAccountId, resourcePrefix: "parameter/aviasurveil360/private-pilot/" })) add("connector-parameter-target-mismatch");
  const ecrPrefix = `${targets.awsAccountId}.dkr-ecr.${targets.region}.on.aws/`;
  for (const name of REQUIRED_SUBJECTS) {
    if (!String(subjects[name]?.image ?? "").startsWith(ecrPrefix)) add(`non-target-ecr-image:${name}`);
  }

  const authorization = manifest.authorization ?? {};
  if (authorization.scope !== "local-preparation-only" || authorization.remoteActionsAuthorized !== false || authorization.productionReleaseAuthorized !== false) add("remote-authority-forbidden");

  if (root && bindings.imageLock) {
    try {
      const lock = JSON.parse(readFileSync(path.join(root, bindings.imageLock.path), "utf8"));
      if (lock.resolved !== true || lock.architecture !== "linux/arm64") add("unresolved-image-lock");
      for (const name of REQUIRED_SUBJECTS) {
        if (lock.subjects?.[name] !== subjects[name]?.image) add(`image-lock-mismatch:${name}`);
      }
    } catch {
      add("invalid-image-lock");
    }
  }

  if (root && bindings.keycloakRealm) {
    try {
      const realm = JSON.parse(readFileSync(path.join(root, bindings.keycloakRealm.path), "utf8"));
      const smtp = realm.smtpServer ?? {};
      const encrypted = (smtp.starttls === "true" && smtp.ssl !== "true") || (smtp.ssl === "true" && smtp.starttls !== "true");
      if (!realm.realm || realm.enabled !== true || !smtp.host || !["465", "587"].includes(String(smtp.port)) || smtp.auth !== "true" || smtp.password !== "${KC_SMTP_PASSWORD}" || !smtp.user || !smtp.from || !encrypted) add("insecure-keycloak-smtp-contract");
    } catch {
      add("invalid-keycloak-realm");
    }
  }

  if (root && bindings.rdsCABundle) {
    try {
      const bundle = readFileSync(path.join(root, bindings.rdsCABundle.path), "utf8");
      if (!bundle.includes("-----BEGIN CERTIFICATE-----") || !bundle.includes("-----END CERTIFICATE-----") || bundle.includes("PRIVATE KEY")) add("invalid-rds-ca-bundle");
    } catch {
      add("invalid-rds-ca-bundle");
    }
  }

  if (root && bindings.runtimeEnvironment) {
    try {
      const source = readFileSync(path.join(root, bindings.runtimeEnvironment.path), "utf8");
      const environment = new Map();
      for (const rawLine of source.split(/\r?\n/u)) {
        const line = rawLine.trim();
        if (line === "" || line.startsWith("#")) continue;
        const match = /^([A-Z][A-Z0-9_]*)=([^\s"']+)$/u.exec(line);
        if (!match || environment.has(match[1])) {
          add("invalid-runtime-environment");
          continue;
        }
        environment.set(match[1], match[2]);
      }
      for (const forbidden of [
        "AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_SHARED_CREDENTIALS_FILE", "AWS_CONFIG_FILE",
        "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
        "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_ROLE_ARN", "AWS_ROLE_SESSION_NAME",
        "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI",
        "AWS_CONTAINER_AUTHORIZATION_TOKEN", "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE",
        "AWS_EC2_METADATA_SERVICE_ENDPOINT", "AWS_EC2_METADATA_SERVICE_ENDPOINT_MODE",
        "AWS_ENDPOINT_URL", "AWS_ENDPOINT_URL_S3",
        "AVIA_OBJECT_STORE_ACCESS_KEY", "AVIA_OBJECT_STORE_SECRET_KEY",
        "AVIA_DATABASE_URL", "AVIA_OIDC_CLIENT_SECRET", "AVIA_SESSION_ENCRYPTION_KEY",
        "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET", "AVIA_SMTP_PASSWORD",
        "AVIA_DATABASE_BOOTSTRAP_PASSWORD", "AVIA_APP_DATABASE_PASSWORD",
        "AVIA_APP_MIGRATION_PASSWORD", "AVIA_KEYCLOAK_DATABASE_PASSWORD",
        "AVIA_KEYCLOAK_SMTP_PASSWORD", "AVIA_CLOUDFLARE_TUNNEL_TOKEN",
        "TUNNEL_TOKEN", "KC_DB_PASSWORD", "KC_SMTP_PASSWORD",
      ]) {
        if (environment.has(forbidden)) add("runtime-aws-profile-or-static-credential-forbidden");
      }
      for (const [name, envName] of Object.entries(RUNTIME_IMAGE_ENV)) {
        if (environment.get(envName) !== subjects[name]?.image) add(`runtime-image-mismatch:${name}`);
      }
      for (const envName of REQUIRED_RUNTIME_FILE_ENV) {
        if (!/^\/run\/aviasurveil360-private-pilot\/secrets\/[A-Za-z0-9._-]+$/u.test(environment.get(envName) ?? "")) add(`invalid-runtime-file-reference:${envName}`);
      }
      if (environment.get("AVIA_AWS_ACCOUNT_ID") !== targets.awsAccountId || environment.get("AVIA_AWS_REGION") !== targets.region || environment.get("AVIA_INSTANCE_ID") !== targets.instanceId) add("runtime-target-mismatch");
      const connectorParameterName = String(targets.connectorParameterArn ?? "").split(":parameter")[1] ?? "";
      if (environment.get("AVIA_CLOUDFLARE_TUNNEL_TOKEN_PARAMETER_NAME") !== connectorParameterName || environment.get("AVIA_CLOUDFLARE_EDGE_IP_VERSION") !== "6" || environment.get("AVIA_FORCE_IPV6") !== "true" || !/^[A-Za-z0-9.-]+(?:,[A-Za-z0-9.-]+)+$/u.test(environment.get("AVIA_CLOUDFLARE_EDGE_HOSTS") ?? "")) add("invalid-ipv6-tunnel-runtime-contract");
      if (!/^\/opt\/aviasurveil360\/private-pilot\/[A-Za-z0-9/._-]+$/u.test(environment.get("AVIA_RDS_CA_BUNDLE_FILE") ?? "")) add("invalid-runtime-file-reference:AVIA_RDS_CA_BUNDLE_FILE");
    } catch {
      add("invalid-runtime-environment");
    }
  }

  return [...new Set(errors)];
}

export function releaseManifestDigest(raw) {
  return sha256(raw);
}
