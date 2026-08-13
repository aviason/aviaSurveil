import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const compose = readFileSync(
  new URL("../compose.yaml", import.meta.url),
  "utf8",
);
const initScript = readFileSync(new URL("./init.sh", import.meta.url), "utf8");
const gateway = readFileSync(new URL("../gateway/Caddyfile", import.meta.url), "utf8");

const buckets = [
  "evidence-quarantine",
  "evidence-clean",
  "inspection-attachments",
  "generated-documents",
];

test("all object buckets are private, separate, and versioned", () => {
  for (const bucket of buckets) {
    assert.match(initScript, new RegExp(`\\b${bucket}\\b`));
  }
  assert.match(initScript, /mc mb --ignore-existing "\$alias\/\$bucket"/);
  assert.match(initScript, /mc version enable "\$alias\/\$bucket"/);
  assert.match(initScript, /mc anonymous set none/);
  assert.doesNotMatch(initScript, /mc anonymous set (download|public)/);
});

test("runtime services never receive MinIO root credentials", () => {
  for (const service of ["api", "worker"]) {
    const block = compose.match(
      new RegExp(`\\n  ${service}:([\\s\\S]*?)(?=\\n  [a-z][a-z0-9-]*:|\\nconfigs:)`),
    )?.[1];
    assert.ok(block, `missing ${service} service`);
    assert.doesNotMatch(block, /minio_root_(user|password)/);
  }
  assert.match(compose, /minio-init\.sh/);
  assert.match(compose, /minio_api_access_key/);
  assert.match(compose, /minio_worker_access_key/);
});

test("fresh named volumes are initialized once before non-root MinIO starts", () => {
  const initializer = compose.match(
    /\n  minio-volume-init:([\s\S]*?)(?=\n  [a-z][a-z0-9-]*:|\nconfigs:)/,
  )?.[1];
  assert.ok(initializer, "missing MinIO volume initializer");
  assert.match(initializer, /user:\s*"0:0"/);
  assert.match(initializer, /entrypoint:\s*\[\/usr\/bin\/chown\]/);
  assert.match(initializer, /command:\s*\[-R,\s*1000:1000,\s*\/data\]/);
  assert.match(initializer, /network_mode:\s*none/);
  assert.doesNotMatch(initializer, /\n\s+networks:/);

  const minio = compose.match(
    /\n  minio:([\s\S]*?)(?=\n  [a-z][a-z0-9-]*:|\nconfigs:)/,
  )?.[1];
  assert.match(minio, /user:\s*"1000:1000"/);
  assert.match(minio, /minio-volume-init:/);
  assert.match(minio, /condition:\s*service_completed_successfully/);
});

test("least-privilege policies separate API and worker capabilities", () => {
  assert.match(initScript, /api-policy\.json/);
  assert.match(initScript, /worker-policy\.json/);
  assert.match(initScript, /s3:GetObject/);
  assert.match(initScript, /s3:PutObject/);
  assert.match(initScript, /s3:GetBucketLocation/);
  assert.doesNotMatch(initScript, /s3:\*/);
});

test("credential-bearing MinIO administration output never reaches runtime logs", () => {
  assert.match(initScript, /run_sensitive_mc\(\)/);
  assert.match(initScript, /run_sensitive_mc mc admin user add/);
  assert.match(initScript, /run_sensitive_mc mc admin policy attach/);
  assert.doesNotMatch(initScript, /^mc admin user add/m);
  assert.doesNotMatch(initScript, /^mc admin policy attach/m);
});

test("full profile keeps scanning disabled and preserves named buckets", () => {
  assert.match(compose, /AVIA_SCANNER_MODE:\s*disabled/);
  assert.match(compose, /AVIA_OBJECT_STORE_QUARANTINE_BUCKET:\s*evidence-quarantine/);
  assert.match(compose, /AVIA_OBJECT_STORE_CANONICAL_BUCKET:\s*evidence-clean/);
  assert.match(compose, /AVIA_OBJECT_STORE_ATTACHMENT_BUCKET:\s*inspection-attachments/);
  assert.match(compose, /AVIA_OBJECT_STORE_DOCUMENT_BUCKET:\s*generated-documents/);
  assert.doesNotMatch(compose, /clamav|gotenberg/iu);
});

test("signed object traffic stays on the one HTTPS gateway origin", () => {
  assert.match(
    compose,
    /AVIA_OBJECT_STORE_PUBLIC_ENDPOINT:\s*["']?localhost:\$\{AVIA_LOCAL_HTTPS_PORT:-8443\}["']?/,
  );
  assert.match(compose, /AVIA_OBJECT_STORE_PUBLIC_TLS:\s*"true"/);
  for (const bucket of buckets) {
    assert.match(gateway, new RegExp(`/${bucket}/\\*`));
  }
  assert.match(gateway, /reverse_proxy minio:9000/);
});
