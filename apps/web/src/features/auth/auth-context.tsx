"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { AuthApi, type Authentication, type User } from "@/lib/auth-api";

export type { Authentication } from "@/lib/auth-api";

export function createRefreshGate(refresh: () => Promise<Authentication>) {
  let inFlight: Promise<Authentication> | null = null;
  return {
    refresh() {
      if (inFlight) return inFlight;
      inFlight = refresh().finally(() => {
        inFlight = null;
      });
      return inFlight;
    },
  };
}

type AuthState = {
  user: User | null;
  ready: boolean;
  login(email: string, password: string): Promise<void>;
  register(name: string, email: string, password: string): Promise<void>;
  logout(): Promise<void>;
  protectedRequest(
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response>;
};

const AuthContext = createContext<AuthState | null>(null);
const api = new AuthApi(process.env.NEXT_PUBLIC_API_URL ?? "/api/v1");
const refreshGate = createRefreshGate(() => api.refresh());

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const accessToken = useRef<string | null>(null);
  const setAuthentication = useCallback(
    (result: { user: User; accessToken: string }) => {
      accessToken.current = result.accessToken;
      setUser(result.user);
    },
    [],
  );
  useEffect(() => {
    refreshGate
      .refresh()
      .then(setAuthentication)
      .catch(() => {
        accessToken.current = null;
        setUser(null);
      })
      .finally(() => setReady(true));
  }, [setAuthentication]);
  const protectedRequest = useCallback(
    async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = (token: string | null) =>
        fetch(input, {
          ...init,
          credentials: "include",
          headers: {
            ...init?.headers,
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
        });
      const first = await request(accessToken.current);
      if (first.status !== 401) return first;
      try {
        const refreshed = await refreshGate.refresh();
        setAuthentication(refreshed);
        return request(refreshed.accessToken);
      } catch {
        accessToken.current = null;
        setUser(null);
        return first;
      }
    },
    [setAuthentication],
  );
  const value = useMemo<AuthState>(
    () => ({
      user,
      ready,
      async login(email, password) {
        const result = await api.login(email, password);
        setAuthentication(result);
      },
      async register(name, email, password) {
        await api.register(name, email, password);
      },
      async logout() {
        await api.logout();
        accessToken.current = null;
        setUser(null);
      },
      protectedRequest,
    }),
    [protectedRequest, ready, setAuthentication, user],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used within AuthProvider");
  return value;
}
