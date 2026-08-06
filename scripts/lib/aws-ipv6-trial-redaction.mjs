#!/usr/bin/env node

import { readFileSync } from "node:fs";

const SECRET_VALUE = /(-----BEGIN(?: RSA| EC| OPENSSH)? PRIVATE KEY-----|\bsk-[A-Za-z0-9]{16,}\b|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.)/u;

function isSensitiveKey(key) {
  const normalized = key.replaceAll("_", "").replaceAll("-", "").toLowerCase();
  return new Set([
    "password",
    "secret",
    "secretstring",
    "secretvalue",
    "token",
    "tokenvalue",
    "apitoken",
    "connectortoken",
    "privatekey",
    "credential",
    "authorizationheader",
  ]).has(normalized);
}

function scan(value, location, errors) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) => scan(entry, `${location}[${index}]`, errors));
    return;
  }
  if (value === null || typeof value !== "object") {
    if (typeof value === "string" && SECRET_VALUE.test(value)) errors.push(`secret-looking-value:${location}`);
    return;
  }
  for (const [key, entry] of Object.entries(value)) {
    const childLocation = `${location}.${key}`;
    if (isSensitiveKey(key) && typeof entry === "string" && entry !== "") {
      errors.push(`unredacted-plan:${childLocation}`);
    }
    scan(entry, childLocation, errors);
  }
}

export function validateRedactedPlan(value) {
  const errors = [];
  scan(value, "plan", errors);
  return [...new Set(errors)];
}

if (process.argv[1] && new URL(import.meta.url).pathname === process.argv[1]) {
  let source = "";
  try {
    source = process.argv[2] ? readFileSync(process.argv[2], "utf8") : await new Promise((resolve, reject) => {
      let input = "";
      process.stdin.setEncoding("utf8");
      process.stdin.on("data", (chunk) => { input += chunk; });
      process.stdin.on("end", () => resolve(input));
      process.stdin.on("error", reject);
    });
    const errors = validateRedactedPlan(JSON.parse(source));
    if (errors.length > 0) {
      for (const error of errors) console.error(error);
      process.exitCode = 65;
    } else {
      console.log("verified locally: IPv6 trial plan redaction scan");
    }
  } catch (error) {
    console.error(`invalid-plan-json:${error instanceof Error ? error.message : String(error)}`);
    process.exitCode = 64;
  }
}
