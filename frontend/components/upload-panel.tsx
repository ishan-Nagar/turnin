"use client";

import { useEffect, useRef, useState } from "react";
import { FileUp, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDateTime } from "@/lib/format";
import type { Assignment } from "@/lib/types";
import { cn } from "@/lib/utils";

const MAX_BYTES = 10 * 1024 * 1024;
const ALLOWED = [".pdf", ".doc", ".docx", ".zip", ".txt", ".md", ".py", ".go", ".java", ".cpp", ".ipynb"];

interface Props {
  assignment: Assignment | null;
  /** Fire-and-forget: the parent applies the optimistic update synchronously. */
  onSubmit: (assignment: Assignment, file: File) => void;
}

export function UploadPanel({ assignment, onSubmit }: Props) {
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { setFile(null); setError(null); }, [assignment?.id]);

  function pick(f: File | undefined) {
    if (!f) return;
    const ext = f.name.slice(f.name.lastIndexOf(".")).toLowerCase();
    if (!ALLOWED.includes(ext)) return setError(`Unsupported type "${ext}". Allowed: ${ALLOWED.join(" ")}`);
    if (f.size === 0 || f.size > MAX_BYTES) return setError("File must be between 1 byte and 10 MB.");
    setError(null);
    setFile(f);
  }

  if (!assignment) {
    return (
      <Card className="sticky top-20">
        <CardContent className="flex min-h-48 flex-col items-center justify-center gap-2 text-center">
          <FileUp className="h-5 w-5 text-muted-foreground" />
          <p className="text-sm">No assignment selected</p>
          <p className="text-xs text-muted-foreground">Choose “Submit” on an open assignment.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="sticky top-20">
      <CardHeader>
        <CardTitle>Submit work</CardTitle>
        <CardDescription>{assignment.title}</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (file) onSubmit(assignment, file);
          }}
        >
          <div
            role="button"
            tabIndex={0}
            onClick={() => inputRef.current?.click()}
            onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && inputRef.current?.click()}
            onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
            onDragLeave={() => setDragging(false)}
            onDrop={(e) => { e.preventDefault(); setDragging(false); pick(e.dataTransfer.files[0]); }}
            className={cn(
              "flex cursor-pointer flex-col items-center justify-center gap-1 rounded-md border border-dashed px-4 py-8 text-center transition-colors",
              "hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
              dragging && "border-foreground bg-muted/60",
            )}
          >
            <FileUp className="h-5 w-5 text-muted-foreground" />
            <p className="text-sm">Drop a file or click to browse</p>
            <p className="font-mono text-[11px] text-muted-foreground">max 10 MB · {ALLOWED.slice(0, 4).join(" ")} …</p>
            <input ref={inputRef} type="file" hidden onChange={(e) => pick(e.target.files?.[0])} />
          </div>

          {file && (
            <div className="flex items-center justify-between rounded-md border px-3 py-2">
              <div className="min-w-0">
                <p className="truncate font-mono text-xs">{file.name}</p>
                <p className="font-mono text-[11px] text-muted-foreground">{(file.size / 1024).toFixed(1)} KB</p>
              </div>
              <button type="button" aria-label="Remove file" onClick={() => setFile(null)}>
                <X className="h-4 w-4 text-muted-foreground hover:text-foreground" />
              </button>
            </div>
          )}

          {error && <p role="alert" className="text-xs text-destructive">{error}</p>}

          <div className="flex items-center justify-between">
            <p className="font-mono text-[11px] text-muted-foreground">Due {formatDateTime(assignment.deadline)}</p>
            <Button type="submit" disabled={!file}>Submit</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

