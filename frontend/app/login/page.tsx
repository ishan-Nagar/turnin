"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api, messageOf } from "@/lib/api";
import { clearSession, homeFor, writeSession } from "@/lib/session";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  // Visiting /login always starts from a clean slate
  useEffect(() => {
    clearSession();
  }, []);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setPending(true);
    setError(null);
    try {
      const session = await api.login(email, password);
      writeSession(session);
      router.replace(homeFor(session.user.role));
    } catch (err) {
      setError(messageOf(err));
      setPending(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-sm space-y-8">
        <div className="space-y-2">
          <p className="font-mono text-sm font-semibold tracking-tight">
            turnin<span className="text-muted-foreground">/</span>
          </p>
          <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
          <p className="text-sm text-muted-foreground">
            Use your institutional account to continue.
          </p>
        </div>

        <form onSubmit={onSubmit} className="space-y-4" noValidate>
          <div className="space-y-1.5">
            <label htmlFor="email" className="text-xs font-medium">
              Email
            </label>
            <Input
              id="email"
              type="email"
              autoComplete="username"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@nalanda.ac.in"
            />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="password" className="text-xs font-medium">
              Password
            </label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          {error && (
            <p
              role="alert"
              className="rounded-md border border-destructive/40 px-3 py-2 text-xs text-destructive"
            >
              {error}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={pending || !email || !password}
          >
            {pending && <Loader2 className="h-4 w-4 animate-spin" />}
            Continue
          </Button>
        </form>
      </div>
    </div>
  );
}

