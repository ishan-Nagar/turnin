export function Stat({ label, value, hint }: { label: string; value: number | string; hint?: string }) {
  return (
    <div className="rounded-lg border p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-2 font-mono text-2xl font-semibold tabular-nums">{value}</p>
      {hint && <p className="mt-1 font-mono text-[11px] text-muted-foreground">{hint}</p>}
    </div>
  );
}
