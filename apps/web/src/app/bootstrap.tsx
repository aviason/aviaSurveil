import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";

import {
  AppProviders,
  ApplicationRuntimeProvider,
  type ApplicationRuntime,
} from "./providers";
import { AppRouter } from "./router";
import { ScenarioProvider } from "./scenario-context";
import { HttpAuthGate } from "../auth/http-auth-gate";
import { OfflineSubjectBoundary, SessionProvider } from "../auth/session-provider";
import { registerAppShellServiceWorker } from "../offline/update-coordinator";
import { bindBrowserQuiescence } from "../offline/client-quiescence";
import {
  currentBrowserRouteID,
  installBrowserTelemetry,
} from "../telemetry/browser-telemetry-bootstrap";
import { BrowserTelemetryErrorBoundary } from "../telemetry/browser-telemetry-boundary";
import "../styles/app.css";

export function bootstrap(runtime: ApplicationRuntime): void {
  const rootElement = document.getElementById("root");
  if (!rootElement) throw new Error("AviaSurveil360 root element is missing");
  const browserTelemetry = installBrowserTelemetry(
    runtime.buildProfile,
    runtime.buildProfile === "http" ? "runtime" : "demo",
    { disabled: import.meta.env.VITE_AVIA_DISABLE_BROWSER_TELEMETRY === "1" },
  );

  const identityMode =
    runtime.identityMode ?? "oidc-session";
  createRoot(rootElement).render(
    <StrictMode>
      <BrowserTelemetryErrorBoundary
        telemetry={browserTelemetry}
        routeID={currentBrowserRouteID}
      >
        <AppProviders runtime={{ ...runtime, identityMode }}>
          {identityMode === "oidc-session" && runtime.sessionClient ? (
            <SessionProvider client={runtime.sessionClient} identityMode="oidc-session">
              <BrowserRouter>
                <HttpAuthGate>
                  {(state) => {
                    const authenticatedRuntime: ApplicationRuntime = {
                      ...runtime,
                      identityMode,
                      subjectId: state.session.subjectId,
                    };
                    return (
                      <ApplicationRuntimeProvider runtime={authenticatedRuntime}>
                        <OfflineSubjectBoundary subjectId={state.session.subjectId}>
                          <ScenarioProvider>
                            <AppRouter />
                          </ScenarioProvider>
                        </OfflineSubjectBoundary>
                      </ApplicationRuntimeProvider>
                    );
                  }}
                </HttpAuthGate>
              </BrowserRouter>
            </SessionProvider>
          ) : (
            <ScenarioProvider>
              <BrowserRouter>
                <AppRouter />
              </BrowserRouter>
            </ScenarioProvider>
          )}
        </AppProviders>
      </BrowserTelemetryErrorBoundary>
    </StrictMode>,
  );

  void registerAppShellServiceWorker()
    .then((registration) => {
      if (registration) bindBrowserQuiescence(registration);
    })
    .catch(() => {
      window.dispatchEvent(new CustomEvent("avia:app-shell-registration-failed"));
    });
}
