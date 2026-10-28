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
  // Controlled by the table, so opening one description closes the other.
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// MetricDescription shows a metric's help truncated to one line; clicking it
// opens the full help with the type, unit and sources the table has no room
// for.
export function MetricDescription({
  metric,
  open,
  onOpenChange,
}: MetricDescriptionProps) {
  if (!metric.help) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }
  return (
    <Popover open={open} onOpenChange={onOpenChange}>
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
        // Returning focus to the trigger would close the next popover the
        // user just opened.
        onCloseAutoFocus={(e) => e.preventDefault()}
      >
        <div className="font-medium break-all">{metric.name}</div>
        <p className="text-sm whitespace-pre-wrap break-words">{metric.help}</p>
        <dl className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Type</dt>
          <dd>
            <MetricTypeTag type={metric.type} />
          </dd>
          {metric.unit && (
            <>
              <dt className="text-muted-foreground">Unit</dt>
              <dd>{metric.unit}</dd>
            </>
          )}
          <dt className="text-muted-foreground">Sources</dt>
          <dd>
            <MetricSources sources={metric.sources} max={10} />
          </dd>
        </dl>
      </PopoverContent>
    </Popover>
  );
}
