import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { login as loginRequest, logout as logoutRequest, refreshSession, setSessionExpiredHandler } from "./api";
import type { Manager } from "./types";

type AuthValue = {
  manager: Manager | null;
  ready: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  replaceManager: (manager: Manager) => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [manager, setManager] = useState<Manager | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let active = true;
    setSessionExpiredHandler(() => {
      if (active) setManager(null);
    });
    refreshSession()
      .then((session) => {
        if (active) setManager(session.manager);
      })
      .catch(() => {
        if (active) setManager(null);
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
      manager,
      ready,
      signIn: async (email, password) => {
        const session = await loginRequest(email, password);
        setManager(session.manager);
      },
      signOut: async () => {
        await logoutRequest();
        setManager(null);
      },
      replaceManager: setManager,
    }),
    [ready, manager],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthProvider is missing");
  return value;
}
