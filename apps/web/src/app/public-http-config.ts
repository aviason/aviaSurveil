import { z } from "zod";

export interface PublicHttpConfig {
  apiBaseUrl: string;
  environmentLabel: string;
}

const publicHttpConfigSchema = z
  .object({
    apiBaseUrl: z.string().min(1),
    environmentLabel: z.string().min(1).max(80),
  })
  .strict();

export function parsePublicHttpConfig(input: unknown): PublicHttpConfig {
  const result = publicHttpConfigSchema.safeParse(input);
  if (!result.success) {
    throw new Error(`Invalid public HTTP configuration: ${result.error.message}`);
  }
  const apiBaseUrl = result.data.apiBaseUrl;
  if (!apiBaseUrl.startsWith("/") || apiBaseUrl.startsWith("//")) {
    throw new Error("Public HTTP configuration must use a same-origin API base URL");
  }
  return result.data;
}

export async function readPublicHttpConfig(
  fetchImplementation: typeof fetch = fetch,
): Promise<PublicHttpConfig> {
  const response = await fetchImplementation("/http-config.json", {
    credentials: "same-origin",
    cache: "no-store",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) {
    throw new Error(`Public HTTP configuration failed with status ${response.status}`);
  }
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.toLowerCase().includes("application/json")) {
    throw new Error("Public HTTP configuration must be JSON");
  }
  return parsePublicHttpConfig(await response.json());
}
