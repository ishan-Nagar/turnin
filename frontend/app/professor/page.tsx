"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertCircle, Check, Download, Pencil, X } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Stat } from "@/components/stat";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, messageOf } from "@/lib/api";
import { displayFileName, formatDateTime, relativeTime } from "@/lib/format";
import { useSession } from "@/lib/session";
import type { Assignment, Submission } from "@/lib/types";
import { cn } from "@/lib/utils";

export default function ProfessorDashboard() {
  const session = useSession("professor");
  const [assignments, setAssignments] = useState<Assignment[] | null>(null);
  const [submissions, setSubmissions] = useState<Submission[] | null>(null);
  const [filterId, setFilterId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Inline grading state
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftScore, setDraftScore] = useState<string>("");
  const [savingId, setSavingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [a, s] = await Promise.all([api.listAssignments(), api.listSubmissions()]);
      setAssignments(a);
      setSubmissions(s);
    } catch (e) {
      setError(messageOf(e));
    }
  }, []);

  useEffect(() => {
    if (session) void load();
  }, [session, load]);

  // Live-refresh while any scan is still running in the backend worker.
  const scanning = (submissions ?? []).filter((s) => s.status === "Pending").length;
  useEffect(() => {
    if (!scanning) return;
    const t = setTimeout(() => void load(), 2000);
    return () => clearTimeout(t);
  }, [scanning, submissions, load]);

  const now = Date.now();
  const active = useMemo(
    () => (assignments ?? []).filter((a) => new Date(a.deadline).getTime() > now).length,
    [assignments, now],
  );
  const visible = useMemo(
    () => (submissions ?? []).filter((s) => !filterId || s.assignment_id === filterId),
    [submissions, filterId],
  );
  const filterTitle = assignments?.find((a) => a.id === filterId)?.title;

  function startGrading(s: Submission) {
    setError(null);
    setEditingId(s.id);
    setDraftScore(s.score !== null ? String(s.score) : "");
  }

  async function saveGrade(s: Submission) {
    const maxScore = s.assignment?.max_score ?? 100;
    const parsed = Number(draftScore);
    if (!Number.isInteger(parsed) || parsed < 0 || parsed > maxScore) {
      setError(`Score for “${s.student?.name ?? "submission"}” must be a whole number between 0 and ${maxScore}.`);
      return;
    }

    setError(null);
    setEditingId(null);
    setSavingId(s.id);

    // Snapshot previous state for rollback
    const previous = submissions;

    // 1. Optimistically update the score and flip status to "Graded"
    setSubmissions((curr) =>
      curr
        ? curr.map((row) =>
            row.id === s.id ? { ...row, score: parsed, status: "Graded", optimistic: true } : row,
          )
        : curr,
    );

    try {
      // 2. Persist to backend
      const updated = await api.gradeSubmission(s.id, parsed);
      setSubmissions((curr) =>
        curr ? curr.map((row) => (row.id === s.id ? { ...row, ...updated, optimistic: false } : row)) : curr,
      );
    } catch (e) {
      // 3. Roll back on failure
      setSubmissions(previous);
      setError(`Failed to save grade: ${messageOf(e)}`);
    } finally {
      setSavingId(null);
    }
  }

  async function download(s: Submission) {
    try {
      await api.download(s, displayFileName(s.file_url));
    } catch (e) {
      setError(messageOf(e));
    }
  }

  if (!session) return null;

  const gradedCount = (submissions ?? []).filter((s) => s.status === "Graded").length;
  const ungradedCount = (submissions ?? []).length - gradedCount;

  return (
    <AppShell session={session}>
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">Submissions and grading across your courses.</p>
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat label="Active assignments" value={active} hint={`${assignments?.length ?? 0} total`} />
        <Stat label="Submissions" value={submissions?.length ?? 0} hint={`${ungradedCount} needs grading`} />
        <Stat label="Awaiting scan" value={scanning} hint={scanning ? "auto-refreshing" : "queue empty"} />
        <Stat label="Graded" value={gradedCount} hint="published to students" />
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

      <Card>
        <CardHeader>
          <CardTitle>Assignments</CardTitle>
          <CardDescription>Select a row to filter submissions below.</CardDescription>
        </CardHeader>
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>Assignment</TableHead>
              <TableHead>Deadline</TableHead>
              <TableHead className="text-right">Max</TableHead>
              <TableHead className="text-right">Submissions</TableHead>
              <TableHead className="text-right">State</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {assignments === null && (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={5}>
                  <div className="h-5 animate-pulse rounded bg-muted" />
                </TableCell>
              </TableRow>
            )}
            {assignments?.map((a) => {
              const open = new Date(a.deadline).getTime() > now;
              return (
                <TableRow
                  key={a.id}
                  onClick={() => setFilterId((cur) => (cur === a.id ? null : a.id))}
                  className={cn("cursor-pointer", filterId === a.id && "bg-muted/60")}
                >
                  <TableCell>
                    <p className="font-medium">{a.title}</p>
                    <p className="font-mono text-[11px] text-muted-foreground">{a.course?.title}</p>
                  </TableCell>
                  <TableCell className="font-mono text-xs tabular-nums">
                    <p>{formatDateTime(a.deadline)}</p>
                    <p className="text-muted-foreground">{relativeTime(a.deadline, now)}</p>
                  </TableCell>
                  <TableCell className="text-right font-mono text-xs tabular-nums">{a.max_score}</TableCell>
                  <TableCell className="text-right font-mono text-xs tabular-nums">
                    {a.submission_count ?? 0}
                  </TableCell>
                  <TableCell className="text-right">
                    <Badge className={open ? "text-foreground" : "text-muted-foreground"}>
                      {open ? "Active" : "Closed"}
                    </Badge>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle>Submissions</CardTitle>
            <CardDescription>
              {filterTitle ? `Filtered: ${filterTitle}` : "All assignments"} ·{" "}
              <span className="font-mono">{visible.length}</span> rows · Click any score to grade or update
            </CardDescription>
          </div>
          {filterId && (
            <Button variant="outline" size="sm" onClick={() => setFilterId(null)}>
              <X className="h-3 w-3" /> Clear filter
            </Button>
          )}
        </CardHeader>
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>Student</TableHead>
              <TableHead>Assignment</TableHead>
              <TableHead>Submitted</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Grade</TableHead>
              <TableHead className="text-right">File</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {submissions === null && (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={6}>
                  <div className="h-5 animate-pulse rounded bg-muted" />
                </TableCell>
              </TableRow>
            )}
            {submissions && visible.length === 0 && (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={6} className="py-10 text-center text-sm text-muted-foreground">
                  No submissions yet.
                </TableCell>
              </TableRow>
            )}
            {visible.map((s) => {
              const maxScore = s.assignment?.max_score ?? 100;
              const isEditing = editingId === s.id;

              return (
                <TableRow key={s.id}>
                  <TableCell>
                    <p className="font-medium">{s.student?.name}</p>
                    <p className="font-mono text-[11px] text-muted-foreground">{s.student?.email}</p>
                  </TableCell>
                  <TableCell>
                    <p>{s.assignment?.title}</p>
                    <p className="font-mono text-[11px] text-muted-foreground">
                      {s.assignment?.course?.title}
                    </p>
                  </TableCell>
                  <TableCell className="font-mono text-xs tabular-nums">
                    <p>{formatDateTime(s.submitted_at)}</p>
                    <p className="text-muted-foreground">{relativeTime(s.submitted_at, now)}</p>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={s.status} syncing={s.optimistic || savingId === s.id} />
                  </TableCell>
                  <TableCell className="text-right">
                    {isEditing ? (
                      <form
                        className="inline-flex items-center justify-end gap-1.5"
                        onSubmit={(e) => {
                          e.preventDefault();
                          void saveGrade(s);
                        }}
                      >
                        <Input
                          type="number"
                          min={0}
                          max={maxScore}
                          step={1}
                          autoFocus
                          value={draftScore}
                          onChange={(e) => setDraftScore(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Escape") setEditingId(null);
                          }}
                          className="h-7 w-16 px-2 text-right font-mono text-xs tabular-nums"
                          aria-label={`Score out of ${maxScore}`}
                        />
                        <span className="font-mono text-xs text-muted-foreground">/{maxScore}</span>
                        <Button type="submit" size="sm" className="h-7 px-2" title="Save grade">
                          <Check className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2"
                          onClick={() => setEditingId(null)}
                          title="Cancel"
                        >
                          <X className="h-3.5 w-3.5" />
                        </Button>
                      </form>
                    ) : s.score !== null ? (
                      <button
                        type="button"
                        onClick={() => startGrading(s)}
                        className="group inline-flex items-center gap-1.5 rounded px-2 py-1 font-mono text-xs tabular-nums transition-colors hover:bg-muted"
                        title="Click to update grade"
                      >
                        <span className="font-semibold">{s.score}</span>
                        <span className="text-muted-foreground">/{maxScore}</span>
                        <Pencil className="h-3 w-3 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                      </button>
                    ) : (
                      <Button variant="outline" size="sm" onClick={() => startGrading(s)}>
                        Grade
                      </Button>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => void download(s)}
                      title={displayFileName(s.file_url)}
                    >
                      <Download className="h-3.5 w-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </Card>
    </AppShell>
  );
}
