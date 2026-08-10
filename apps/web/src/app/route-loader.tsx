import {
  Component,
  lazy,
  type ComponentType,
  type LazyExoticComponent,
  type PropsWithChildren,
  type ReactNode,
} from "react";

export const ROUTE_LOAD_TIMEOUT_MS = 15_000;

export function loadRouteWithTimeout<T>(
  loader: () => Promise<T>,
  timeoutMs = ROUTE_LOAD_TIMEOUT_MS,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let settled = false;
    const timeout = globalThis.setTimeout(() => {
      settled = true;
      reject(new Error("The requested workspace route could not be loaded. The local preprod connection may be restarting."));
    }, timeoutMs);

    Promise.resolve()
      .then(loader)
      .then(
        (module) => {
          if (settled) return;
          settled = true;
          globalThis.clearTimeout(timeout);
          resolve(module);
        },
        (error: unknown) => {
          if (settled) return;
          settled = true;
          globalThis.clearTimeout(timeout);
          reject(error);
        },
      );
  });
}

export function lazyRoute<T extends ComponentType>(
  loader: () => Promise<{ default: T }>,
): LazyExoticComponent<T> {
  return lazy(() => loadRouteWithTimeout(loader));
}

export function RouteLoadingState() {
  return (
    <main className="route-load-state" data-testid="route-loading" role="status">
      <section className="route-load-card" aria-label="Loading workspace route">
        <span className="route-load-mark" aria-hidden="true">AS</span>
        <p className="route-load-eyebrow">AviaSurveil360</p>
        <h1>Loading workspace…</h1>
        <p>Restoring the secured route and its current session.</p>
        <span className="route-load-progress" aria-hidden="true" />
      </section>
    </main>
  );
}

export function RouteLoadFailure({ onReload = () => window.location.reload() }: { onReload?: () => void }) {
  return (
    <main className="route-load-state route-load-state--failed">
      <section className="route-load-card" role="alert" aria-labelledby="route-load-failure-title">
        <span className="route-load-mark" aria-hidden="true">AS</span>
        <p className="route-load-eyebrow">Connection interrupted</p>
        <h1 id="route-load-failure-title">This workspace route could not be loaded</h1>
        <p>The local preprod services may be restarting. Reload after they are ready to restore your signed-in workspace.</p>
        <button onClick={onReload} type="button">Reload application</button>
      </section>
    </main>
  );
}

interface RouteLoadBoundaryProps extends PropsWithChildren {
  resetKey: string;
  fallback?: ReactNode;
}

interface RouteLoadBoundaryState {
  failed: boolean;
}

export class RouteLoadBoundary extends Component<RouteLoadBoundaryProps, RouteLoadBoundaryState> {
  state: RouteLoadBoundaryState = { failed: false };

  static getDerivedStateFromError(): RouteLoadBoundaryState {
    return { failed: true };
  }

  componentDidUpdate(previousProps: RouteLoadBoundaryProps) {
    if (this.state.failed && previousProps.resetKey !== this.props.resetKey) {
      this.setState({ failed: false });
    }
  }

  render() {
    if (this.state.failed) return this.props.fallback ?? <RouteLoadFailure />;
    return this.props.children;
  }
}
