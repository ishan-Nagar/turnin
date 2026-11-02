"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { clearSession } from "@/lib/session";
import type { Session } from "@/lib/types";

export function AppShell({ session, children }: { session: Session; children: React.ReactNode }) {
  const router = useRouter();

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <span className="font-mono text-sm font-semibold tracking-tight">
              turnin<span className="text-muted-foreground">/</span>
            </span>
            <span className="rounded border px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
              {session.user.role}
            </span>
          </div>
          <div className="flex items-center gap-4">
            <div className="hidden text-right leading-tight sm:block">
              <p className="text-sm">{session.user.name}</p>
              <p className="font-mono text-[11px] text-muted-foreground">{session.user.email}</p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => { clearSession(); router.replace("/login"); }}
            >
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl space-y-6 px-6 py-8">{children}</main>
    </div>
  );
}
