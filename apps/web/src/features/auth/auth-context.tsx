"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { AuthApi, type User } from "@/lib/auth-api";

type AuthState = {
  user: User | null;
  ready: boolean;
  login(email: string, password: string): Promise<void>;
  register(name: string, email: string, password: string): Promise<void>;
  logout(): Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);
const api = new AuthApi(process.env.NEXT_PUBLIC_API_URL ?? "/api/v1");

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    api
      .refresh()
      .then((result) => setUser(result.user))
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);
  const value = useMemo<AuthState>(
    () => ({
      user,
      ready,
      async login(email, password) {
        const result = await api.login(email, password);
        setUser(result.user);
      },
      async register(name, email, password) {
        const result = await api.register(name, email, password);
        setUser(result.user);
      },
      async logout() {
        await api.logout();
        setUser(null);
      },
    }),
    [ready, user],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used within AuthProvider");
  return value;
}
