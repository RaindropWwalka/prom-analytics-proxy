import type { UnusedAnalysisGroupBy, UnusedAnalysisRow } from "@/api/metrics";
import { ROUTES } from "@/lib/routes";

// groupLabel names a row by what the analysis is grouped by.
export function groupLabel(
  row: UnusedAnalysisRow,
  groupBy: UnusedAnalysisGroupBy,
): string {
  switch (groupBy) {
    case "job":
      return row.job || "—";
    case "source":
      return row.source || "—";
    default:
      return `${row.job || "—"} @ ${row.source || "—"}`;
  }
}

export function unusedPercent(row: UnusedAnalysisRow): number {
  return row.total > 0 ? Math.round((row.unused / row.total) * 100) : 0;
}

// catalogLink opens the catalog on a group's unused metrics.
export function catalogLink(row: UnusedAnalysisRow): string {
  const params = new URLSearchParams({ usage: "unused" });
  if (row.job) params.set("job", row.job);
  if (row.source) params.set("source", row.source);
  return `${ROUTES.METRICS_EXPLORER}?${params}`;
}

export function totals(rows: UnusedAnalysisRow[]) {
  return rows.reduce(
    (acc, r) => ({ total: acc.total + r.total, unused: acc.unused + r.unused }),
    { total: 0, unused: 0 },
  );
}
