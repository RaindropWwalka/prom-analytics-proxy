import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

interface MetricSourcesProps {
  sources?: string[];
  // How many sources to show before collapsing the rest into "+N".
  max?: number;
  // Given, the badges open the full list on click; the table controls it so
  // only one list or description is open at a time.
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// MetricSources lists the inventory metadata sources reporting a metric.
export function MetricSources({
  sources,
  max = 2,
  open,
  onOpenChange,
}: MetricSourcesProps) {
  if (!sources || sources.length === 0) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }
  const shown = sources.slice(0, max);
  const hidden = sources.length - shown.length;
  const badges = (
    <>
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
    </>
  );

  if (!onOpenChange) {
    return (
      <div
        className="flex flex-wrap items-center gap-1"
        title={sources.join("\n")}
      >
        {badges}
      </div>
    );
  }

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={`${sources.length} sources`}
          className="flex flex-wrap items-center gap-1 cursor-pointer text-left"
          // The row itself opens the metric's page.
          onClick={(e) => e.stopPropagation()}
        >
          {badges}
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-80 max-h-80 overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
        // Returning focus to the trigger would close the next popover the
        // user just opened.
        onCloseAutoFocus={(e) => e.preventDefault()}
      >
        <div className="mb-2 text-sm text-muted-foreground">
          Reported by {sources.length}{" "}
          {sources.length === 1 ? "source" : "sources"}
        </div>
        <ul className="space-y-1 text-sm">
          {sources.map((source) => (
            <li key={source} className="break-all">
              {source}
            </li>
          ))}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
