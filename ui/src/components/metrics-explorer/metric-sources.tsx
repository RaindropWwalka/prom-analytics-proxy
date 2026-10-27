import { Badge } from "@/components/ui/badge";

interface MetricSourcesProps {
  sources?: string[];
  // How many sources to show before collapsing the rest into "+N".
  max?: number;
}

// MetricSources lists the inventory metadata sources reporting a metric.
// Every source stays readable from the cell's tooltip once the list is
// collapsed.
export function MetricSources({ sources, max = 2 }: MetricSourcesProps) {
  if (!sources || sources.length === 0) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }
  const shown = sources.slice(0, max);
  const hidden = sources.length - shown.length;
  return (
    <div
      className="flex flex-wrap items-center gap-1"
      title={sources.join("\n")}
    >
      {shown.map((source) => (
        <Badge
          key={source}
          variant="outline"
          className="max-w-[140px] truncate"
        >
          {source}
        </Badge>
      ))}
      {hidden > 0 && <Badge variant="secondary">+{hidden}</Badge>}
    </div>
  );
}
