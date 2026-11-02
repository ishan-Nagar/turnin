import { clearSession, readSession } from "./session";
import type { Assignment, Session, Submission } from "./types";

const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}
export const messageOf = (e: unknown) => (e instanceof Error ? e.message : "Unexpected error");

async function raw(path: string, init: RequestInit = {}): Promise<Response> {
  const session = readSession();
  const headers = new Headers(init.headers);
  if (session) headers.set("Authorization", `Bearer ${session.token}`);

  let res: Response;
  try {
    res = await fetch(`${BASE}${path}`, { ...init, headers });
  } catch {
    throw new ApiError(0, "network_error", "Unable to reach the server");
  }
  if (res.status === 401 && session) {
    clearSession();
    window.location.assign("/login");
  }
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res;
}

async function json<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await raw(path, init);
  return (await res.json()).data as T;
}

export const api = {
  login: (email: string, password: string) =>
    json<Session>("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    }),

  listAssignments: () => json<Assignment[]>("/api/assignments"),
  listSubmissions: () => json<Submission[]>("/api/submissions"),
  gradeSubmission: (submissionId: string, score: number) =>
    json<Submission>(`/api/submissions/${submissionId}/grade`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ score }),
    }),

  submit: (assignmentId: string, file: File) => {
    const form = new FormData();
    form.append("file", file);
    return json<Submission>(`/api/assignments/${assignmentId}/submissions`, { method: "POST", body: form });
  },

  /** Authenticated download: fetch as blob, then trigger a save. */
  async download(submission: Submission, filename: string) {
    const res = await raw(`/api/submissions/${submission.id}/file`);
    const url = URL.createObjectURL(await res.blob());
    const a = Object.assign(document.createElement("a"), { href: url, download: filename });
    a.click();
    URL.revokeObjectURL(url);
  },
};

