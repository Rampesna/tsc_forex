import { lazy, Suspense, type ReactNode } from "react";
import { Navigate, useLocation } from "./router";
import { useAuth } from "./auth";
import { Layout } from "./Layout";
import { Login } from "./pages/Login";
import { LoadingBlock } from "./UI";

const Dashboard = lazy(() => import("./pages/Dashboard").then((module) => ({ default: module.Dashboard })));
const Users = lazy(() => import("./pages/Users").then((module) => ({ default: module.Users })));
const UserDetailPage = lazy(() => import("./pages/UserDetail").then((module) => ({ default: module.UserDetailPage })));
const Online = lazy(() => import("./pages/Online").then((module) => ({ default: module.Online })));
const Revenue = lazy(() => import("./pages/Revenue").then((module) => ({ default: module.Revenue })));
const Campaigns = lazy(() => import("./pages/Campaigns").then((module) => ({ default: module.Campaigns })));
const Audit = lazy(() => import("./pages/Audit").then((module) => ({ default: module.Audit })));
const Settings = lazy(() => import("./pages/Settings").then((module) => ({ default: module.Settings })));

export default function App() {
  const { ready, user } = useAuth();
  const { pathname } = useLocation();
  if (!ready) return <div className="boot"><div className="brand-mark"><span>T</span><span>F</span></div><span className="spinner" /></div>;
  if (pathname === "/login") return user ? <Navigate to="/" replace /> : <Login />;
  if (!user) return <Navigate to="/login" replace />;

  let page: ReactNode;
  if (pathname === "/") page = <Dashboard />;
  else if (pathname === "/users") page = <Users />;
  else if (/^\/users\/[^/]+$/.test(pathname)) page = <UserDetailPage />;
  else if (pathname === "/online") page = <Online />;
  else if (pathname === "/revenue") page = <Revenue />;
  else if (pathname === "/campaigns") page = <Campaigns />;
  else if (pathname === "/audit") page = <Audit />;
  else if (pathname === "/settings") page = <Settings />;
  else return <Navigate to="/" replace />;

  return <Layout><Suspense fallback={<LoadingBlock height={520} />}>{page}</Suspense></Layout>;
}
