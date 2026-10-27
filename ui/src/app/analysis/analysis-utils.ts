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

// AnalysisFilters narrows the analysis table: text filters match a
// dimension's value case-insensitively, number filters keep rows at or
// above them. Empty means no filter.
export interface AnalysisFilters {
  job: string;
  service: string;
  source: string;
  minTotal: string;
  minUnused: string;
  minSeries: string;
  minUnusedSeries: string;
  minPct: string;
}

export const NO_FILTERS: AnalysisFilters = {
  job: "",
  service: "",
  source: "",
  minTotal: "",
  minUnused: "",
  minSeries: "",
  minUnusedSeries: "",
  minPct: "",
};

export function hasFilters(f: AnalysisFilters): boolean {
  return Object.values(f).some((v) => v.trim() !== "");
}

export function filterRows<T extends UnusedAnalysisRow & { pct: number }>(
  rows: T[],
  f: AnalysisFilters,
): T[] {
  const text = (value: string | undefined, needle: string) =>
    needle.trim() === "" ||
    (value ?? "").toLowerCase().includes(needle.trim().toLowerCase());
  const atLeast = (value: number, min: string) => {
    if (min.trim() === "") return true;
    const n = Number(min);
    return Number.isNaN(n) || value >= n;
  };
  return rows.filter(
    (r) =>
      text(r.job, f.job) &&
      text(r.service, f.service) &&
      text(r.source, f.source) &&
      atLeast(r.total, f.minTotal) &&
      atLeast(r.unused, f.minUnused) &&
      atLeast(r.series, f.minSeries) &&
      atLeast(r.unusedSeries, f.minUnusedSeries) &&
      atLeast(r.pct, f.minPct),
  );
}
