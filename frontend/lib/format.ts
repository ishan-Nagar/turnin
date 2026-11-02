const dateTime = new Intl.DateTimeFormat("en-IN", {
  day: "2-digit", month: "short", year: "numeric",
  hour: "2-digit", minute: "2-digit", hour12: false,
});

export const formatDateTime = (iso: string) => dateTime.format(new Date(iso));

/** "in 2d 4h" / "35m ago" */
export function relativeTime(iso: string, now = Date.now()): string {
  const diff = new Date(iso).getTime() - now;
  const mins = Math.floor(Math.abs(diff) / 60_000);
  const hours = Math.floor(mins / 60);
  const days = Math.floor(hours / 24);
  const span = days > 0 ? `${days}d ${hours % 24}h` : hours > 0 ? `${hours}h ${mins % 60}m` : `${Math.max(mins, 1)}m`;
  return diff >= 0 ? `in ${span}` : `${span} ago`;
}

/** Strips the server's "<uuid>_" storage prefix and any directory. */
export const displayFileName = (fileUrl: string) =>
  (fileUrl.split("/").pop() ?? fileUrl).replace(/^[0-9a-f-]{36}_/, "");

