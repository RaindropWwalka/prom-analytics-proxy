import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { executionOrigin } from "./execution-origin-utils";

export function ExecutionOrigin({
  headers,
}: {
  headers?: Record<string, string>;
}) {
  const origin = executionOrigin(headers);
  switch (origin.kind) {
    case "alert":
      return (
        <div className="flex min-w-0 items-center gap-2">
          <Badge variant="outline">Alert</Badge>
          <span
            className="truncate"
            title={[origin.folder, origin.rule].filter(Boolean).join(" / ")}
          >
            {origin.rule ?? "—"}
          </span>
        </div>
      );
    case "dashboard":
      return (
        <div className="flex min-w-0 items-center gap-2">
          <Badge variant="outline">Dashboard</Badge>
          <span
            className="truncate"
            title={[origin.dashboard, origin.panel].filter(Boolean).join(" / ")}
          >
            {origin.dashboard}
            {origin.panel && (
              <span className="text-muted-foreground"> · {origin.panel}</span>
            )}
          </span>
        </div>
      );
    case "client":
      return (
        <span
          className="block truncate font-mono text-xs"
          title={origin.client}
        >
          {origin.client}
        </span>
      );
    default:
      return <span className="text-muted-foreground">—</span>;
  }
}

// HeadersPopover lists every header stored with an execution.
export function HeadersPopover({
  headers,
}: {
  headers?: Record<string, string>;
}) {
  const entries = Object.entries(headers ?? {})
    .filter(([k, v]) => k.trim() !== "" && v.trim() !== "")
    .sort(([a], [b]) => a.localeCompare(b));
  if (entries.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="text-sm underline-offset-2 hover:underline cursor-pointer"
          onClick={(e) => e.stopPropagation()}
        >
          {entries.length} {entries.length === 1 ? "header" : "headers"}
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[28rem] max-w-[90vw]">
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 font-mono text-xs">
          {entries.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="text-muted-foreground">{k}</dt>
              <dd className="break-all">{v}</dd>
            </div>
          ))}
        </dl>
      </PopoverContent>
    </Popover>
  );
}
