import {
  Children,
  cloneElement,
  isValidElement,
  startTransition,
  useCallback,
  useMemo,
  type AnchorHTMLAttributes,
  type ComponentType,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
} from "react";
import * as LegacyRouter from "react-router-dom-v5";

export interface Location<State = unknown> {
  pathname: string;
  search: string;
  hash: string;
  state: State;
  key?: string;
}

export type To =
  | string
  | {
      pathname?: string;
      search?: string;
      hash?: string;
      state?: unknown;
    };

interface RouterProps {
  children?: ReactNode;
}

interface MemoryRouterProps extends RouterProps {
  initialEntries?: To[];
  initialIndex?: number;
}

interface RouteProps {
  element: ReactElement;
  exact?: boolean;
  path: string;
}

interface NavigateOptions {
  replace?: boolean;
  state?: unknown;
}

interface NavigateProps extends NavigateOptions {
  to: To;
}

interface LinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  replace?: boolean;
  to: To;
}

type SearchParamsInit =
  | string
  | URLSearchParams
  | Record<string, string | string[]>
  | Array<[string, string]>;

export const BrowserRouter = LegacyRouter.BrowserRouter as ComponentType<RouterProps>;
export const MemoryRouter = LegacyRouter.MemoryRouter as ComponentType<MemoryRouterProps>;

function isModifiedClick(event: MouseEvent<HTMLAnchorElement>): boolean {
  return event.button !== 0 || event.metaKey || event.altKey || event.ctrlKey || event.shiftKey;
}

function toHref(to: To): string {
  if (typeof to === "string") return to;
  const search = to.search ? (to.search.startsWith("?") ? to.search : `?${to.search}`) : "";
  const hash = to.hash ? (to.hash.startsWith("#") ? to.hash : `#${to.hash}`) : "";
  return `${to.pathname ?? ""}${search}${hash}` || "/";
}

export function Link({ onClick, replace = false, to, ...anchorProps }: LinkProps) {
  const history = LegacyRouter.useHistory();
  const href = toHref(to);

  const handleClick = useCallback(
    (event: MouseEvent<HTMLAnchorElement>) => {
      onClick?.(event);
      if (
        event.defaultPrevented ||
        isModifiedClick(event) ||
        (event.currentTarget.target !== "" && event.currentTarget.target !== "_self") ||
        event.currentTarget.hasAttribute("download")
      ) {
        return;
      }

      event.preventDefault();
      startTransition(() => {
        if (replace) {
          history.replace(to as Parameters<typeof history.replace>[0]);
        } else {
          history.push(to as Parameters<typeof history.push>[0]);
        }
      });
    },
    [history, onClick, replace, to],
  );

  return <a {...anchorProps} href={href} onClick={handleClick} />;
}

export function Routes({ children }: RouterProps) {
  const exactChildren = Children.map(children, (child) =>
    isValidElement<RouteProps>(child)
      ? cloneElement(child, { exact: child.props.exact ?? true })
      : child,
  );
  return <LegacyRouter.Switch>{exactChildren}</LegacyRouter.Switch>;
}

export function Route({ element, exact = true, path }: RouteProps) {
  return <LegacyRouter.Route exact={exact} path={path} render={() => element} />;
}

export function Navigate({ replace = false, state, to }: NavigateProps) {
  const destination =
    typeof to === "string"
      ? { pathname: to, state }
      : { ...to, state: state ?? to.state };
  return <LegacyRouter.Redirect push={!replace} to={destination} />;
}

export function useLocation<State = unknown>() {
  return LegacyRouter.useLocation() as Location<State>;
}

export function useNavigate() {
  const history = LegacyRouter.useHistory();
  return useCallback(
    (to: To | number, options: NavigateOptions = {}) => {
      if (typeof to === "number") {
        startTransition(() => history.go(to));
        return;
      }
      const method = options.replace ? history.replace : history.push;
      startTransition(() => method(to, options.state));
    },
    [history],
  );
}

function decodeRouteParameter(value: string | undefined): string | undefined {
  if (value === undefined) return undefined;
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function useParams<Params extends Record<string, string | undefined> = Record<string, string>>() {
  const params = LegacyRouter.useParams() as Record<string, string | undefined>;
  return Object.fromEntries(
    Object.entries(params).map(([key, value]) => [key, decodeRouteParameter(value)]),
  ) as Params;
}

function createSearchParams(init: SearchParamsInit = "") {
  if (init instanceof URLSearchParams || typeof init === "string" || Array.isArray(init)) {
    return new URLSearchParams(init);
  }
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(init)) {
    for (const item of Array.isArray(value) ? value : [value]) params.append(key, item);
  }
  return params;
}

export function useSearchParams(): [
  URLSearchParams,
  (nextInit: SearchParamsInit, options?: NavigateOptions) => void,
] {
  const location = useLocation();
  const navigate = useNavigate();
  const searchParams = useMemo(
    () => createSearchParams(location.search),
    [location.search],
  );
  const setSearchParams = useCallback(
    (nextInit: SearchParamsInit, options?: NavigateOptions) => {
      const search = createSearchParams(nextInit).toString();
      navigate(
        {
          pathname: location.pathname,
          search: search ? `?${search}` : "",
          hash: location.hash,
        },
        options,
      );
    },
    [location.hash, location.pathname, navigate],
  );
  return [searchParams, setSearchParams];
}
