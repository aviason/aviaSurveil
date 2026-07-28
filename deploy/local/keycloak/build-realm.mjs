#!/usr/bin/env node

import { randomUUID } from "node:crypto";
import {
  chmodSync,
  mkdirSync,
  readFileSync,
  renameSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";

const secretPlaceholder = "__AVIA_OIDC_CLIENT_SECRET__";
const serviceSecretPlaceholder = "__AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET__";

function parseArguments(arguments_) {
  const values = new Map();
  for (let index = 0; index < arguments_.length; index += 2) {
    const flag = arguments_[index];
    const value = arguments_[index + 1];
    if (!flag?.startsWith("--") || value === undefined) {
      throw new Error(
        "expected --source, --output, --client-secret-file, and --service-client-secret-file",
      );
    }
    values.set(flag, value);
  }

  const source = values.get("--source");
  const output = values.get("--output");
  const clientSecretFile = values.get("--client-secret-file");
  const serviceClientSecretFile = values.get("--service-client-secret-file");
  const publicOriginValue = values.get("--public-origin");
  if (
    !source ||
    !output ||
    !clientSecretFile ||
    !serviceClientSecretFile ||
    (values.size !== 4 && values.size !== 5)
  ) {
    throw new Error(
      "expected --source, --output, --client-secret-file, --service-client-secret-file, and optional --public-origin",
    );
  }
  let publicOrigin;
  if (publicOriginValue !== undefined) {
    const parsed = new URL(publicOriginValue);
    if (
      (parsed.protocol !== "http:" && parsed.protocol !== "https:") ||
      parsed.username ||
      parsed.password ||
      parsed.search ||
      parsed.hash ||
      (parsed.pathname !== "" && parsed.pathname !== "/")
    ) {
      throw new Error("public origin must be an absolute HTTP(S) origin");
    }
    publicOrigin = parsed.origin;
  }
  return {
    source,
    output,
    clientSecretFile,
    serviceClientSecretFile,
    publicOrigin,
  };
}

function buildRealm({
  source,
  output,
  clientSecretFile,
  serviceClientSecretFile,
  publicOrigin,
}) {
  const sourceText = readFileSync(source, "utf8");
  const placeholderMatches = sourceText.match(
    new RegExp(secretPlaceholder, "g"),
  );
  if (placeholderMatches?.length !== 1) {
    throw new Error("realm source must contain exactly one client-secret placeholder");
  }
  const servicePlaceholderMatches = sourceText.match(
    new RegExp(serviceSecretPlaceholder, "g"),
  );
  if (servicePlaceholderMatches?.length !== 1) {
    throw new Error(
      "realm source must contain exactly one service-client-secret placeholder",
    );
  }

  const clientSecret = readFileSync(clientSecretFile, "utf8").trim();
  if (!clientSecret || clientSecret === secretPlaceholder) {
    throw new Error("OIDC client secret file must contain a non-placeholder value");
  }
  const serviceClientSecret = readFileSync(
    serviceClientSecretFile,
    "utf8",
  ).trim();
  if (
    !serviceClientSecret ||
    serviceClientSecret === serviceSecretPlaceholder
  ) {
    throw new Error(
      "Keycloak service client secret file must contain a non-placeholder value",
    );
  }

  const realm = JSON.parse(sourceText);
  const webClient = realm.clients?.find(
    (candidate) => candidate.clientId === "aviasurveil360-web",
  );
  if (!webClient || webClient.secret !== secretPlaceholder) {
    throw new Error("realm source is missing the reviewed web client");
  }
  webClient.secret = clientSecret;
  const serviceClient = realm.clients?.find(
    (candidate) => candidate.clientId === "aviasurveil360-lifecycle",
  );
  if (!serviceClient || serviceClient.secret !== serviceSecretPlaceholder) {
    throw new Error("realm source is missing the reviewed lifecycle client");
  }
  serviceClient.secret = serviceClientSecret;
  if (publicOrigin) {
    webClient.redirectUris = [`${publicOrigin}/auth/callback`];
    webClient.webOrigins = [publicOrigin];
    webClient.attributes["post.logout.redirect.uris"] = `${publicOrigin}/*`;
  }

  const outputDirectory = path.dirname(output);
  mkdirSync(outputDirectory, { recursive: true, mode: 0o700 });
  const temporaryOutput = path.join(
    outputDirectory,
    `.${path.basename(output)}.${randomUUID()}.tmp`,
  );
  try {
    writeFileSync(temporaryOutput, `${JSON.stringify(realm, null, 2)}\n`, {
      encoding: "utf8",
      flag: "wx",
      mode: 0o600,
    });
    chmodSync(temporaryOutput, 0o600);
    renameSync(temporaryOutput, output);
    chmodSync(output, 0o600);
  } finally {
    rmSync(temporaryOutput, { force: true });
  }
}

buildRealm(parseArguments(process.argv.slice(2)));
