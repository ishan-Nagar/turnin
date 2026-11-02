import { cn } from "@/lib/utils";

type DivProps = React.HTMLAttributes<HTMLDivElement>;

export const Card = ({ className, ...p }: DivProps) => (
  <div className={cn("rounded-lg border bg-card text-card-foreground", className)} {...p} />
);
export const CardHeader = ({ className, ...p }: DivProps) => (
  <div className={cn("flex flex-col gap-1 border-b p-4", className)} {...p} />
);
export const CardTitle = ({ className, ...p }: React.HTMLAttributes<HTMLHeadingElement>) => (
  <h2 className={cn("text-sm font-semibold tracking-tight", className)} {...p} />
);
export const CardDescription = ({ className, ...p }: React.HTMLAttributes<HTMLParagraphElement>) => (
  <p className={cn("text-xs text-muted-foreground", className)} {...p} />
);
export const CardContent = ({ className, ...p }: DivProps) => <div className={cn("p-4", className)} {...p} />;
