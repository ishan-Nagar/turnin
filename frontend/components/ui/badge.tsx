import { cn } from "@/lib/utils";

export function Badge({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn("inline-flex items-center gap-1.5 rounded border px-2 py-0.5 font-mono text-[11px] leading-4", className)}
      {...props}
    />
  );
}

