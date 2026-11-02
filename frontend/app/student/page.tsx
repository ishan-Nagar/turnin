"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, RefreshCw, X } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Stat } from "@/components/stat";
import { StatusBadge } from "@/components/status-badge";
import { UploadPanel } from "@/components/upload-panel";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, messageOf } from "@/lib/api";
import { displayFileName, formatDateTime, relativeTime } from "@/lib/format";
import { useSession } from "@/lib/session";
import type { Assignment, Submission } from "@/lib/types";
import { cn } from "@/lib/utils";

export default function StudentDashboard() {
  const session = useSession("student");

  const [assignments, setAssignments] = useState<Assignment[] | null>(null);
  const [optimistic, setOptimistic] = useState<Record<string, Submission>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const seq = useRef(0);

  const refresh = useCallback(async () => {
    const mine = ++seq.current;
    setRefreshing(true);
    try {
      const data = await api.listAssignments();
      if (mine === seq.current) setAssignments(data);
    } catch (e) {
      if (mine === seq.current) setError(messageOf(e));
    } finally {
      if (mine === seq.current) setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    if (session) void refresh();
  }, [session, refresh]);

  // Derived view = server state ⊕ optimistic overlay.
  const rows = useMemo(
    () => (assignments ?? []).map((a) => ({ ...a, submission: optimistic[a.id] ?? a.submission ?? null })),
    [assignments, optimistic],
  );

  // Poll every 2s while a scan is Pending, or every 15s while awaiting a professor's grade.
  const awaitingScan = (assignments ?? []).some((a) => a.submission?.status === "Pending");
  const awaitingGrade = (assignments ?? []).some(
    (a) => a.submission && a.submission.status !== "Graded",
  );
  useEffect(() => {
    if (!awaitingScan && !awaitingGrade) return;
    const interval = awaitingScan ? 2000 : 15000;
    const t = setTimeout(() => void refresh(), interval);
    return () => clearTimeout(t);
  }, [awaitingScan, awaitingGrade, assignments, refresh]);

  const handleSubmit = useCallback(
    async (assignment: Assignment, file: File) => {
      if (!session) return;
      setError(null);

      // 1. Reflect the submission locally, synchronously, before any network I/O.
      const draft: Submission = {
        id: `optimistic-${assignment.id}`,
        assignment_id: assignment.id,
        student_id: session.user.id,
        file_url: file.name,
        status: "Pending",
        score: null,
        submitted_at: new Date().toISOString(),
        optimistic: true,
      };
      setOptimistic((p) => ({ ...p, [assignment.id]: draft }));
      setSelectedId(null);

      try {
        // 2. Reconcile with the server's truth.
        const saved = await api.submit(assignment.id, file);
        seq.current++;
        setAssignments((prev) =>
          prev ? prev.map((a) => (a.id === assignment.id ? { ...a, submission: saved } : a)) : prev,
        );
      } catch (e) {
        // 3. Roll back and restore context.
        setError(`Submission of “${assignment.title}” failed: ${messageOf(e)}`);
        setSelectedId(assignment.id);
      } finally {
        setOptimistic((p) => {
          const next = { ...p };
          delete next[assignment.id];
          return next;
        });
      }
    },
    [session],
  );

  if (!session) return null;

  const now = Date.now();
  const isOpen = (a: Assignment) => new Date(a.deadline).getTime() > now;
  const sorted = [...rows].sort((x, y) => {
    const rank = (a: (typeof rows)[number]) => (!a.submission && isOpen(a) ? 0 : a.submission ? 1 : 2);
    return rank(x) - rank(y) || +new Date(x.deadline) - +new Date(y.deadline);
  });
  const selected = rows.find((a) => a.id === selectedId && !a.submission && isOpen(a)) ?? null;

  const gradedRows = rows.filter((a) => a.submission && a.submission.score !== null);
  const earnedPts = gradedRows.reduce((sum, a) => sum + (a.submission?.score ?? 0), 0);
  const possiblePts = gradedRows.reduce((sum, a) => sum + a.max_score, 0);

  return (
    <AppShell session={session}>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Assignments</h1>
          <p className="text-sm text-muted-foreground">
            Welcome back, {session.user.name.split(" ")[0]}.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void refresh()} disabled={refreshing}>
          <RefreshCw className={cn("h-3.5 w-3.5", refreshing && "animate-spin")} />
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat
          label="Due"
          value={rows.filter((a) => !a.submission && isOpen(a)).length}
          hint="open, not submitted"
        />
        <Stat label="Submitted" value={rows.filter((a) => a.submission).length} />
        <Stat
          label="Scanning"
          value={rows.filter((a) => a.submission?.status === "Pending").length}
          hint="plagiarism check"
        />
        <Stat
          label="Graded"
          value={gradedRows.length}
          hint={possiblePts > 0 ? `${earnedPts} / ${possiblePts} pts earned` : "no grades posted yet"}
        />
      </div>

      {error && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-md border border-destructive/40 px-4 py-3 text-sm text-destructive"
        >
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <p className="flex-1">{error}</p>
          <button aria-label="Dismiss" onClick={() => setError(null)}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="grid items-start gap-6 lg:grid-cols-[1fr_360px]">
        <Card>
          <CardHeader>
            <CardTitle>Coursework & Grades</CardTitle>
            <CardDescription>Open assignments are listed first. Grades appear here once published.</CardDescription>
          </CardHeader>
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Assignment</TableHead>
                <TableHead>Due</TableHead>
                <TableHead className="text-right">Grade</TableHead>
                <TableHead className="text-right">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {assignments === null &&
                Array.from({ length: 4 }).map((_, i) => (
                  <TableRow key={i} className="hover:bg-transparent">
                    <TableCell colSpan={4}>
                      <div className="h-5 animate-pulse rounded bg-muted" />
                    </TableCell>
                  </TableRow>
                ))}

              {sorted.map((a) => {
                const sub = a.submission;
                const hasScore = sub && sub.score !== null;
                const pct = hasScore ? Math.round(((sub.score ?? 0) / a.max_score) * 100) : null;

                return (
                  <TableRow key={a.id} className={cn(selected?.id === a.id && "bg-muted/60")}>
                    <TableCell>
                      <p className="font-medium">{a.title}</p>
                      <p className="font-mono text-[11px] text-muted-foreground">{a.course?.title}</p>
                    </TableCell>
                    <TableCell className="font-mono text-xs tabular-nums">
                      <p>{formatDateTime(a.deadline)}</p>
                      <p className={cn("text-muted-foreground", !isOpen(a) && "opacity-60")}>
                        {relativeTime(a.deadline, now)}
                      </p>
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs tabular-nums">
                      {hasScore ? (
                        <div>
                          <p className="font-semibold text-foreground">
                            {sub.score}
                            <span className="font-normal text-muted-foreground">/{a.max_score}</span>
                          </p>
                          <p className="text-[11px] text-muted-foreground">{pct}%</p>
                        </div>
                      ) : sub ? (
                        <div>
                          <p className="text-muted-foreground">—/{a.max_score}</p>
                          <p className="text-[11px] text-muted-foreground">Awaiting grade</p>
                        </div>
                      ) : (
                        <span className="text-muted-foreground">{a.max_score} pts</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      {sub ? (
                        <div className="flex flex-col items-end gap-1">
                          <StatusBadge status={sub.status} syncing={sub.optimistic} />
                          <span className="max-w-44 truncate font-mono text-[11px] text-muted-foreground">
                            {displayFileName(sub.file_url)}
                          </span>
                        </div>
                      ) : isOpen(a) ? (
                        <Button
                          size="sm"
                          variant={selected?.id === a.id ? "default" : "outline"}
                          onClick={() => setSelectedId(a.id)}
                        >
                          Submit
                        </Button>
                      ) : (
                        <span className="font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                          Closed
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </Card>

        <UploadPanel assignment={selected} onSubmit={handleSubmit} />
      </div>
    </AppShell>
  );
}



