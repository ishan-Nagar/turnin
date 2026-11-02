import { Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

/** Restrained semantic dots on a monochrome base. */
export function StatusBadge({ status, syncing }: { status: string; syncing?: boolean }) {
  const pending = status === "Pending";
  const graded = status === "Graded";

  return (
    <Badge className={cn(pending ? "text-muted-foreground" : "text-foreground")}>
      {syncing ? (
        <Loader2 className="h-3 w-3 animate-spin" aria-hidden />
      ) : (
        <span
          aria-hidden
          className={cn(
            "h-1.5 w-1.5 rounded-full",
            pending && "animate-pulse bg-amber-400",
            graded && "bg-foreground",
            !pending && !graded && "bg-emerald-500",
          )}
        />
      )}
      {syncing ? "Syncing…" : status}
    </Badge>
  );
}
