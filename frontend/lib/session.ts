"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import type { Role, Session } from "./types";

const KEY = "turnin.session";

export const homeFor = (role: Role) => (role === "professor" ? "/professor" : "/student");

export function readSession(): Session | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(KEY);
    return raw ? (JSON.parse(raw) as Session) : null;
  } catch {
    return null;
  }
}
export const writeSession = (s: Session) => window.localStorage.setItem(KEY, JSON.stringify(s));
export const clearSession = () => window.localStorage.removeItem(KEY);

/** Client-side route guard. Returns null until the session is verified for `required`. */
export function useSession(required: Role): Session | null {
  const router = useRouter();
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    const s = readSession();
    if (!s) {
      router.replace("/login");
    } else if (s.user.role !== required) {
      router.replace(homeFor(s.user.role));
    } else {
      setSession(s);
    }
  }, [required, router]);

  return session;
}
