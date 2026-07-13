import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { onTokenChange, tokens } from "../lib/api";
import { api } from "../lib/endpoints";
import type { Profile, TokenResponse } from "../lib/types";
import { useMarkets } from "../store/markets";
import { passkeyLogin, passkeyRegister } from "./passkey";
import { appleSignIn } from "./apple";

interface AuthCtx {
  ready: boolean;
  authed: boolean;
  profile: Profile | null;
  busy: boolean;
  error: string | null;
  clearError: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  loginWithGoogleToken: (idToken: string) => Promise<void>;
  loginWithApple: () => Promise<void>;
  loginWithPasskey: () => Promise<void>;
  registerPasskey: () => Promise<boolean>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
  setProfile: (p: Profile) => void;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [authed, setAuthed] = useState(!!tokens.access);
  const [profile, setProfileState] = useState<Profile | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const connect = useMarkets((s) => s.connect);

  const refreshProfile = useCallback(async () => {
    try {
      const p = await api.profile();
      setProfileState(p);
    } catch {
      /* ignore */
    }
  }, []);

  const applyTokens = useCallback(
    async (t: TokenResponse) => {
      tokens.set(t.access_token, t.refresh_token);
      setAuthed(true);
      connect(t.access_token);
      await refreshProfile();
    },
    [connect, refreshProfile],
  );

  // Initial session bootstrap + token-change (refresh/expiry) handling.
  useEffect(() => {
    (async () => {
      if (tokens.access) {
        connect(tokens.access);
        await refreshProfile();
      } else {
        connect(null);
      }
      setReady(true);
    })();
    const off = onTokenChange((access) => {
      if (access) {
        setAuthed(true);
        connect(access);
      } else {
        setAuthed(false);
        setProfileState(null);
        connect(null);
      }
    });
    return () => {
      off();
    };
  }, [connect, refreshProfile]);

  const wrap = useCallback(
    async (fn: () => Promise<void>) => {
      setBusy(true);
      setError(null);
      try {
        await fn();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Bir hata oluştu");
        throw e;
      } finally {
        setBusy(false);
      }
    },
    [],
  );

  const login = useCallback(
    (email: string, password: string) =>
      wrap(async () => applyTokens(await api.login(email, password))),
    [wrap, applyTokens],
  );
  const register = useCallback(
    (email: string, password: string) =>
      wrap(async () => applyTokens(await api.register(email, password))),
    [wrap, applyTokens],
  );
  const loginWithGoogleToken = useCallback(
    (idToken: string) => wrap(async () => applyTokens(await api.google(idToken))),
    [wrap, applyTokens],
  );
  const loginWithApple = useCallback(
    () =>
      wrap(async () => {
        const { idToken, fullName } = await appleSignIn();
        applyTokens(await api.apple(idToken, fullName));
      }),
    [wrap, applyTokens],
  );
  const loginWithPasskey = useCallback(
    () => wrap(async () => applyTokens(await passkeyLogin())),
    [wrap, applyTokens],
  );

  const registerPasskey = useCallback(async () => {
    try {
      await passkeyRegister();
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Geçiş anahtarı hatası");
      return false;
    }
  }, []);

  const logout = useCallback(() => {
    const rt = tokens.refresh;
    if (rt) api.logout(rt).catch(() => {});
    tokens.clear();
    setAuthed(false);
    setProfileState(null);
    connect(null);
  }, [connect]);

  const value = useMemo<AuthCtx>(
    () => ({
      ready,
      authed,
      profile,
      busy,
      error,
      clearError: () => setError(null),
      login,
      register,
      loginWithGoogleToken,
      loginWithApple,
      loginWithPasskey,
      registerPasskey,
      logout,
      refreshProfile,
      setProfile: setProfileState,
    }),
    [
      ready,
      authed,
      profile,
      busy,
      error,
      login,
      register,
      loginWithGoogleToken,
      loginWithApple,
      loginWithPasskey,
      registerPasskey,
      logout,
      refreshProfile,
    ],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error("useAuth must be used within AuthProvider");
  return c;
}
