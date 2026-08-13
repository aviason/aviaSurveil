import { createHttpBackend } from "../../src/backend/http-backend";
import { createCanonicalTestFetch } from "../../src/test-profile/http-test-boundary";
import {
  backendContract,
  type BackendContractFixture,
  type BackendContractHarness,
} from "./backend-contract";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

backendContract(async (fixture: BackendContractFixture = "canonical"): Promise<BackendContractHarness> => {
  const resetURL = new URL(`${apiURL}/__test/reset`);
  if (fixture === "coordination") resetURL.searchParams.set("fixture", fixture);
  const response = await fetch(resetURL, {
    method: "POST",
    headers: { "x-avia-test-token": testToken },
  });
  if (!response.ok) {
    throw new Error(`Canonical HTTP reset failed with ${response.status}: ${await response.text()}`);
  }
  return {
    backendFor: (principal) => {
      const subjectId =
        principal.subjectId === "USR-INSPECTOR-AMINA"
          ? "154ec5ac-6f97-4f55-916f-d2f142fc6211"
          : principal.subjectId;
      return createHttpBackend(
        { apiBaseUrl: apiURL, environmentLabel: "Canonical HTTP contract" },
        { fetchImplementation: createCanonicalTestFetch(subjectId, testToken) },
      );
    },
  };
});
