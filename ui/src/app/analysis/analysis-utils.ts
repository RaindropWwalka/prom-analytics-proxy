import type { UnusedAnalysisGroupBy, UnusedAnalysisRow } from "@/api/metrics";
import { ROUTES } from "@/lib/routes";

export type Dimension = "job" | "service" | "source";

// DIMENSIONS is every grouping dimension, in the order the API takes them.
export const DIMENSIONS: Dimension[] = ["job", "service", "source"];

const DIMENSION_TITLES: Record<Dimension, string> = {
  job: "Job",
  service: "Service",
  source: "Source",
};

// groupByOf turns the chosen dimensions into the API's groupBy, or null
// when none is chosen.
export function groupByOf(dims: Dimension[]): UnusedAnalysisGroupBy | null {
  const ordered = DIMENSIONS.filter((d) => dims.includes(d));
  return ordered.length ? (ordered.join(",") as UnusedAnalysisGroupBy) : null;
}

function dimensionsOf(groupBy: UnusedAnalysisGroupBy): Dimension[] {
  return groupBy.split(",") as Dimension[];
}

// groupLabel names a row by what the analysis is grouped by:
// "job / service @ source", with whichever parts the grouping has.
export function groupLabel(
  row: UnusedAnalysisRow,
  groupBy: UnusedAnalysisGroupBy,
): string {
  const dims = dimensionsOf(groupBy);
  const dash = (v?: string) => v || "—";
  const owner = dims
    .filter((d) => d !== "source")
    .map((d) => dash(row[d]))
    .join(" / ");
  if (!dims.includes("source")) return owner;
  return owner ? `${owner} @ ${dash(row.source)}` : dash(row.source);
}

// groupTitle names the grouping for headers, in groupLabel's shape.
export function groupTitle(groupBy: UnusedAnalysisGroupBy): string {
  const dims = dimensionsOf(groupBy);
  const owner = dims
    .filter((d) => d !== "source")
    .map((d) => DIMENSION_TITLES[d])
    .join(" / ");
  if (!dims.includes("source")) return owner;
  return owner ? `${owner} @ Source` : "Source";
}

export function dimensionTitle(d: Dimension): string {
  return DIMENSION_TITLES[d];
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
