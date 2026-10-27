import type { UnusedAnalysisGroupBy, UnusedAnalysisRow } from "@/api/metrics";
import { ROUTES } from "@/lib/routes";

// groupLabel names a row by what the analysis is grouped by.
export function groupLabel(
  row: UnusedAnalysisRow,
  groupBy: UnusedAnalysisGroupBy,
): string {
  const dash = (v?: string) => v || "—";
  switch (groupBy) {
    case "job":
      return dash(row.job);
    case "service":
      return dash(row.service);
    case "source":
      return dash(row.source);
    case "job,source":
      return `${dash(row.job)} @ ${dash(row.source)}`;
    case "service,source":
      return `${dash(row.service)} @ ${dash(row.source)}`;
  }
}

// groupTitle names the grouping for headers.
export function groupTitle(groupBy: UnusedAnalysisGroupBy): string {
  return {
    job: "Job",
    service: "Service",
    source: "Source",
    "job,source": "Job @ Source",
    "service,source": "Service @ Source",
  }[groupBy];
}

export function unusedPercent(row: UnusedAnalysisRow): number {
  return row.total > 0 ? Math.round((row.unused / row.total) * 100) : 0;
}

// catalogLink opens the catalog on a group's unused metrics.
export function catalogLink(row: UnusedAnalysisRow): string {
  const params = new URLSearchParams({ usage: "unused" });
  if (row.job) params.set("job", row.job);
  if (row.service) params.set("service", row.service);
  if (row.source) params.set("source", row.source);
  return `${ROUTES.METRICS_EXPLORER}?${params}`;
}

export function totals(rows: UnusedAnalysisRow[]) {
  return rows.reduce(
    (acc, r) => ({ total: acc.total + r.total, unused: acc.unused + r.unused }),
    { total: 0, unused: 0 },
  );
}
