import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export interface QueryOriginsData {
  executions?: number;
  dashboardRuns?: number;
  alertRuns?: number;
  dashboards?: string[];
  rules?: string[];
  clients?: string[];
}

interface QueryOriginsProps {
  origins: QueryOriginsData;
  // Controlled by the table, so opening one list closes the other.
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// QueryOrigins tells who ran a query - dashboards, alert rules, other
// clients - with how many runs each kind; the full lists open on click.
export function QueryOrigins({
  origins,
  open,
  onOpenChange,
}: QueryOriginsProps) {
  const dashboards = origins.dashboardRuns ?? 0;
  const alerts = origins.alertRuns ?? 0;
  const other = Math.max(0, (origins.executions ?? 0) - dashboards - alerts);
  if (dashboards + alerts + other === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  const sections = [
    { title: "Dashboards", runs: dashboards, names: origins.dashboards },
    { title: "Alert rules", runs: alerts, names: origins.rules },
    { title: "Other clients", runs: other, names: origins.clients },
  ].filter((s) => s.runs > 0);

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Query origins"
          className="flex flex-wrap items-center gap-1 cursor-pointer text-left"
          onClick={(e) => e.stopPropagation()}
        >
          {dashboards > 0 && (
            <Badge variant="outline">Dashboards {dashboards}</Badge>
          )}
          {alerts > 0 && <Badge variant="outline">Alerts {alerts}</Badge>}
          {other > 0 && <Badge variant="secondary">Other {other}</Badge>}
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-96 max-h-96 overflow-y-auto space-y-3"
        onClick={(e) => e.stopPropagation()}
        // Returning focus to the trigger would close the next popover the
        // user just opened.
        onCloseAutoFocus={(e) => e.preventDefault()}
      >
        {sections.map((s) => (
          <div key={s.title}>
            <div className="mb-1 text-sm font-medium">
              {s.title}{" "}
              <span className="text-muted-foreground">({s.runs} runs)</span>
            </div>
            {s.names && s.names.length > 0 ? (
              <ul className="space-y-0.5 text-sm">
                {s.names.map((n) => (
                  <li key={n} className="break-all">
                    {n}
                  </li>
                ))}
              </ul>
            ) : (
              <div className="text-sm text-muted-foreground">
                No name in the headers
              </div>
            )}
          </div>
        ))}
      </PopoverContent>
    </Popover>
  );
}
