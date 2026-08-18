import crypto from "node:crypto";
import fs from "node:fs";

const args = process.argv.slice(2);
const value = (name) => {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
};

if (args.includes("--help") || args.length === 0) {
  console.log("usage: node scripts/record-offline-release-qualification.mjs --lane <official-lane> --artifact <sha256:digest|artifact-path>");
  process.exit(0);
}

const lane = value("--lane");
const artifact = value("--artifact");
const officialLanes = new Set([
  "ios-ipados-safari",
  "ios-ipados-chrome",
  "macos-safari",
  "macos-chrome",
  "android-chrome",
  "windows-chrome",
]);
if (!officialLanes.has(lane) || !artifact) {
  console.error("lane must be one of the six official normal-tab lanes and artifact is required");
  process.exit(2);
}

let artifactDigest = artifact;
if (fs.existsSync(artifact) && fs.statSync(artifact).isFile()) {
  artifactDigest = `sha256:${crypto.createHash("sha256").update(fs.readFileSync(artifact)).digest("hex")}`;
}
if (!/^sha256:[0-9a-f]{64}$/u.test(artifactDigest)) {
  console.error("artifact must be a sha256 digest or a readable artifact file");
  process.exit(2);
}

console.log(JSON.stringify({
  lane,
  artifactDigest,
  runtime: "normal-browser-tab",
  status: "not run",
  evidence: "candidate-only",
  release: "release pending",
  productionReady: false,
  externalWrites: false,
}, null, 2));
