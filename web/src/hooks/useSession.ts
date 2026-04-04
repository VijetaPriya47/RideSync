"use client";

import { useCallback, useEffect, useState } from "react";
import {
  clearSession,
  getSession,
  setSession,
  type SessionUser,
} from "../lib/session";

export function useSession() {
  const [session, setS] = useState<{ token: string; user: SessionUser } | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setS(getSession());
    setReady(true);
  }, []);

  const login = useCallback((token: string, user: SessionUser) => {
    setSession(token, user);
    setS({ token, user });
  }, []);

  const logout = useCallback(() => {
    clearSession();
    setS(null);
  }, []);

  const refresh = useCallback(() => {
    setS(getSession());
  }, []);

  return { session, ready, login, logout, refresh };
}
