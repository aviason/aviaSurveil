import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import {
  chmodSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const keycloakDirectory = path.dirname(fileURLToPath(import.meta.url));
const sourcePath = path.join(keycloakDirectory, "realm-source.json");
const builderPath = path.join(keycloakDirectory, "build-realm.mjs");

function loadJSON(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function webClient(realm) {
  return realm.clients.find(
    (candidate) => candidate.clientId === "aviasurveil360-web",
  );
}

function lifecycleClient(realm) {
  return realm.clients.find(
    (candidate) => candidate.clientId === "aviasurveil360-lifecycle",
  );
}

function userProfile(realm) {
  const providers =
    realm.components?.["org.keycloak.userprofile.UserProfileProvider"] ?? [];
  assert.equal(
    providers.length,
    1,
    "realm must define exactly one declarative user-profile provider",
  );
  assert.equal(providers[0].providerId, "declarative-user-profile");
  const serializedConfig =
    providers[0].config?.["kc.user.profile.config"]?.[0];
  assert.equal(typeof serializedConfig, "string");
  return JSON.parse(serializedConfig);
}

test("reviewed realm source enforces production OIDC and optional TOTP", () => {
  const realm = loadJSON(sourcePath);
  assert.equal(realm.realm, "aviasurveil360");
  assert.equal(realm.enabled, true);
  assert.equal(realm.registrationAllowed, false);
  assert.equal(realm.resetPasswordAllowed, false);
  assert.equal(realm.sslRequired, "external");
  assert.equal(realm.eventsEnabled, true);
  assert.deepEqual(realm.eventsListeners, ["jboss-logging"]);
  assert.equal(realm.adminEventsEnabled, true);
  assert.equal(realm.adminEventsDetailsEnabled, true);
  const serviceAccount = realm.users.find(
    (candidate) =>
      candidate.serviceAccountClientId === "aviasurveil360-lifecycle",
  );
  assert.ok(serviceAccount, "lifecycle service account must exist");
  assert.deepEqual(
    realm.users.filter(
      (candidate) =>
        !candidate.serviceAccountClientId &&
        (/bootstrap|break.?glass/iu.test(candidate.username ?? "") ||
          (candidate.realmRoles ?? []).includes("realm-admin")),
    ),
    [],
    "realm import must not contain a standing bootstrap or break-glass administrator",
  );
  assert.deepEqual(
    [...serviceAccount.clientRoles["realm-management"]].sort(),
    ["manage-users", "query-users", "view-realm", "view-users"],
  );
  assert.equal(
    serviceAccount.clientRoles["realm-management"].includes(
      "impersonation",
    ),
    false,
  );
  assert.equal(JSON.stringify(realm).includes("LocalInspectorPass"), false);

  const roles = realm.roles.realm.map(({ name }) => name).sort();
  assert.deepEqual(roles, [
    "admin",
    "auditee",
    "executiveDirector",
    "finance",
    "gm",
    "inspector",
    "leadInspector",
    "manager",
  ]);

  const client = webClient(realm);
  assert.ok(client, "aviasurveil360-web client must exist");
  assert.equal(client.publicClient, false);
  assert.equal(client.standardFlowEnabled, true);
  assert.equal(client.directAccessGrantsEnabled, false);
  assert.equal(client.serviceAccountsEnabled, false);
  assert.deepEqual(client.redirectUris, [
    "https://localhost:8443/auth/callback",
  ]);
  assert.deepEqual(client.webOrigins, ["https://localhost:8443"]);
  assert.equal(client.attributes["pkce.code.challenge.method"], "S256");
  assert.equal(
    client.attributes["post.logout.redirect.uris"],
    "https://localhost:8443/*",
  );
  assert.equal(client.secret, "__AVIA_OIDC_CLIENT_SECRET__");
  const serviceClient = lifecycleClient(realm);
  assert.ok(serviceClient, "lifecycle service client must exist");
  assert.equal(serviceClient.publicClient, false);
  assert.equal(serviceClient.standardFlowEnabled, false);
  assert.equal(serviceClient.directAccessGrantsEnabled, false);
  assert.equal(serviceClient.serviceAccountsEnabled, true);
  assert.equal(serviceClient.secret, "__AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET__");

  const configureTOTP = realm.requiredActions.find(
    (action) => action.alias === "CONFIGURE_TOTP",
  );
  const invitationActions = realm.requiredActions
    .filter((action) =>
      ["UPDATE_PASSWORD", "VERIFY_EMAIL"].includes(action.alias)
    )
    .map(({ alias, enabled, defaultAction }) => ({
      alias,
      enabled,
      defaultAction,
    }))
    .sort((left, right) => left.alias.localeCompare(right.alias));
  assert.deepEqual(invitationActions, [
    {
      alias: "UPDATE_PASSWORD",
      enabled: true,
      defaultAction: false,
    },
    {
      alias: "VERIFY_EMAIL",
      enabled: true,
      defaultAction: false,
    },
  ]);
  assert.deepEqual(
    {
      enabled: configureTOTP?.enabled,
      defaultAction: configureTOTP?.defaultAction,
    },
    { enabled: true, defaultAction: false },
  );
  assert.equal(realm.otpPolicyType, "totp");
  assert.deepEqual(realm.smtpServer, {
    host: "mailpit",
    port: "1025",
    from: "no-reply@aviasurveil360.test",
    fromDisplayName: "AviaSurveil360",
    auth: "true",
    user: "aviasurveil360",
    password: "__AVIA_KEYCLOAK_SMTP_PASSWORD__",
    starttls: "false",
    ssl: "false",
  });

  const mapperNames = client.protocolMappers.map(({ name }) => name).sort();
  assert.deepEqual(mapperNames, [
    "AviaSurveil360 organization",
    "AviaSurveil360 roles",
    "provider session id",
  ]);

  const profile = userProfile(realm);
  assert.equal(
    Object.hasOwn(profile, "unmanagedAttributePolicy"),
    false,
    "omitting the policy is Keycloak 26's fail-closed unmanaged-attribute default",
  );
  const organization = profile.attributes.find(
    (attribute) => attribute.name === "organization_id",
  );
  assert.deepEqual(organization, {
    name: "organization_id",
    displayName: "Organization ID",
    multivalued: false,
    required: { roles: ["admin"] },
    permissions: {
      view: ["admin"],
      edit: ["admin"],
    },
    validations: {
      length: { min: 1, max: 128 },
    },
  });
});

test("realm builder injects the mounted client secret only into a 0600 runtime file", () => {
  const temporaryDirectory = mkdtempSync(
    path.join(tmpdir(), "aviasurveil360-realm-contract-"),
  );
  try {
    const secretPath = path.join(temporaryDirectory, "oidc-client-secret");
    const serviceSecretPath = path.join(
      temporaryDirectory,
      "keycloak-service-client-secret",
    );
    const smtpPasswordPath = path.join(
      temporaryDirectory,
      "keycloak-smtp-password",
    );
    const outputPath = path.join(temporaryDirectory, "realm.json");
    writeFileSync(secretPath, "runtime-client-secret\n", { mode: 0o600 });
    writeFileSync(serviceSecretPath, "runtime-service-secret\n", {
      mode: 0o600,
    });
    writeFileSync(smtpPasswordPath, "runtime-smtp-password\n", {
      mode: 0o600,
    });
    chmodSync(secretPath, 0o600);
    chmodSync(serviceSecretPath, 0o600);
    chmodSync(smtpPasswordPath, 0o600);

    execFileSync(process.execPath, [
      builderPath,
      "--source",
      sourcePath,
      "--output",
      outputPath,
      "--client-secret-file",
      secretPath,
      "--service-client-secret-file",
      serviceSecretPath,
      "--smtp-password-file",
      smtpPasswordPath,
    ]);

    const builtRealm = loadJSON(outputPath);
    assert.equal(webClient(builtRealm).secret, "runtime-client-secret");
    assert.equal(
      lifecycleClient(builtRealm).secret,
      "runtime-service-secret",
    );
    assert.equal(builtRealm.smtpServer.password, "runtime-smtp-password");
    assert.equal(statSync(outputPath).mode & 0o777, 0o600);
    assert.equal(readFileSync(sourcePath, "utf8").includes("runtime-client-secret"), false);
  } finally {
    rmSync(temporaryDirectory, { recursive: true, force: true });
  }
});

test("realm builder can bind a task-owned loopback origin without changing the reviewed source", () => {
  const temporaryDirectory = mkdtempSync(
    path.join(tmpdir(), "aviasurveil360-realm-loopback-"),
  );
  try {
    const secretPath = path.join(temporaryDirectory, "oidc-client-secret");
    const serviceSecretPath = path.join(
      temporaryDirectory,
      "keycloak-service-client-secret",
    );
    const smtpPasswordPath = path.join(
      temporaryDirectory,
      "keycloak-smtp-password",
    );
    const outputPath = path.join(temporaryDirectory, "realm.json");
    writeFileSync(secretPath, "loopback-client-secret\n", { mode: 0o600 });
    writeFileSync(serviceSecretPath, "loopback-service-secret\n", {
      mode: 0o600,
    });
    writeFileSync(smtpPasswordPath, "loopback-smtp-password\n", {
      mode: 0o600,
    });

    execFileSync(process.execPath, [
      builderPath,
      "--source",
      sourcePath,
      "--output",
      outputPath,
      "--client-secret-file",
      secretPath,
      "--service-client-secret-file",
      serviceSecretPath,
      "--smtp-password-file",
      smtpPasswordPath,
      "--public-origin",
      "http://127.0.0.1:4174",
    ]);

    const client = webClient(loadJSON(outputPath));
    assert.deepEqual(client.redirectUris, [
      "http://127.0.0.1:4174/auth/callback",
    ]);
    assert.deepEqual(client.webOrigins, ["http://127.0.0.1:4174"]);
    assert.equal(
      client.attributes["post.logout.redirect.uris"],
      "http://127.0.0.1:4174/*",
    );
    assert.deepEqual(webClient(loadJSON(sourcePath)).redirectUris, [
      "https://localhost:8443/auth/callback",
    ]);
  } finally {
    rmSync(temporaryDirectory, { recursive: true, force: true });
  }
});
