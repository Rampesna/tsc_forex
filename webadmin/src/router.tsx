import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type MouseEvent,
  type ReactNode,
} from "react";

type NavigateOptions = { replace?: boolean };
type RouterValue = {
  pathname: string;
  navigate: (to: string, options?: NavigateOptions) => void;
};

const RouterContext = createContext<RouterValue | null>(null);

function currentPath() {
  const path = window.location.pathname.replace(/\/+$/, "");
  return path || "/";
}

export function RouterProvider({ children }: { children: ReactNode }) {
  const [pathname, setPathname] = useState(currentPath);

  useEffect(() => {
    const onPopState = () => setPathname(currentPath());
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  const value = useMemo<RouterValue>(() => ({
    pathname,
    navigate: (to, options = {}) => {
      const normalized = to === "/" ? "/" : `/${to.replace(/^\/+|\/+$/g, "")}`;
      if (normalized === currentPath()) return;
      window.history[options.replace ? "replaceState" : "pushState"]({}, "", normalized);
      setPathname(normalized);
      window.scrollTo({ top: 0, behavior: "auto" });
    },
  }), [pathname]);

  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

function useRouter() {
  const value = useContext(RouterContext);
  if (!value) throw new Error("RouterProvider is missing");
  return value;
}

export function useLocation() {
  const { pathname } = useRouter();
  return { pathname };
}

export function useNavigate() {
  return useRouter().navigate;
}

export function useParams() {
  const { pathname } = useRouter();
  const match = pathname.match(/^\/users\/([^/]+)$/);
  return { id: match ? decodeURIComponent(match[1]) : undefined };
}

export function Navigate({ to, replace = false }: { to: string; replace?: boolean }) {
  const navigate = useNavigate();
  useEffect(() => navigate(to, { replace }), [navigate, replace, to]);
  return null;
}

export function NavLink({
  to,
  end = false,
  children,
  onClick,
}: {
  to: string;
  end?: boolean;
  children: ReactNode;
  onClick?: () => void;
}) {
  const { pathname, navigate } = useRouter();
  const active = end ? pathname === to : pathname === to || pathname.startsWith(`${to}/`);

  function follow(event: MouseEvent<HTMLAnchorElement>) {
    onClick?.();
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    event.preventDefault();
    navigate(to);
  }

  return <a href={to} className={active ? "active" : ""} onClick={follow}>{children}</a>;
}
