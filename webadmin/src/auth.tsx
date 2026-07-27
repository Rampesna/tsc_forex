import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { login as loginRequest, logout as logoutRequest, refreshSession, setSessionExpiredHandler } from "./api";
import type { AdminUser } from "./types";

type AuthValue = {
  user: AdminUser | null;
  ready: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  replaceUser: (user: AdminUser) => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AdminUser | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let active = true;
    setSessionExpiredHandler(() => {
      if (active) setUser(null);
    });
    refreshSession()
      .then((session) => {
        if (active) setUser(session.user);
      })
      .catch(() => {
        if (active) setUser(null);
      })
      .finally(() => {
        if (active) setReady(true);
      });
    return () => {
      active = false;
      setSessionExpiredHandler(null);
    };
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      user,
      ready,
      signIn: async (email, password) => {
        const session = await loginRequest(email, password);
        setUser(session.user);
      },
      signOut: async () => {
        await logoutRequest();
        setUser(null);
      },
      replaceUser: setUser,
    }),
    [ready, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthProvider is missing");
  return value;
}
