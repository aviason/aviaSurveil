import { bootstrap } from "../app/bootstrap";
import { readPublicHttpConfig } from "../app/public-http-config";
import { createSessionClient } from "../auth/session-client";
import { createHttpBackend } from "../backend/http-backend";

async function start(): Promise<void> {
  const config = await readPublicHttpConfig();
  const sessionClient = createSessionClient();
  bootstrap({
    backend: createHttpBackend(config, {
      csrfToken: sessionClient.csrfToken,
      onAuthenticationLost: () => {
        window.dispatchEvent(new CustomEvent("avia:authentication-lost"));
      },
    }),
    buildProfile: "http",
    environmentLabel: config.environmentLabel,
    identityMode: "oidc-session",
    sessionClient,
  });
}

void start();
