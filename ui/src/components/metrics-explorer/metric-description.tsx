import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { MetricTypeTag } from "@/components/metrics-explorer/metric-type-tag";
import { MetricSources } from "@/components/metrics-explorer/metric-sources";
import { MetricMetadata } from "@/lib/types";

interface MetricDescriptionProps {
  metric: MetricMetadata;
}

// MetricDescription shows a metric's help truncated to one line; clicking it
// opens the full help with the type, unit and sources the table has no room
// for.
export function MetricDescription({ metric }: MetricDescriptionProps) {
  if (!metric.help) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="block w-full truncate text-left text-sm text-muted-foreground hover:text-foreground cursor-pointer"
          // The row itself opens the metric's page.
          onClick={(e) => e.stopPropagation()}
        >
          {metric.help}
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-96 space-y-3"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="font-medium break-all">{metric.name}</div>
        <p className="text-sm whitespace-pre-wrap break-words">{metric.help}</p>
        <dl className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Type</dt>
          <dd>
            <MetricTypeTag type={metric.type} />
          </dd>
          <dt className="text-muted-foreground">Unit</dt>
          <dd>{metric.unit || "—"}</dd>
          <dt className="text-muted-foreground">Sources</dt>
          <dd>
            <MetricSources sources={metric.sources} max={10} />
          </dd>
        </dl>
      </PopoverContent>
    </Popover>
  );
}
